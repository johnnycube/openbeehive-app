package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/johnnycube/openbeehive-app/server/internal/auth"
	"github.com/johnnycube/openbeehive-app/server/internal/config"
	"github.com/johnnycube/openbeehive-app/server/internal/demo"
	"github.com/johnnycube/openbeehive-app/server/internal/gen/openbeehive/v1/openbeehivev1connect"
	"github.com/johnnycube/openbeehive-app/server/internal/service"
	"github.com/johnnycube/openbeehive-app/server/internal/storage/blob"
	sqlstore "github.com/johnnycube/openbeehive-app/server/internal/storage/sql"
	wsync "github.com/johnnycube/openbeehive-app/server/internal/sync"
	"github.com/johnnycube/openbeehive-app/server/internal/web"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("Openbeehive · profile=%s · DB=%s · Blob=%s", cfg.Profile, cfg.DB.Driver, cfg.Blob.Backend)

	// --- Storage backends (pluggable) ---
	store, err := sqlstore.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migration: %v", err)
	}

	// Demo tenant (off by default; BEEHIVE_DEMO=true). Seeds 15 hives across 4
	// apiaries and re-seeds hourly so the showcase stays consistent.
	if cfg.Demo.Enabled {
		if err := demo.New(store.DB(), wsync.NewHLC("demo"), cfg).Install(ctx); err != nil {
			log.Printf("demo: install failed: %v", err)
		}
	}

	blobs, err := blob.New(cfg)
	if err != nil {
		log.Fatalf("blob: %v", err)
	}
	_ = blobs // pass to the inspection service

	// --- Auth (session + OIDC multi-provider + WebAuthn) ---
	sessions := auth.NewSessionManager(cfg.Auth.SessionSecret, cfg.Auth.SessionTTL)
	var interceptors []connect.Interceptor

	mux := http.NewServeMux()

	authEnabled := len(cfg.OIDC.Enabled) > 0 || cfg.Auth.WebAuthn.Enabled || cfg.Auth.PasswordEnabled

	// Provisioner: creates/links users and their tenants (organizations).
	prov := auth.NewProvisioner(store.Users(), store.Orgs(), store.Members())

	// Multi-tenant endpoints (me / switch / create / invite / accept).
	if authEnabled {
		auth.NewTenantAPI(sessions, store.Users(), store.Orgs(), store.Members(), store.Invites(), prov, cfg).Routes(mux)
	}

	// Email + password onboarding (first account = admin; verification optional).
	if cfg.Auth.PasswordEnabled {
		auth.NewPasswordAuth(store.Users(), store.Invites(), sessions, cfg, prov).Routes(mux)
		log.Printf("auth: email/password onboarding enabled (verification=%v, open registration=%v)",
			cfg.Auth.EmailVerification, cfg.Auth.RegistrationOpen)
	}
	if len(cfg.OIDC.Enabled) > 0 {
		am, err := auth.NewManager(ctx, cfg)
		if err != nil {
			log.Fatalf("oidc: %v", err)
		}
		log.Printf("auth: OIDC providers active: %v", am.Names())
		mux.HandleFunc("/auth/login", am.LoginHandler(sessions))
		// Callback provisions/links an in-app user record (by subject, then email).
		mux.HandleFunc("/auth/callback", am.CallbackHandler(sessions, store.Users(), prov, cfg))
	}
	if cfg.Auth.WebAuthn.Enabled {
		wa, err := auth.NewWebAuthn(cfg, store.DB())
		if err != nil {
			log.Fatalf("webauthn: %v", err)
		}
		log.Printf("auth: WebAuthn enabled (rp=%s)", cfg.Auth.WebAuthn.RPID)
		wa.Routes(mux, sessions)
	}

	if authEnabled {
		mux.HandleFunc("/auth/logout", sessions.LogoutHandler())
		interceptors = append(interceptors, auth.SessionInterceptor(sessions))
		// Demo sessions are read-only: reject mutating RPCs server-side.
		if cfg.Demo.Enabled {
			interceptors = append(interceptors, auth.ReadOnlyGuard())
		}
	} else {
		// Self-host single-user mode: no auth configured -> inject a fixed
		// local identity so the offline-first app works without login.
		log.Printf("auth: none configured, single-user mode (user=%s)", auth.LocalUser.UserID)
		interceptors = append(interceptors, auth.DevInterceptor())
	}

	// --- Connect-RPC handlers ---
	opts := connect.WithInterceptors(interceptors...)

	mux.Handle(openbeehivev1connect.NewApiaryServiceHandler(
		service.NewApiaryService(store.Apiaries()), opts))
	mux.Handle(openbeehivev1connect.NewHiveServiceHandler(
		service.NewHiveService(store.Hives()), opts))
	// Offline sync: central push/pull endpoint (main write path).
	hlc := wsync.NewHLC(cfg.Sync.NodeID)
	mux.Handle(openbeehivev1connect.NewSyncServiceHandler(
		service.NewSyncService(store.DB(), hlc), opts))

	// FS blob serving in self-host mode (files only, no directory listings).
	if cfg.Blob.Backend == config.BlobFS {
		mux.Handle("/files/", http.StripPrefix("/files/",
			noDirListing(http.FileServer(http.Dir(cfg.Blob.BaseDir)))))
	}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Embedded SPA (single-binary production). Registered last, as the catch-all.
	if cfg.Web.Serve {
		if h, err := web.Handler(cfg); err != nil {
			log.Printf("web: not serving embedded app (%v) — run vite separately in dev", err)
		} else {
			mux.Handle("/", h)
			log.Printf("web: serving embedded app at /")
		}
	}

	// CORS for browser clients; Connect-Web needs these headers.
	handler := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Connect-Protocol-Version", "Grpc-Status", "Grpc-Message"},
		AllowCredentials: cfg.CORS.AllowCredentials,
	}).Handler(mux)

	srv := &http.Server{
		Addr: cfg.Addr,
		// h2c allows gRPC (HTTP/2) without TLS; TLS termination at the reverse proxy.
		Handler:           h2c.NewHandler(handler, &http2.Server{}),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	runCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-runCtx.Done()
	log.Printf("shutting down…")
	shutCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// noDirListing hides directory indexes from the blob file server.
func noDirListing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}
