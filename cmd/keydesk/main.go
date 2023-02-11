package main

import (
	"context"
	goerrors "errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/vpngine/naclkey"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/rs/cors"
)

//go:generate swagger generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user

// TokenLifeTime - token time to life.
const TokenLifeTime = 3600

// Default web config.
const (
	DefaultStaticDir = "/var/www"
	DefaultHomeDir   = ""
	DefaultEtcDir    = "/etc"
	DefaultIndexFile = "index.html"
)

const (
	routerPublicKeyFilename   = "router.pub"
	shufflerPublicKeyFilename = "shuffler.pub"
)

// ErrStaticDirEmpty - no static dir name.
var ErrStaticDirEmpty = goerrors.New("empty static dirname")

func main() {
	var handler http.Handler

	pcors, listen, BrigadeID, etcDir, webDir, dbDir, err := parseArgs()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	db := keydesk.BrigadeStorage{
		BrigadeFilename: filepath.Join(dbDir, keydesk.BrigadeFilename),
		StatsFilename:   filepath.Join(dbDir, keydesk.StatsFilename),
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

func parseArgs() (bool, net.Listener, string, string, string, string, error) {
	etcDir := flag.String("c", DefaultEtcDir, "Dir for config files (for test)")
	homeDir := flag.String("d", DefaultHomeDir, "Dir for db files (for test)")
	brigadeID := flag.String("id", "", "BrigadeID (for test)")
	listenAddr := flag.String("l", "", "Listen addr:port (for test)")
	staticDir := flag.String("w", DefaultStaticDir, "Dir for web files (for test)")
	pcors := flag.Bool("cors", false, "Turn on permessive CORS")

	flag.Parse()

	if *staticDir == "" {
		return false, nil, "", "", "", "", ErrStaticDirEmpty
	}

	webdir, err := filepath.Abs(*staticDir)
	if err != nil {
		return false, nil, "", "", "", "", fmt.Errorf("web dir: %w", err)
	}

	if *listenAddr != "" && *brigadeID != "" {
		if *homeDir == "" {
			*homeDir, err = os.Getwd()
			if err != nil {
				return false, nil, "", "", "", "", fmt.Errorf("cur dir: %w", err)
			}
		}

		dbdir, err := filepath.Abs(*homeDir)
		if err != nil {
			return false, nil, "", "", "", "", fmt.Errorf("dbdir dir: %w", err)
		}

		if *etcDir == "" {
			*etcDir, err = os.Getwd()
			if err != nil {
				return false, nil, "", "", "", "", fmt.Errorf("cur dir: %w", err)
			}
		}

		etcdir, err := filepath.Abs(*etcDir)
		if err != nil {
			return false, nil, "", "", "", "", fmt.Errorf("etcdir dir: %w", err)
		}

		listen, err := net.Listen("tcp", *listenAddr)
		if err != nil {
			return true, nil, "", "", "", "", fmt.Errorf("cannot listen: %w", err)
		}

		return true, listen, *brigadeID, etcdir, webdir, dbdir, nil
	}

	usr, err := user.Current()
	if err != nil {
		return false, nil, "", "", "", "", fmt.Errorf("cannot define user: %w", err)
	}

	id := usr.Username

	if *homeDir == "" {
		*homeDir = filepath.Join("home", id)
	}

	dbdir, err := filepath.Abs(*homeDir)
	if err != nil {
		return false, nil, "", "", "", "", fmt.Errorf("dbdir dir: %w", err)
	}

	etcdir, err := filepath.Abs(*etcDir)
	if err != nil {
		return false, nil, "", "", "", "", fmt.Errorf("etcdir dir: %w", err)
	}

	listeners, err := activation.Listeners()
	if err != nil {
		return false, nil, "", "", "", "", fmt.Errorf("cannot retrieve listeners: %w", err)
	}

	if len(listeners) != 1 {
		return false, nil, "", "", "", "", fmt.Errorf("unexpected number of socket activation (%d != 1)",
			len(listeners))
	}

	return *pcors, listeners[0], id, etcdir, webdir, dbdir, nil
}

func readPubKeys(path string) ([naclkey.NaclBoxKeyLength]byte, [naclkey.NaclBoxKeyLength]byte, error) {
	empty := [naclkey.NaclBoxKeyLength]byte{}

	routerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, routerPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("router key: %w", err)
	}

	shufflerPublicKey, err := naclkey.ReadPublicKeyFile(filepath.Join(path, shufflerPublicKeyFilename))
	if err != nil {
		return empty, empty, fmt.Errorf("shuffler key: %w", err)
	}

	return routerPublicKey, shufflerPublicKey, nil
}
