package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	goerrors "errors"
	"flag"
	"fmt"
	"github.com/go-openapi/runtime/middleware"
	"github.com/rs/cors"
	"github.com/vpngen/keydesk/internal/auth"
	"github.com/vpngen/keydesk/internal/maintenance"
	"github.com/vpngen/keydesk/internal/server"
	"github.com/vpngen/keydesk/internal/stat"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/message"
	"github.com/vpngen/keydesk/keydesk/push"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

//go:generate go run github.com/go-swagger/go-swagger/cmd/swagger@latest generate server -t ../../gen -f ../../swagger/swagger.yml --exclude-main -A user
//go:generate go run github.com/go-swagger/go-swagger/cmd/swagger@latest generate client -t ../../gen -f ../../swagger/swagger.yml

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

// Args errors.
var (
	ErrInvalidBrigadierName = goerrors.New("invalid brigadier name")
	ErrEmptyPersonName      = goerrors.New("empty person name")
	ErrEmptyPersonDesc      = goerrors.New("empty person desc")
	ErrEmptyPersonURL       = goerrors.New("empty person url")
	ErrInvalidPersonName    = goerrors.New("invalid person name")
	ErrInvalidPersonDesc    = goerrors.New("invalid person desc")
	ErrInvalidPersonURL     = goerrors.New("invalid person url")
	ErrStaticDirEmpty       = goerrors.New("empty static dirname")
)

func main() {
	cfg, err := parseArgs2(parseFlags(flag.CommandLine, os.Args[1:]))
	if err != nil {
		log.Fatalf("Can't init: %s\n", err)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(cfg.etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "Etc: %s\n", cfg.etcDir)
	_, _ = fmt.Fprintf(os.Stderr, "DBDir: %s\n", cfg.dbDir)

	db := &storage.BrigadeStorage{
		BrigadeID:       cfg.brigadeID,
		BrigadeFilename: filepath.Join(cfg.dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(cfg.dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     cfg.addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	switch {
	case cfg.addr.IsValid() && cfg.addr.Addr().IsUnspecified():
		_, _ = fmt.Fprintln(os.Stderr, "Command address:port is COMMON")
	case cfg.addr.IsValid():
		_, _ = fmt.Fprintf(os.Stderr, "Command address:port: %s\n", cfg.addr)
	default:
		_, _ = fmt.Fprintln(os.Stderr, "Command address:port is for DEBUG")
	}

	// Just create brigadier.
	if cfg.brigadierName != "" || cfg.replaceBrigadier {
		if err := createBrigadier(
			db,
			cfg.chunked,
			cfg.brigadierName,
			cfg.person,
			cfg.replaceBrigadier,
			cfg.vpnConfigs,
			&routerPublicKey,
			&shufflerPublicKey,
		); err != nil {
			log.Fatalf("Can't create brigadier: %s\n", err)
		}
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, "Cert Dir: %s\n", cfg.certDir)
	_, _ = fmt.Fprintf(os.Stderr, "Stat Dir: %s\n", cfg.statsDir)
	_, _ = fmt.Fprintf(os.Stderr, "Web files: %s\n", cfg.webDir)
	_, _ = fmt.Fprintf(os.Stderr, "Permessive CORS: %t\n", cfg.enableCORS)
	_, _ = fmt.Fprintf(os.Stderr, "Starting %s keydesk\n", cfg.brigadeID)

	allowedAddress := ""
	calculatedAddrPort, ok := db.CalculatedAPIAddress()
	if ok {
		allowedAddress = calculatedAddrPort.String()
		_, _ = fmt.Fprintf(os.Stderr, "Resqrict requests by address: %s \n", allowedAddress)
	}

	if len(cfg.listeners) == 0 && !cfg.addr.IsValid() {
		_, _ = fmt.Fprintln(os.Stderr, "neither listeners nor address:port specified, exiting")
		os.Exit(1)
	}

	if len(cfg.listeners) == 0 {
		prev := calculatedAddrPort.Prev().String()

		l, err := net.Listen("tcp6", fmt.Sprintf("[%s]:80", prev))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, prev, "listen HTTP error:", err)
			os.Exit(1)
		}
		cfg.listeners = append(cfg.listeners, l)

		l, err = net.Listen("tcp6", fmt.Sprintf("[%s]:443", prev))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, prev, "listen HTTPS error:", err)
			os.Exit(1)
		}
		cfg.listeners = append(cfg.listeners, l)
	}

	handler := initSwaggerAPI(db, &routerPublicKey, &shufflerPublicKey, cfg.enableCORS, cfg.webDir, allowedAddress)

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	done := make(chan struct{})
	statDone := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Handler:     handler,
		IdleTimeout: 60 * time.Minute,
	}

	var serverTLS *http.Server

	if len(cfg.listeners) == 2 {
		// openssl req -x509 -nodes -days 10000 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -subj '/CN=vpn.works/O=VPNGen/C=LT/ST=Vilniaus Apskritis/L=Vilnius' -keyout vpn.works.key -out vpn.works.crt
		switch cert, err := tls.LoadX509KeyPair(
			filepath.Join(cfg.certDir, TLSCertFilename),
			filepath.Join(cfg.certDir, TLSKeyFilename),
		); err {
		case nil:
			serverTLS = &http.Server{
				TLSConfig:   &tls.Config{Certificates: []tls.Certificate{cert}},
				Handler:     handler,
				IdleTimeout: 60 * time.Minute,
			}
		default:
			_, _ = fmt.Fprintf(os.Stderr, "Skip TLS: can't open cert/key pair: %s\n", err)
		}
	}

	go func() {
		<-quit
		_, _ = fmt.Fprintln(os.Stderr, "Quit signal received...")
		statDone <- struct{}{}

		wg := sync.WaitGroup{}

		closeFunc := func(srv *http.Server) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			srv.SetKeepAlivesEnabled(false)
			if err := srv.Shutdown(ctx); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Can't gracefully shut down the server: %s\n", err)
			}
		}

		_, _ = fmt.Fprintln(os.Stderr, "Server is shutting down")
		wg.Add(1)
		go closeFunc(srv)

		if serverTLS != nil {
			_, _ = fmt.Fprintln(os.Stderr, "Server TLS is shutting down")
			wg.Add(1)
			go closeFunc(serverTLS)
		}

		wg.Wait()

		close(done)
	}()

	if len(cfg.listeners) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Listen HTTP: %s\n", cfg.listeners[0].Addr().String())
		// Start accepting connections.
		go func() {
			if err := srv.Serve(cfg.listeners[0]); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Can't serve: %s\n", err)
			}
		}()
	}

	if serverTLS != nil && len(cfg.listeners) == 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Listen HTTPS: %s\n", cfg.listeners[1].Addr().String())
		// Start accepting connections.
		go func() {
			if err := serverTLS.ServeTLS(cfg.listeners[1], "", ""); err != nil && !goerrors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Can't serve TLS: %s\n", err)
			}
		}()
	}

	_, rdata := os.LookupEnv("VGSTATS_RANDOM_DATA")

	go stat.CollectingData(db, statDone, rdata, cfg.statsDir)

	// Wait for existing connections before exiting.
	<-done
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

	// TODO: do we have to print wgconf, filename?
	_, _, confJson, creationErr := keydesk.AddBrigadier(db, name, person, replace, vpnCfgs, routerPublicKey, shufflerPublicKey)

	enc := json.NewEncoder(w)

	enc.SetIndent(" ", " ")

	if creationErr != nil {
		me := maintenance.Error{}
		if goerrors.As(creationErr, &me) {
			return enc.Encode(keydesk.Answer{
				Code:    http.StatusServiceUnavailable,
				Desc:    http.StatusText(http.StatusServiceUnavailable),
				Status:  keydesk.AnswerStatusError,
				Message: me.Error(),
			})
		}

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

	if _, err := fmt.Println(); err != nil {
		return fmt.Errorf("print newline: %w", err)
	}

	return nil
}

func initSwaggerAPI(
	db *storage.BrigadeStorage,
	routerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	pcors bool,
	webDir string,
	allowedAddr string,
) http.Handler {
	api := server.NewServer(
		db,
		message.New(db),
		push.New(
			db,
			"Lcw1hBkJBH2oSGevZBAp86kr4PDlQ1QxOFH8LkBNs_c",
			"BI8uqN-GskHtmeqH10szMwNNR29opGc31t8d2QGRPXCwLhoEo9vY6DNYx9X147TKVQEHrAXA3BfKfVuDBE06TbE",
		),
		auth.Service{
			Issuer:   "keydesk",
			Subject:  db.BrigadeID,
			Audience: []string{"keydesk"},
		},
		routerPublicKey,
		shufflerPublicKey,
		TokenLifeTime,
	)

	return api.Serve(func(handler http.Handler) http.Handler {
		if pcors {
			handler = cors.AllowAll().Handler(handler)
		}

		handler = maintenanceMiddlewareBuilder(
			"/.maintenance",
			filepath.Dir(db.BrigadeFilename)+"/.maintenance",
		)(handler)

		handler = uiMiddlewareBuilder(webDir, allowedAddr)(handler)

		return handler
	})
}

func uiMiddlewareBuilder(dir string, allowedAddr string) middleware.Builder {
	return func(handler http.Handler) http.Handler {
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

			_, _ = fmt.Fprintf(os.Stderr, "Connect From: %s\n", r.RemoteAddr)

			handler.ServeHTTP(w, r)
		})
	}
}

func maintenanceMiddlewareBuilder(paths ...string) middleware.Builder {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, till := maintenance.CheckInPaths(paths...)
			if ok {
				me := maintenance.NewError(till)
				w.Header().Set("Retry-After", me.RetryAfter().String())
				w.WriteHeader(http.StatusServiceUnavailable)
				if err := json.NewEncoder(w).Encode(me); err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "encode maintenance error:", err)
				}
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
}
