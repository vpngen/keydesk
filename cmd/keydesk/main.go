package main

import (
	"context"
	"crypto/tls"
	"encoding/base32"
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
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/coreos/go-systemd/activation"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/vapnapi"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
)

//go:generate swagger generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user

// TokenLifeTime - token time to life.
const TokenLifeTime = 3600

// Default web config.
const (
	DefaultWebDir    = "/var/www"
	DefaultIndexFile = "index.html"
	DefaultCertDir   = "/etc/vgcert"
	TLSCertFilename  = "vpn.works.crt"
	TLSKeyFilename   = "vpn.works.key"
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
	chunked, pcors, listeners, addr, BrigadeID, etcDir, webDir, dbDir, statDir, certDir, name, person, replace, err := parseArgs()
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
		StatFilename:    filepath.Join(statDir, fmt.Sprintf(storage.StatFilename, BrigadeID)),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:              keydesk.MaxUsers,
			MonthlyQuotaRemaining: keydesk.MonthlyQuotaRemaining,
			ActivityPeriod:        keydesk.ActivityPeriod,
		},
	}
	if err := db.SelfCheck(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	fmt.Fprintf(os.Stderr, "Etc: %s\n", etcDir)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", dbDir)
	fmt.Fprintf(os.Stderr, "Stat Dir: %s\n", statDir)
	fmt.Fprintf(os.Stderr, "Command address:port: %s\n", addr)

	// Just create brigadier.
	if name != "" {
		if err := createBrigadier(db, chunked, name, person, replace, &routerPublicKey, &shufflerPublicKey); err != nil {
			log.Fatalf("Can't create brigadier: %s\n", err)
		}

		return
	}

	fmt.Fprintf(os.Stderr, "Cert Dir: %s\n", certDir)
	fmt.Fprintf(os.Stderr, "Web files: %s\n", webDir)
	fmt.Fprintf(os.Stderr, "Permessive CORS: %t\n", pcors)
	fmt.Fprintf(os.Stderr, "Starting %s keydesk\n", BrigadeID)

	idleTimer := time.NewTimer(keydesk.MaxIdlePeriod)
	handler := initSwaggerAPI(db, BrigadeID, &routerPublicKey, &shufflerPublicKey, pcors, webDir, idleTimer)

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Handler:     handler,
		IdleTimeout: 60 * time.Minute,
	}

	var serverTLS *http.Server

	if len(listeners) == 2 {
		// openssl req -x509 -nodes -days 10000 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -subj '/CN=vpn.works/O=VPNGen/C=LT/ST=Vilniaus Apskritis/L=Vilnius' -keyout vpn.works.key -out vpn.works.crt
		switch cert, err := tls.LoadX509KeyPair(
			filepath.Join(certDir, TLSCertFilename),
			filepath.Join(certDir, TLSKeyFilename),
		); err {
		case nil:
			serverTLS = &http.Server{
				TLSConfig:   &tls.Config{Certificates: []tls.Certificate{cert}},
				Handler:     handler,
				IdleTimeout: 60 * time.Minute,
			}
		default:
			fmt.Fprintf(os.Stderr, "Skip TLS: can't open cert/key pair: %s\n", err)
		}
	}

	go func() {
		select {
		case <-quit:
			fmt.Fprintln(os.Stderr, "Quit signal received...")
		case t := <-idleTimer.C:
			fmt.Fprintln(os.Stderr, "Idle timeout exeeded...", t)
		}

		wg := sync.WaitGroup{}

		closeFunc := func(srv *http.Server) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			srv.SetKeepAlivesEnabled(false)
			if err := srv.Shutdown(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Can't gracefully shut down the server: %s\n", err)
			}
		}

		fmt.Fprintln(os.Stderr, "Server is shutting down")
		wg.Add(1)
		go closeFunc(server)

		if serverTLS != nil {
			fmt.Fprintln(os.Stderr, "Server TLS is shutting down")
			wg.Add(1)
			go closeFunc(serverTLS)
		}

		wg.Wait()

		close(done)
	}()

	fmt.Fprintf(os.Stderr, "Listen HTTP: %s\n", listeners[0].Addr().String())
	if serverTLS != nil {
		fmt.Fprintf(os.Stderr, "Listen HTTPS: %s\n", listeners[1].Addr().String())
	}

	// Start accepting connections.
	go func() {
		if err := server.Serve(listeners[0]); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Can't serve: %s\n", err)
		}
	}()

	if serverTLS != nil && len(listeners) == 2 {
		// Start accepting connections.
		go func() {
			if err := serverTLS.ServeTLS(listeners[1], "", ""); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Can't serve TLS: %s\n", err)
			}
		}()
	}

	// Wait for existing connections before exiting.
	<-done
}

func parseArgs() (bool, bool, []net.Listener, netip.AddrPort, string, string, string, string, string, string, string, namesgenerator.Person, bool, error) {
	var (
		id                              string
		etcdir, dbdir, statdir, certdir string
		person                          namesgenerator.Person
		addrPort                        netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("cannot define user: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("cur dir: %w", err)
	}

	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("cur dir: %w", err)
	}

	webDir := flag.String("w", DefaultWebDir, "Dir for web files.")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	statDir := flag.String("s", "", "Dir for statistic files (for test). Default: "+storage.DefaultStatDir+"/<BrigadeID>")
	certDir := flag.String("e", "", "Dir for TLS certificate and key (for test). Default: "+DefaultCertDir)
	pcors := flag.Bool("cors", false, "Turn on permessive CORS (for test)")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	listenAddr := flag.String("l", "", "Listen addr:port (http and https separate with commas)")

	brigadierName := flag.String("name", "", "brigadierName :: base64")
	personName := flag.String("person", "", "personName :: base64")
	personDesc := flag.String("desc", "", "personDesc :: base64")
	personURL := flag.String("url", "", "personURL :: base64")
	replaceBrigadier := flag.Bool("r", false, "Replace brigadier config")

	addr := flag.String("a", vapnapi.TemplatedAddrPort, "API endpoint address:port")

	chunked := flag.Bool("ch", false, "chunked output")

	flag.Parse()

	if *webDir == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrStaticDirEmpty
	}

	webdir, err := filepath.Abs(*webDir)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("web dir: %w", err)
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("etcdir dir: %w", err)
		}
	}

	if *statDir != "" {
		statdir, err = filepath.Abs(*statDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("statdir dir: %w", err)
		}
	}

	if *certDir != "" {
		certdir, err = filepath.Abs(*certDir)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("statdir dir: %w", err)
		}
	}

	switch *brigadeID {
	case "", sysUser.Username:
		id = sysUser.Username

		if *filedbDir == "" {
			dbdir = filepath.Join(storage.DefaultHomeDir, id)
		}

		if *etcDir == "" {
			etcdir = keydesk.DefaultEtcDir
		}

		if *statDir != "" {
			statdir = filepath.Join(storage.DefaultStatDir, id)
		}

		if *certDir != "" {
			certdir = DefaultCertDir
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

		if *certDir == "" {
			certdir = cwd
		}
	}

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("id uuid: %s: %w", id, err)
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("api addr: %w", err)
		}
	}

	if *brigadierName == "" {
		var listeners []net.Listener

		switch *listenAddr {
		case "":
			listeners, err = activation.Listeners()
			if err != nil {
				return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("cannot retrieve listeners: %w", err)
			}

			if len(listeners) != 1 && len(listeners) != 2 {
				return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("unexpected number of socket activation (%d != 1|2)",
					len(listeners))
			}
		default:
			for _, laddr := range strings.Split(*listenAddr, ",") {
				l, err := net.Listen("tcp", laddr)
				if err != nil {
					return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("cannot listen: %w", err)
				}

				listeners = append(listeners, l)
			}

			if len(listeners) != 1 && len(listeners) != 2 {
				return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("unexpected number of litening (%d != 1|2)",
					len(listeners))
			}
		}

		return *chunked, *pcors, listeners, addrPort, id, etcdir, webdir, dbdir, statdir, certdir, "", person, false, nil
	}

	// brigadierName must be not empty and must be a valid UTF8 string
	buf, err := base64.StdEncoding.DecodeString(*brigadierName)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("brigadier name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrInvalidBrigadierName
	}

	name := string(buf)

	// personName must be not empty and must be a valid UTF8 string
	if *personName == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrEmptyPersonName
	}

	buf, err = base64.StdEncoding.DecodeString(*personName)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("person name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrInvalidPersonName
	}

	person.Name = string(buf)

	// personDesc must be not empty and must be a valid UTF8 string
	if *personDesc == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrEmptyPersonDesc
	}

	buf, err = base64.StdEncoding.DecodeString(*personDesc)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("person desc: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrInvalidPersonDesc
	}

	person.Desc = string(buf)

	// personURL must be not empty and must be a valid UTF8 string
	if *personURL == "" {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrEmptyPersonURL
	}

	buf, err = base64.StdEncoding.DecodeString(*personURL)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("person url: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, ErrInvalidPersonURL
	}

	u := string(buf)

	_, err = url.Parse(u)
	if err != nil {
		return false, false, nil, addrPort, "", "", "", "", "", "", "", person, false, fmt.Errorf("parse person url: %w", err)
	}

	person.URL = u

	return *chunked, *pcors, nil, addrPort, id, etcdir, webdir, dbdir, statdir, certdir, name, person, *replaceBrigadier, nil
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

func createBrigadier(db *storage.BrigadeStorage,
	chunked bool,
	name string,
	person namesgenerator.Person,
	replace bool,
	routerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
) error {
	var w io.WriteCloser

	wgconf, filename, err := keydesk.AddBrigadier(db, name, person, replace, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return fmt.Errorf("add brigadier: %w", err)
	}

	switch chunked {
	case true:
		w = httputil.NewChunkedWriter(os.Stdout)
		defer w.Close()
	default:
		w = os.Stdout
	}

	if _, err := fmt.Fprintln(w, filename); err != nil {
		return fmt.Errorf("print filename: %w", err)
	}

	if _, err := fmt.Fprintln(w, wgconf); err != nil {
		return fmt.Errorf("pring wgconf: %w", err)
	}

	return nil
}

func initSwaggerAPI(db *storage.BrigadeStorage,
	brigadeID string,
	routerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	pcors bool,
	webDir string,
	idleTimer *time.Timer,
) http.Handler {
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

	api.BearerAuth = keydesk.ValidateBearer(brigadeID)
	api.PostTokenHandler = operations.PostTokenHandlerFunc(keydesk.CreateToken(brigadeID, TokenLifeTime))
	api.PostUserHandler = operations.PostUserHandlerFunc(func(params operations.PostUserParams, principal interface{}) middleware.Responder {
		return keydesk.AddUser(db, params, principal, routerPublicKey, shufflerPublicKey)
	})
	api.DeleteUserUserIDHandler = operations.DeleteUserUserIDHandlerFunc(func(params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
		return keydesk.DelUserUserID(db, params, principal)
	})
	api.GetUserHandler = operations.GetUserHandlerFunc(func(params operations.GetUserParams, principal interface{}) middleware.Responder {
		return keydesk.GetUsers(db, params, principal)
	})

	switch pcors {
	case true:
		return cors.AllowAll().Handler(
			uiMiddleware(api.Serve(nil), webDir, idleTimer),
		)
	default:
		return uiMiddleware(api.Serve(nil), webDir, idleTimer)
	}
}

func uiMiddleware(handler http.Handler, dir string, idleTimer *time.Timer) http.Handler {
	staticFS := http.Dir(dir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu := sync.Mutex{}

		mu.Lock()

		if !idleTimer.Stop() {
			<-idleTimer.C
		}

		idleTimer.Reset(keydesk.MaxIdlePeriod)

		mu.Unlock()

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
