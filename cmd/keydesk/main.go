package main

import (
	"context"
	"crypto/tls"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"flag"
	"fmt"
	"github.com/vpngen/keydesk/internal/stat"
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
	"github.com/vpngen/keydesk/vpnapi"
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
	chunked, jsonOut, pcors, listeners, addr, BrigadeID, etcDir, webDir, dbDir, certDir, name, person, replace, vpnCfgs, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	statsDir := flag.String("s", "", "Dir with brigades statistics. Default: "+storage.DefaultStatsDir+"/<BrigadeID>")

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Etc: %s\n", etcDir)
	_, _ = fmt.Fprintf(os.Stdout, "DBDir: %s\n", dbDir)

	db := &storage.BrigadeStorage{
		BrigadeID:       BrigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	allowedAddress := ""
	calculatedAddrPort, ok := db.CalculatedAPIAddress()
	if ok {
		allowedAddress = calculatedAddrPort.String()
		_, _ = fmt.Fprintf(os.Stdout, "Resqrict requests by address: %s \n", allowedAddress)
	}

	// Just create brigadier.
	if name != "" || replace {
		if err := createBrigadier(db, chunked, jsonOut, name, person, replace, vpnCfgs, &routerPublicKey, &shufflerPublicKey); err != nil {
			log.Fatalf("Can't create brigadier: %s\n", err)
		}

		return
	}

	switch {
	case addr.IsValid() && addr.Addr().IsUnspecified():
		_, _ = fmt.Fprintln(os.Stdout, "Command address:port is COMMON")
	case addr.IsValid():
		_, _ = fmt.Fprintf(os.Stdout, "Command address:port: %s\n", addr)
	default:
		_, _ = fmt.Fprintln(os.Stdout, "Command address:port is for DEBUG")
	}

	_, _ = fmt.Fprintf(os.Stdout, "Cert Dir: %s\n", certDir)
	_, _ = fmt.Fprintf(os.Stdout, "Stat Dir: %s\n", *statsDir)
	_, _ = fmt.Fprintf(os.Stdout, "Web files: %s\n", webDir)
	_, _ = fmt.Fprintf(os.Stdout, "Permessive CORS: %t\n", pcors)
	_, _ = fmt.Fprintf(os.Stdout, "Starting %s keydesk\n", BrigadeID)

	if len(listeners) == 0 && !addr.IsValid() {
		_, _ = fmt.Fprintln(os.Stdout, "neither listeners nor address:port specified, exiting", err)
		os.Exit(1)
	}

	if len(listeners) == 0 {
		prev := calculatedAddrPort.Prev().String()

		l, err := net.Listen("tcp6", fmt.Sprintf("[%s]:80", prev))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stdout, prev, "listen HTTP error:", err)
			os.Exit(1)
		}
		listeners = append(listeners, l)

		l, err = net.Listen("tcp6", fmt.Sprintf("[%s]:443", prev))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stdout, prev, "listen HTTPS error:", err)
			os.Exit(1)
		}
		listeners = append(listeners, l)
	}

	handler := initSwaggerAPI(db, BrigadeID, &routerPublicKey, &shufflerPublicKey, pcors, webDir, allowedAddress)

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	done := make(chan struct{})
	statDone := make(chan struct{})
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
			_, _ = fmt.Fprintf(os.Stdout, "Skip TLS: can't open cert/key pair: %s\n", err)
		}
	}

	go func() {
		<-quit
		_, _ = fmt.Fprintln(os.Stdout, "Quit signal received...")
		statDone <- struct{}{}

		wg := sync.WaitGroup{}

		closeFunc := func(srv *http.Server) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			srv.SetKeepAlivesEnabled(false)
			if err := srv.Shutdown(ctx); err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Can't gracefully shut down the server: %s\n", err)
			}
		}

		_, _ = fmt.Fprintln(os.Stdout, "Server is shutting down")
		wg.Add(1)
		go closeFunc(server)

		if serverTLS != nil {
			_, _ = fmt.Fprintln(os.Stdout, "Server TLS is shutting down")
			wg.Add(1)
			go closeFunc(serverTLS)
		}

		wg.Wait()

		close(done)
	}()

	if len(listeners) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Listen HTTP: %s\n", listeners[0].Addr().String())
		// Start accepting connections.
		go func() {
			if err := server.Serve(listeners[0]); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Can't serve: %s\n", err)
			}
		}()
	}

	if serverTLS != nil && len(listeners) == 2 {
		_, _ = fmt.Fprintf(os.Stdout, "Listen HTTPS: %s\n", listeners[1].Addr().String())
		// Start accepting connections.
		go func() {
			if err := serverTLS.ServeTLS(listeners[1], "", ""); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Can't serve TLS: %s\n", err)
			}
		}()
	}

	go stat.CollectingData(statDone, addr, true, BrigadeID, dbDir, *statsDir) // TODO: is addr the same?

	// Wait for existing connections before exiting.
	<-done
}

func parseArgs() (bool, bool, bool, []net.Listener, netip.AddrPort, string, string, string, string, string, string, namesgenerator.Person, bool, *storage.ConfigsImplemented, error) {
	var (
		id                     string
		etcdir, dbdir, certdir string
		person                 namesgenerator.Person
		addrPort               netip.AddrPort
	)

	sysUser, err := user.Current()
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot define user: %w", err)
	}

	webDir := flag.String("w", DefaultWebDir, "Dir for web files.")
	etcDir := flag.String("c", "", "Dir for config files (for test). Default: "+keydesk.DefaultEtcDir)
	filedbDir := flag.String("d", "", "Dir for db files (for test). Default: "+storage.DefaultHomeDir+"/<BrigadeID>")
	certDir := flag.String("e", "", "Dir for TLS certificate and key (for test). Default: "+DefaultCertDir)
	pcors := flag.Bool("cors", false, "Turn on permessive CORS (for test)")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	listenAddr := flag.String("l", "", "Listen addr:port (http and https separate with commas)")

	brigadierName := flag.String("name", "", "brigadierName :: base64")
	personName := flag.String("person", "", "personName :: base64")
	personDesc := flag.String("desc", "", "personDesc :: base64")
	personURL := flag.String("url", "", "personURL :: base64")
	replaceBrigadier := flag.Bool("r", false, "Replace brigadier config")

	addr := flag.String("a", vpnapi.TemplatedAddrPort, "API endpoint address:port")

	chunked := flag.Bool("ch", false, "chunked output")
	jsonOut := flag.Bool("j", false, "json output")

	wgcCfgs := flag.String("wg", "native,amnezia", "Wireguard configs ("+storage.ConfigsWg+")")
	ovcCfgs := flag.String("ovc", "", "OpenVPN over Cloak configs ("+storage.ConfigsOvc+")")
	ipsecCfgs := flag.String("ipsec", "", "IPSec configs ("+storage.ConfigsIPSec+")")
	outlineCfgs := flag.String("outline", "", "Outline configs ("+storage.ConfigsOutline+")")

	flag.Parse()

	vpnCfgs := storage.NewConfigsImplemented()

	if *wgcCfgs != "" {
		vpnCfgs.AddWg(*wgcCfgs)
	}

	if *ovcCfgs != "" {
		vpnCfgs.AddOvc(*ovcCfgs)
	}

	if *ipsecCfgs != "" {
		vpnCfgs.AddIPSec(*ipsecCfgs)
	}

	if *outlineCfgs != "" {
		vpnCfgs.AddOutline(*outlineCfgs)
	}

	if *webDir == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrStaticDirEmpty
	}

	webdir, err := filepath.Abs(*webDir)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("web dir: %w", err)
	}

	if *filedbDir != "" {
		dbdir, err = filepath.Abs(*filedbDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("dbdir dir: %w", err)
		}
	}

	if *etcDir != "" {
		etcdir, err = filepath.Abs(*etcDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("etcdir dir: %w", err)
		}
	}

	if *certDir != "" {
		certdir, err = filepath.Abs(*certDir)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("statdir dir: %w", err)
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

		if *certDir == "" {
			certdir = DefaultCertDir
		}
	default:
		id = *brigadeID

		cwd, err := os.Getwd()
		if err == nil {
			cwd, _ = filepath.Abs(cwd)
		}

		if *filedbDir == "" {
			dbdir = cwd
		}

		if *etcDir == "" {
			etcdir = cwd
		}

		if *certDir == "" {
			certdir = cwd
		}
	}

	// brigadeID must be base32 decodable.
	binID, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(id)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("id base32: %s: %w", id, err)
	}

	_, err = uuid.FromBytes(binID)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("id uuid: %s: %w", id, err)
	}

	if *addr != "-" {
		addrPort, err = netip.ParseAddrPort(*addr)
		if err != nil {
			return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("api addr: %w", err)
		}
	}

	if *replaceBrigadier {
		return *chunked, *jsonOut, *pcors, nil, addrPort, id, etcdir, webdir, dbdir, certdir, "", person, *replaceBrigadier, vpnCfgs, nil
	}

	if *brigadierName == "" {
		var listeners []net.Listener

		switch *listenAddr {
		case "":
			// get listeners from activation sockets
			listeners, err = activation.Listeners()
			if err != nil {
				return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot retrieve listeners: %w", err)
			}

			//if len(listeners) != 1 && len(listeners) != 2 {
			//	return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("unexpected number of socket activation (%d != 1|2)",
			//		len(listeners))
			//}
			return *chunked, *jsonOut, *pcors, listeners, addrPort, id, etcdir, webdir, dbdir, certdir, "", person, false, nil, nil
		default:
			// get listeners from argument
			for _, laddr := range strings.Split(*listenAddr, ",") {
				l, err := net.Listen("tcp", laddr)
				if err != nil {
					return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("cannot listen: %w", err)
				}

				listeners = append(listeners, l)
			}

			if len(listeners) != 1 && len(listeners) != 2 {
				return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("unexpected number of litening (%d != 1|2)",
					len(listeners))
			}
		}

		return *chunked, *jsonOut, *pcors, listeners, addrPort, id, etcdir, webdir, dbdir, certdir, "", person, false, nil, nil
	}

	// brigadierName must be not empty and must be a valid UTF8 string
	buf, err := base64.StdEncoding.DecodeString(*brigadierName)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("brigadier name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrInvalidBrigadierName
	}

	name := string(buf)

	// personName must be not empty and must be a valid UTF8 string
	if *personName == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrEmptyPersonName
	}

	buf, err = base64.StdEncoding.DecodeString(*personName)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("person name: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrInvalidPersonName
	}

	person.Name = string(buf)

	// personDesc must be not empty and must be a valid UTF8 string
	if *personDesc == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrEmptyPersonDesc
	}

	buf, err = base64.StdEncoding.DecodeString(*personDesc)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("person desc: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrInvalidPersonDesc
	}

	person.Desc = string(buf)

	// personURL must be not empty and must be a valid UTF8 string
	if *personURL == "" {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrEmptyPersonURL
	}

	buf, err = base64.StdEncoding.DecodeString(*personURL)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("person url: %w", err)
	}

	if !utf8.Valid(buf) {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, ErrInvalidPersonURL
	}

	u := string(buf)

	_, err = url.Parse(u)
	if err != nil {
		return false, false, false, nil, addrPort, "", "", "", "", "", "", person, false, nil, fmt.Errorf("parse person url: %w", err)
	}

	person.URL = u

	return *chunked, *jsonOut, *pcors, nil, addrPort, id, etcdir, webdir, dbdir, certdir, name, person, *replaceBrigadier, vpnCfgs, nil
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
	jsonOut bool,
	name string,
	person namesgenerator.Person,
	replace bool,
	vpnCfgs *storage.ConfigsImplemented,
	routerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
) error {
	var w io.WriteCloser

	switch chunked {
	case true:
		w = httputil.NewChunkedWriter(os.Stdout)
		defer w.Close()
	default:
		w = os.Stdout
	}

	wgconf, filename, confJson, creationErr := keydesk.AddBrigadier(db, name, person, replace, vpnCfgs, routerPublicKey, shufflerPublicKey)

	switch jsonOut {
	case true:
		enc := json.NewEncoder(w)

		enc.SetIndent(" ", " ")

		if creationErr != nil {
			err := fmt.Errorf("add brigadier: %w", creationErr)

			answer := &keydesk.Answer{
				Code:    http.StatusInternalServerError,
				Desc:    http.StatusText(http.StatusInternalServerError),
				Status:  keydesk.AnswerStatusError,
				Message: err.Error(),
			}

			if err := enc.Encode(answer); err != nil {
				return fmt.Errorf("print json: %w", err)
			}

			return err
		}

		answer := &keydesk.Answer{
			Code:    http.StatusCreated,
			Desc:    http.StatusText(http.StatusCreated),
			Status:  keydesk.AnswerStatusSuccess,
			Configs: *confJson,
		}

		if err := enc.Encode(answer); err != nil {
			return fmt.Errorf("print json: %w", err)
		}
	default:
		if creationErr != nil {
			return fmt.Errorf("add brigadier: %w", creationErr)
		}

		if _, err := fmt.Fprintln(w, filename); err != nil {
			return fmt.Errorf("print filename: %w", err)
		}

		if _, err := fmt.Fprintln(w, wgconf); err != nil {
			return fmt.Errorf("pring wgconf: %w", err)
		}
	}

	if _, err := fmt.Println(); err != nil {
		return fmt.Errorf("print newline: %w", err)
	}

	return nil
}

func initSwaggerAPI(db *storage.BrigadeStorage,
	brigadeID string,
	routerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	pcors bool,
	webDir string,
	allowedAddr string,
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
	api.GetUsersStatsHandler = operations.GetUsersStatsHandlerFunc(func(params operations.GetUsersStatsParams, principal interface{}) middleware.Responder {
		return keydesk.GetUsersStats(db, params, principal)
	})

	switch pcors {
	case true:
		return cors.AllowAll().Handler(
			uiMiddleware(api.Serve(nil), webDir, allowedAddr),
		)
	default:
		return uiMiddleware(api.Serve(nil), webDir, allowedAddr)
	}
}

func uiMiddleware(handler http.Handler, dir string, allowedAddr string) http.Handler {
	staticFS := http.Dir(dir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddrPort, err := netip.ParseAddrPort(r.RemoteAddr)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "Connect From Unparseable: %s: %s\n", r.RemoteAddr, err)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

			return
		}

		remoteAddr := remoteAddrPort.Addr().String()

		if allowedAddr != "" && remoteAddr != allowedAddr {
			_, _ = fmt.Fprintf(os.Stdout, "Connect From: %s Restricted\n", r.RemoteAddr)
			_, _ = fmt.Fprintf(os.Stdout, "Remote: %s Expected:%s\n", remoteAddr, allowedAddr)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

			return
		}

		filename := filepath.Join(dir, r.URL.Path)
		finfo, err := os.Stat(filename)

		// If the file doesn't exist and it is a directory, try to serve the default index file.
		if err == nil && finfo.IsDir() {
			_, err = os.Stat(filepath.Join(filename, DefaultIndexFile))
		}

		// If the file exists, serve it.
		if err == nil {
			w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
			http.FileServer(staticFS).ServeHTTP(w, r)

			return
		}

		_, _ = fmt.Fprintf(os.Stdout, "Connect From: %s\n", r.RemoteAddr)

		handler.ServeHTTP(w, r)
	})
}
