package main

import (
	"context"
	goerrors "errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	osuser "os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"

	"github.com/vpngen/keykeeper/gen/restapi"
	"github.com/vpngen/keykeeper/gen/restapi/operations"
	"github.com/vpngen/keykeeper/token"
	"github.com/vpngen/keykeeper/user"

	"github.com/coreos/go-systemd/v22/activation"
)

//go:generate swagger generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user

// TokenLifeTime - token time to life.
const TokenLifeTime = 3600

func main() {

	usr, err := osuser.Current()
	if err != nil {
		log.Fatalf(err.Error())
	}

	BrigadierID := usr.Username

	listeners, err := activation.Listeners()
	if err != nil {
		log.Panicf("cannot retrieve listeners: %s", err)
	}

	if len(listeners) != 1 {
		log.Panicf("unexpected number of socket activation (%d != 1)",
			len(listeners))
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

	server := &http.Server{
		Handler:     uiMiddleware(api.Serve(nil)),
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

	fmt.Printf("Starting %s keykeeper\n", BrigadierID)

	// Start accepting connections.
	if err := server.Serve(listeners[0]); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Can't serve: %s\n", err)
	}

	// Wait for existing connections before exiting.
	<-done
}

func uiMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Shortcut helpers for swagger-ui
		if r.URL.Path == "/swagger-ui" || r.URL.Path == "/api/help" {
			http.Redirect(w, r, "/swagger-ui/", http.StatusFound)
			return
		}
		// Serving ./swagger-ui/
		if strings.Index(r.URL.Path, "/swagger-ui/") == 0 {
			pwd, _ := os.Getwd()
			http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir(filepath.Join(pwd, "swagger-ui")))).ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
