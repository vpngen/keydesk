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
	osuser "os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"

	"github.com/vpngen/keydesk/env"
	"github.com/vpngen/keydesk/gen/restapi"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/token"
	"github.com/vpngen/keydesk/user"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/rs/cors"
)

//go:generate swagger generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user

// TokenLifeTime - token time to life.
const TokenLifeTime = 3600

// Default web config.
const (
	DefaultStaticDir = "/var/www"
	DefaultEtcDir    = "/etc/keydesk"
	DefaultIndexFile = "index.html"
)

// ErrStaticDirEmpty - no static dir name.
var ErrStaticDirEmpty = goerrors.New("empty static dirname")

func main() {
	var handler http.Handler

	pcors, listen, BrigadierID, staticDir, etcDir, err := bootstrap()
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	// load embedded swagger file
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	env.Env.BrigadierID = BrigadierID

	err = env.ReadConfigs(etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	err = env.CreatePool()
	if err != nil {
		log.Fatalln(err)
	}

	defer env.Env.DB.Close()

	// create new service API
	api := operations.NewUserAPI(swaggerSpec)

	api.ServeError = errors.ServeError

	api.UseSwaggerUI()

	api.JSONConsumer = runtime.JSONConsumer()

	api.BinProducer = runtime.ByteStreamProducer()
	api.JSONProducer = runtime.JSONProducer()

	api.BearerAuth = token.ValidateBearer(BrigadierID)
	api.PostTokenHandler = operations.PostTokenHandlerFunc(token.CreateToken(BrigadierID, TokenLifeTime))

	api.PostUserHandler = operations.PostUserHandlerFunc(user.AddUser)
	api.DeleteUserUserIDHandler = operations.DeleteUserUserIDHandlerFunc(user.DelUserUserID)
	api.GetUserHandler = operations.GetUserHandlerFunc(user.GetUsers)

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	switch pcors {
	case true:
		handler = cors.AllowAll().Handler(
			uiMiddleware(api.Serve(nil), staticDir),
		)
	default:
		handler = uiMiddleware(api.Serve(nil), staticDir)
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

	fmt.Printf("Starting %s keydesk\n", BrigadierID)

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

		handler.ServeHTTP(w, r)
	})
}

func bootstrap() (bool, net.Listener, string, string, string, error) {
	staticDir := flag.String("w", DefaultStaticDir, "Dir for web files (for test)")
	etcDir := flag.String("c", DefaultEtcDir, "Dir for config files (for test)")
	listenAddr := flag.String("l", "", "Listen addr:port (for test)")
	brigadierID := flag.String("id", "", "BrigadierID (for test)")
	pcors := flag.Bool("cors", false, "Turn on permessive CORS")

	flag.Parse()

	if *staticDir == "" {
		return false, nil, "", "", "", ErrStaticDirEmpty
	}

	dir, err := filepath.Abs(*staticDir)
	if err != nil {
		return false, nil, "", "", "", fmt.Errorf("static dir: %w", err)
	}

	if *etcDir == "" {
		return false, nil, "", "", "", ErrStaticDirEmpty
	}

	etc, err := filepath.Abs(*etcDir)
	if err != nil {
		return false, nil, "", "", "", fmt.Errorf("etc dir: %w", err)
	}

	if *listenAddr != "" && *brigadierID != "" {
		listen, err := net.Listen("tcp", *listenAddr)
		if err != nil {
			return true, nil, "", "", "", fmt.Errorf("cannot listen: %w", err)
		}

		return true, listen, *brigadierID, dir, etc, nil
	}

	usr, err := osuser.Current()
	if err != nil {
		return false, nil, "", "", "", fmt.Errorf("cannot define user: %w", err)
	}

	id := usr.Username

	listeners, err := activation.Listeners()
	if err != nil {
		return false, nil, "", "", "", fmt.Errorf("cannot retrieve listeners: %w", err)
	}

	if len(listeners) != 1 {
		return false, nil, "", "", "", fmt.Errorf("unexpected number of socket activation (%d != 1)",
			len(listeners))
	}

	return *pcors, listeners[0], id, dir, etc, nil
}
