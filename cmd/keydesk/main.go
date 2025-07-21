package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	stderrors "errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/rs/cors"
	goSwaggerAuth "github.com/vpngen/keydesk/internal/auth/go-swagger"
	"github.com/vpngen/keydesk/internal/maintenance"
	msgapp "github.com/vpngen/keydesk/internal/messages/app"
	msgsvc "github.com/vpngen/keydesk/internal/messages/service"
	"github.com/vpngen/keydesk/internal/server"
	shflrapp "github.com/vpngen/keydesk/internal/shuffler/app"
	"github.com/vpngen/keydesk/internal/stat"
	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
	jwtsvc "github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/pkg/runner"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// Default web config.
const (
	DefaultWebDir    = "/var/www"
	DefaultIndexFile = "index.html"
	DefaultCertDir   = "/etc/vgcert"
	TLSCertFilename  = "vpn.works.crt"
	TLSKeyFilename   = "vpn.works.key"
	TokenLifeTime    = 3600
)

// Args errors.
var (
	ErrInvalidBrigadierName = stderrors.New("invalid brigadier name")
	ErrEmptyPersonName      = stderrors.New("empty person name")
	ErrEmptyPersonDesc      = stderrors.New("empty person desc")
	ErrEmptyPersonURL       = stderrors.New("empty person url")
	ErrInvalidPersonName    = stderrors.New("invalid person name")
	ErrInvalidPersonDesc    = stderrors.New("invalid person desc")
	ErrInvalidPersonURL     = stderrors.New("invalid person url")
	ErrStaticDirEmpty       = stderrors.New("empty static dirname")
	ErrNotAlowedInThisMode  = stderrors.New("not allowed in this mode")
)

func errQuit(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
	os.Exit(1)
}

func main() {
	cfg, err := parseArgs2(parseFlags(flag.CommandLine, os.Args[1:]))
	if err != nil {
		errQuit("Can't init", err)
	}

	routerPublicKey, shufflerPublicKey, err := readPubKeys(cfg.etcDir)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Fprintf(os.Stderr, "Etc: %s\n", cfg.etcDir)
	fmt.Fprintf(os.Stderr, "DBDir: %s\n", cfg.dbDir)

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
		errQuit("Storage initialization", err)
	}

	switch {
	case cfg.addr.IsValid() && cfg.addr.Addr().IsUnspecified():
		fmt.Fprintln(os.Stderr, "Command address:port is COMMON")
	case cfg.addr.IsValid():
		fmt.Fprintf(os.Stderr, "Command address:port: %s\n", cfg.addr)
	default:
		fmt.Fprintln(os.Stderr, "Command address:port is for DEBUG")
	}

	raw, brigade, err := db.OpenDbToModify()
	if err != nil {
		errQuit("open db", err)
	}

	if brigade.Ver < 11 && brigade.Proto0Port != 0 && len(brigade.Proto0FakeDomains) == 0 {
		brigade.Proto0FakeDomains = keydesk.GetRandomSites0()

		if err := raw.Commit(brigade); err != nil {
			errQuit("commit db", err)
		}
	}

	if err = raw.Close(); err != nil {
		errQuit("close db", err)
	}

	// Just create brigadier.
	if cfg.brigadierName != "" || cfg.replaceBrigadier {
		if brigade.Mode != storage.ModeBrigade {
			errQuit("Can't create brigadier", ErrNotAlowedInThisMode)
		}

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
			errQuit("Can't create brigadier", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "Cert Dir: %s\n", cfg.certDir)
	fmt.Fprintf(os.Stderr, "Stat Dir: %s\n", cfg.statsDir)
	fmt.Fprintf(os.Stderr, "Web files: %s\n", cfg.webDir)
	fmt.Fprintf(os.Stderr, "Permessive CORS: %t\n", cfg.enableCORS)
	fmt.Fprintf(os.Stderr, "Starting %s keydesk\n", cfg.brigadeID)

	allowedAddress := ""
	calculatedAddrPort, ok := db.CalculatedAPIAddress()
	if ok {
		allowedAddress = calculatedAddrPort.String()
		fmt.Fprintf(os.Stderr, "Resqrict requests by address: %s \n", allowedAddress)
	}

	//if len(cfg.listeners) == 0 && !cfg.addr.IsValid() {
	//	errQuit("neither listeners nor address:port specified", nil)
	//}

	if brigade.Mode == storage.ModeBrigade && len(cfg.listeners) == 0 {
		prev := calculatedAddrPort.Prev().String()

		l, err := net.Listen("tcp6", fmt.Sprintf("[%s]:80", prev))
		if err != nil {
			errQuit("listen HTTP", err)
		}
		cfg.listeners = append(cfg.listeners, l)

		l, err = net.Listen("tcp6", fmt.Sprintf("[%s]:443", prev))
		if err != nil {
			errQuit("listen HTTPS", err)
		}
		cfg.listeners = append(cfg.listeners, l)
	}

	handler := initSwaggerAPI(
		db,
		&routerPublicKey,
		&shufflerPublicKey,
		cfg.enableCORS,
		cfg.webDir,
		allowedAddress,
		cfg.jwtKeydeskIssuer,
		cfg.jwtKeydesAuthorizer,
	)

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.

	statDone := make(chan struct{})

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
				Handler:     redirectToHTTPS()(handler),
				IdleTimeout: 60 * time.Minute,
			}
		default:
			fmt.Fprintf(os.Stderr, "Skip TLS: can't open cert/key pair: %s\n", err)
			if err := cfg.listeners[1].Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Close TLS listener: %s\n", err)
			}
		}
	}

	baseCtx := context.Background()
	r := runner.New(baseCtx)

	if len(cfg.listeners) > 0 {
		r.AddTask("keydesk http", runner.Task{
			Func: func(ctx context.Context) error {
				fmt.Fprintf(os.Stderr, "Listen HTTP: %s\n", cfg.listeners[0].Addr().String())
				if err := srv.Serve(cfg.listeners[0]); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
					return err
				}

				return nil
			},
			Shutdown: func(ctx context.Context) error {
				return srv.Shutdown(ctx)
			},
		})
	}

	if serverTLS != nil && len(cfg.listeners) == 2 {
		r.AddTask("keydesk https", runner.Task{
			Func: func(ctx context.Context) error {
				_, _ = fmt.Fprintf(os.Stderr, "Listen HTTPS: %s\n", cfg.listeners[1].Addr().String())
				if err := serverTLS.ServeTLS(cfg.listeners[1], "", ""); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
					return err
				}
				return nil
			},
			Shutdown: func(ctx context.Context) error {
				return serverTLS.Shutdown(ctx)
			},
		})
	}

	_, rdata := os.LookupEnv("VGSTATS_RANDOM_DATA")

	r.AddTask("stat", runner.Task{
		Func: func(ctx context.Context) error {
			stat.CollectingData(db, statDone, rdata, cfg.statsDir)
			return nil
		},
		Shutdown: func(ctx context.Context) error {
			statDone <- struct{}{}
			return nil
		},
	})

	fmt.Fprintf(os.Stderr, "Brigade mode: %s \n", brigade.Mode)

	if brigade.Mode == storage.ModeBrigade &&
		cfg.messageAPISocket != nil &&
		!cfg.jwtMsgAuthorizer.IsNil() {
		echoSrv, err := msgapp.SetupServer(db, cfg.jwtMsgAuthorizer)
		if err != nil {
			errQuit("message server", err)
		}
		echoSrv.Listener = cfg.messageAPISocket

		r.AddTask("messages", runner.Task{
			Func: func(ctx context.Context) error {
				if err := echoSrv.Start(""); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
					return err
				}
				return nil
			},
			Shutdown: func(ctx context.Context) error {
				if err = echoSrv.Shutdown(ctx); err != nil {
					return err
				}
				return nil
			},
		})
	}

	// start socket interface for any mode to stats access
	if cfg.shufflerAPISocket != nil {
		echoSrv, err := shflrapp.SetupServer(db, cfg.jwtMsgAuthorizer, routerPublicKey, shufflerPublicKey)
		if err != nil {
			errQuit("shuffler server", err)
		}
		echoSrv.Listener = cfg.shufflerAPISocket
		r.AddTask("shuffler", runner.Task{
			Func: func(ctx context.Context) error {
				if err = echoSrv.Start(""); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
					return err
				}
				return nil
			},
			Shutdown: func(ctx context.Context) error {
				if err = echoSrv.Shutdown(ctx); err != nil {
					return err
				}
				return nil
			},
		})
	}

	r.Run()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	if err = r.Stop(); err != nil {
		errQuit("runner", err)
	}
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
		if stderrors.As(creationErr, &me) {
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

	return nil
}

func initSwaggerAPI(
	db *storage.BrigadeStorage,
	routerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
	pcors bool,
	webDir string,
	allowedAddr string,
	issuer jwtsvc.KeydeskTokenIssuer,
	authorizer jwtsvc.KeydeskTokenAuthorizer,
) http.Handler {
	api := server.NewServer(
		db,
		msgsvc.New(db),
		issuer,
		goSwaggerAuth.NewService(authorizer),
		routerPublicKey,
		shufflerPublicKey,
		TokenLifeTime,
	)

	handler := api.Serve(nil)
	handler = maintenanceMiddlewareBuilder(
		"/.maintenance",
		filepath.Dir(db.BrigadeFilename)+"/.maintenance",
		filepath.Dir(db.BrigadeFilename)+"/.maintenance_till_restore",
	)(handler)
	handler = uiMiddlewareBuilder(webDir, allowedAddr)(handler)
	if pcors {
		return cors.AllowAll().Handler(handler)
	}
	return handler
}

func uiMiddlewareBuilder(dir string, allowedAddr string) middleware.Builder {
	return func(handler http.Handler) http.Handler {
		staticFS := http.Dir(dir)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteAddrPort, err := netip.ParseAddrPort(r.RemoteAddr)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Connect From Unparseable: %s: %s\n", r.RemoteAddr, err)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

				return
			}

			remoteAddr := remoteAddrPort.Addr().String()

			if allowedAddr != "" && remoteAddr != allowedAddr {
				fmt.Fprintf(os.Stdout, "Connect From: %s Restricted\n", r.RemoteAddr)
				fmt.Fprintf(os.Stdout, "Remote: %s Expected:%s\n", remoteAddr, allowedAddr)
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

			fmt.Fprintf(os.Stderr, "Connect From: %s\n", r.RemoteAddr)

			handler.ServeHTTP(w, r)
		})
	}
}

func maintenanceMiddlewareBuilder(paths ...string) middleware.Builder {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, till, msg := maintenance.CheckInPaths(paths...)
			if ok {
				me := maintenance.NewError(till, msg)
				w.Header().Set("Retry-After", me.RetryAfter().String())
				w.WriteHeader(http.StatusServiceUnavailable)
				if err := json.NewEncoder(w).Encode(me); err != nil {
					fmt.Fprintln(os.Stderr, "encode maintenance error:", err)
				}
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
}

func redirectToHTTPS() middleware.Builder {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Redirect only if the request is not already using HTTPS
			if r.Header.Get("X-Forwarded-Proto") != "https" && r.TLS == nil {
				http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)

				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}
