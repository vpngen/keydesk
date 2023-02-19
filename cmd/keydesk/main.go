package main

import (
	"context"
	"encoding/base64"
	goerrors "errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/vpngen/keydesk/epapi"
	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/rs/cors"
)

//go:generate swagger generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user

// TokenLifeTime - token time to life.
const TokenLifeTime = 3600

// Default web config.
const (
	DefaultWebDir    = "/var/www"
	DefaultIndexFile = "index.html"
)

// ErrStaticDirEmpty - no static dir name.
var ErrStaticDirEmpty = goerrors.New("empty static dirname")

// Args errors.
var (
	ErrInvalidBrigadierName = goerrors.New("invalid brigadier name")
	ErrEmptyPersonName      = goerrors.New("empty person name")
	ErrEmptyPersonDesc      = goerrors.New("empty person desc")
	ErrEmptyPersonURL       = goerrors.New("empty person url")
	ErrInvalidPersonName    = goerrors.New("invalid person name")
	ErrInvalidPersonDesc    = goerrors.New("invalid person desc")
	ErrInvalidPersonURL     = goerrors.New("invalid person url")
)

func main() {
	var handler http.Handler

	chunked, pcors, listen, addr, BrigadeID, etcDir, webDir, dbDir, statDir, name, person, replace, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	db := &storage.BrigadeStorage{
		BrigadeID:       BrigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		StatsFilename:   filepath.Join(statDir, fmt.Sprintf(storage.StatFilename, BrigadeID)),
		APIAddrPort:     addr,
	}

	// Just create brigadier.
	if name != "" {
		var w io.WriteCloser

		wgconf, filename, err := keydesk.AddBrigadier(db, name, person, replace, &routerPublicKey, &shufflerPublicKey)
		if err != nil {
			log.Fatalf("Can't create brigadier: %s\n", err)
		}

		switch chunked {
		case true:
			w = httputil.NewChunkedWriter(os.Stdout)
			defer w.Close()
		default:
			w = os.Stdout
		}

		_, err = fmt.Fprintln(w, filename)
		if err != nil {
			log.Fatalf("Can't print filename: %s\n", err)
		}

		_, err = fmt.Fprintln(w, wgconf)
		if err != nil {
			log.Fatalf("Can't print wgconf: %s\n", err)
		}

		return
	}

	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	// create new service API
	api := operations.NewUserAPI(swaggerSpec)

	api.ServeError = errors.ServeError

	api.UseSwaggerUI()

	api.JSONConsumer = runtime.JSONConsumer()

	api.BinProducer = runtime.ByteStreamProducer()
	api.JSONProducer = runtime.JSONProducer()

	api.BearerAuth = keydesk.ValidateBearer(BrigadeID)
	api.PostTokenHandler = operations.PostTokenHandlerFunc(keydesk.CreateToken(BrigadeID, TokenLifeTime))
	api.PostUserHandler = operations.PostUserHandlerFunc(func(params operations.PostUserParams, principal interface{}) middleware.Responder {
		return keydesk.AddUser(db, params, principal, &routerPublicKey, &shufflerPublicKey)
	})
	api.DeleteUserUserIDHandler = operations.DeleteUserUserIDHandlerFunc(func(params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
		return keydesk.DelUserUserID(db, params, principal)
	})
	api.GetUserHandler = operations.GetUserHandlerFunc(func(params operations.GetUserParams, principal interface{}) middleware.Responder {
		return keydesk.GetUsers(db, params, principal)
	})

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	switch pcors {
	case true:
		handler = cors.AllowAll().Handler(
			uiMiddleware(api.Serve(nil), webDir),
		)
	default:
		handler = uiMiddleware(api.Serve(nil), webDir)
	}

	server := &http.Server{
		Handler:     handler,
		IdleTimeout: 60 * time.Minute,
	}

	go func() {
		<-quit
		log.Println("server is shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Panicf("cannot gracefully shut down the server: %s", err)
		}
		close(done)
	}()

	fmt.Printf("Starting %s keydesk\n", BrigadeID)
	fmt.Printf("Etc: %s\n", etcDir)
	fmt.Printf("DBDir: %s\n", dbDir)
	fmt.Printf("Web files: %s\n", webDir)
	fmt.Printf("Listen: %s\n", listen.Addr().String())
	fmt.Printf("Permessive CORS: %t\n", pcors)

	// Start accepting connections.
	if err := server.Serve(listen); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Can't serve: %s\n", err)
	}

	// Wait for existing connections before exiting.
	<-done
}

func uiMiddleware(handler http.Handler, dir string) http.Handler {
	staticFS := http.Dir(dir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Join(dir, r.URL.Path)
		finfo, err := os.Stat(filename)

		if err == nil && finfo.IsDir() {
			_, err = os.Stat(filepath.Join(filename, DefaultIndexFile))
		}

		if err == nil {
			http.FileServer(staticFS).ServeHTTP(w, r)

			return
		}

		fmt.Fprintf(os.Stderr, "Connect From: %s\n", r.RemoteAddr)

		handler.ServeHTTP(w, r)
	})
}

func parseArgs() (bool, bool, net.Listener, netip.AddrPort, string, string, string, string, string, string, namesgenerator.Person, bool, error) {
	var (
		listen                 net.Listener
		id                     string
		etcdir, dbdir, statdir string
		person                 namesgenerator.Person
		addrPort               netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("cannot define user: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("cur dir: %w", err)
	}

	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("cur dir: %w", err)
	}

	webDir := flag.String("w", DefaultWebDir, "Dir for web files.")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultFileDbDir)
	statDir := flag.String("s", storage.DefaultStatDir, "Dir for stst files (for test)")
	pcors := flag.Bool("cors", false, "Turn on permessive CORS (for test)")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	listenAddr := flag.String("l", "", "Listen addr:port (for test)")

	brigadierName := flag.String("name", "", "brigadierName :: base64")
	personName := flag.String("person", "", "personName :: base64")
	personDesc := flag.String("desc", "", "personDesc :: base64")
	personURL := flag.String("url", "", "personURL :: base64")
	replaceBrigadier := flag.Bool("r", false, "Replace brigadier config")

	addr := flag.String("a", epapi.TemplatedAddrPort, "API endpoint address:port")

	chunked := flag.Bool("ch", false, "chunked output")

	flag.Parse()

	if *webDir == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrStaticDirEmpty
	}

	webdir, err := filepath.Abs(*webDir)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("web dir: %w", err)
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("etcdir dir: %w", err)
		}
	}

	if *statDir != "" {
		statdir, err = filepath.Abs(*statDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("statdir dir: %w", err)
		}
	}

	switch *brigadeID {
	case "":
		id = sysUser.Username

		if *filedbDir == "" {
			dbdir = filepath.Join("home", id)
		}

		if *etcDir == "" {
			etcdir = keydesk.DefaultEtcDir
		}

		if *statDir != "" {
			statdir = storage.DefaultStatDir
		}
	default:
		id = *brigadeID

		if *filedbDir == "" {
			dbdir = cwd
		}

		if *etcDir == "" {
			etcdir = cwd
		}

		if *statDir == "" {
			statdir = cwd
		}
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("api addr: %w", err)
		}
	}

	if *brigadierName == "" {
		switch *listenAddr {
		case "":
			listeners, err := activation.Listeners()
			if err != nil {
				return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("cannot retrieve listeners: %w", err)
			}

			if len(listeners) != 1 {
				return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("unexpected number of socket activation (%d != 1)",
					len(listeners))
			}

			listen = listeners[0]
		default:
			l, err := net.Listen("tcp", *listenAddr)
			if err != nil {
				return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("cannot listen: %w", err)
			}

			listen = l
		}

		return *chunked, *pcors, listen, addrPort, id, etcdir, webdir, dbdir, statdir, "", person, false, nil
	}

	// brigadierName must be not empty and must be a valid UTF8 string
	buf, err := base64.StdEncoding.DecodeString(*brigadierName)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("brigadier name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrInvalidBrigadierName
	}

	name := string(buf)

	// personName must be not empty and must be a valid UTF8 string
	if *personName == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrEmptyPersonName
	}

	buf, err = base64.StdEncoding.DecodeString(*personName)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("person name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrInvalidPersonName
	}

	person.Name = string(buf)

	// personDesc must be not empty and must be a valid UTF8 string
	if *personDesc == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrEmptyPersonDesc
	}

	buf, err = base64.StdEncoding.DecodeString(*personDesc)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("person desc: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrInvalidPersonDesc
	}

	person.Desc = string(buf)

	// personURL must be not empty and must be a valid UTF8 string
	if *personURL == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrEmptyPersonURL
	}

	buf, err = base64.StdEncoding.DecodeString(*personURL)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("person url: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, ErrInvalidPersonURL
	}

	u := string(buf)

	_, err = url.Parse(u)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", person, false, fmt.Errorf("parse person url: %w", err)
	}

	person.URL = u

	return *chunked, *pcors, listen, addrPort, id, etcdir, webdir, dbdir, statdir, name, person, *replaceBrigadier, nil
}

func readPubKeys(path string) ([naclkey.NaclBoxKeyLength]byte, [naclkey.NaclBoxKeyLength]byte, error) {
	empty := [naclkey.NaclBoxKeyLength]byte{}

	routerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, keydesk.RouterPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("router key: %w", err)
	}

	shufflerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, keydesk.ShufflerPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("shuffler key: %w", err)
	}

	return routerPublicKey, shufflerPublicKey, nil
}
