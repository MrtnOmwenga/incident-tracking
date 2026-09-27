// Package web serves Lighthouse over HTTP: the public status page (HTML rendered on the server),
// and the JSON API behind the console.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"github.com/MrtnOmwenga/lighthouse/internal/auth"
	"github.com/MrtnOmwenga/lighthouse/internal/config"
	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

type Server struct {
	Config      config.Config
	Pool        *pgxpool.Pool
	Auth        *auth.Service
	Log         *slog.Logger
	OwnerTenant string // whose monitors the public status page shows
	Now         func() time.Time

	pages     *template.Template
	sandboxes *limiter // new sandboxes per client
	writes    *limiter // API writes per client
}

func New(cfg config.Config, pool *pgxpool.Pool, authn *auth.Service, log *slog.Logger, ownerTenant string) *Server {
	return &Server{
		Config: cfg, Pool: pool, Auth: authn, Log: log, OwnerTenant: ownerTenant, Now: time.Now,
		pages:     template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html")),
		sandboxes: newLimiter(rate.Every(10*time.Minute), 3),
		writes:    newLimiter(rate.Every(200*time.Millisecond), 20),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheFor(time.Hour, http.FileServerFS(static))))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/status", http.StatusFound) })

	// Public.
	mux.HandleFunc("GET /status", s.statusPage)
	mux.HandleFunc("GET /status/incidents/{id}", s.incidentPage)
	mux.HandleFunc("GET /api/status", s.publicStatus)

	// Signing in and out.
	mux.HandleFunc("GET /auth/github", s.errs(s.Auth.GitHubStart))
	mux.HandleFunc("GET /auth/github/callback", s.githubCallback)
	mux.HandleFunc("POST /auth/dev", s.errs(func(w http.ResponseWriter, r *http.Request) error {
		if err := s.Auth.DevLogin(w, r); err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, map[string]string{"role": "owner"})
	}))
	mux.HandleFunc("POST /auth/logout", s.errs(func(w http.ResponseWriter, r *http.Request) error {
		if err := s.Auth.Logout(w, r); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	mux.HandleFunc("POST /api/sandbox", s.errs(s.createSandbox))
	mux.HandleFunc("GET /api/me", s.signedIn(s.me))

	// The console API: every query runs inside the caller's tenant.
	mux.HandleFunc("GET /api/my-status", s.signedIn(s.myStatus))
	mux.HandleFunc("GET /api/monitors", s.signedIn(s.listMonitors))
	mux.HandleFunc("POST /api/monitors", s.signedIn(s.createMonitor))
	mux.HandleFunc("GET /api/monitors/{id}", s.signedIn(s.getMonitor))
	mux.HandleFunc("PUT /api/monitors/{id}", s.signedIn(s.updateMonitor))
	mux.HandleFunc("DELETE /api/monitors/{id}", s.signedIn(s.deleteMonitor))
	mux.HandleFunc("GET /api/monitors/{id}/checks", s.signedIn(s.listChecks))
	mux.HandleFunc("PUT /api/monitors/{id}/mode", s.signedIn(s.setMode))
	mux.HandleFunc("GET /api/incidents", s.signedIn(s.listIncidents))
	mux.HandleFunc("POST /api/incidents", s.signedIn(s.createIncident))
	mux.HandleFunc("GET /api/incidents/{id}", s.signedIn(s.getIncident))
	mux.HandleFunc("PATCH /api/incidents/{id}", s.signedIn(s.updateIncident))
	mux.HandleFunc("POST /api/incidents/{id}/comments", s.signedIn(s.addComment))

	return s.recoverer(s.logRequests(securityHeaders(s.Config, s.sameOrigin(s.limitWrites(s.Auth.Middleware(mux))))))
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.Pool.Ping(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) githubCallback(w http.ResponseWriter, r *http.Request) {
	err := s.Auth.GitHubCallback(w, r)
	switch {
	case err == nil:
		http.Redirect(w, r, "/", http.StatusFound)
	case errors.Is(err, auth.ErrNotOwner):
		s.renderError(w, http.StatusForbidden, "This is Martin's console", "Only the owner can sign in here. You can still look around: the status page is public.")
	default:
		s.Log.Warn("github sign-in failed", "err", err)
		s.renderError(w, http.StatusBadRequest, "Sign-in failed", "GitHub sign-in didn't complete. Please try again.")
	}
}

func (s *Server) createSandbox(w http.ResponseWriter, r *http.Request) error {
	if !s.sandboxes.allow(s.clientIP(r)) {
		return errTooMany
	}
	if err := s.Auth.Sandbox(w, r); err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, map[string]any{"role": "sandbox", "expiresInMinutes": int(s.Config.SandboxTTL.Minutes())})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, id auth.Identity) error {
	return writeJSON(w, http.StatusOK, map[string]string{"role": id.Role, "login": id.Login})
}

// signedIn wraps a handler that needs a session.
func (s *Server) signedIn(h func(http.ResponseWriter, *http.Request, auth.Identity) error) http.HandlerFunc {
	return s.errs(func(w http.ResponseWriter, r *http.Request) error {
		id, ok := auth.FromContext(r.Context())
		if !ok {
			return errUnauthenticated
		}
		return h(w, r, id)
	})
}

// inTenant runs fn in a transaction scoped to the caller's tenant.
func (s *Server) inTenant(r *http.Request, id auth.Identity, fn func(pgx.Tx) error) error {
	return store.WithTenant(r.Context(), s.Pool, id.TenantID, fn)
}

// Errors, as RFC 9457 problem details.

type problem struct {
	status int
	title  string
	detail string
}

func (p problem) Error() string { return p.title }

var (
	errUnauthenticated = problem{http.StatusUnauthorized, "Sign in first", ""}
	errForbidden       = problem{http.StatusForbidden, "Not allowed", ""}
	errNotFound        = problem{http.StatusNotFound, "Not found", ""}
	errTooMany         = problem{http.StatusTooManyRequests, "Too many requests", "Slow down and try again shortly."}
)

func invalid(detail string) problem {
	return problem{http.StatusUnprocessableEntity, "Invalid request", detail}
}

func (s *Server) errs(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		var p problem
		var pg *pgconn.PgError
		switch {
		case errors.As(err, &p):
		case errors.Is(err, store.ErrNotFound):
			p = errNotFound
		case errors.Is(err, auth.ErrForbidden):
			p = errForbidden
		case errors.As(err, &pg) && pg.Code == "23505":
			p = problem{http.StatusConflict, "Already exists", "That slug is already in use."}
		case errors.As(err, &pg) && (pg.Code == "23514" || pg.Code == "22001"):
			p = invalid("A value is outside its allowed range.")
		case errors.Is(err, context.Canceled):
			return
		default:
			s.Log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
			p = problem{http.StatusInternalServerError, "Something went wrong", ""}
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(p.status)
		body := map[string]any{"status": p.status, "title": p.title}
		if p.detail != "" {
			body["detail"] = p.detail
		}
		_ = json.NewEncoder(w).Encode(body)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// readJSON decodes a small JSON body strictly: unknown fields and trailing data are errors.
func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return invalid("The body must be a JSON object with the documented fields.")
	}
	if dec.More() {
		return invalid("The body must hold a single JSON object.")
	}
	return nil
}

// Middleware.

func securityHeaders(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src 'self' data:; script-src 'self'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if cfg.SecureCookies() {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// sameOrigin refuses state-changing requests from other sites. SameSite=Lax cookies already stop
// most cross-site requests; this also covers same-site subdomains and older browsers.
func (s *Server) sameOrigin(next http.Handler) http.Handler {
	public, _ := url.Parse(s.Config.PublicURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			origin := r.Header.Get("Origin")
			site := r.Header.Get("Sec-Fetch-Site")
			allowed := false
			if origin != "" {
				o, err := url.Parse(origin)
				allowed = err == nil && o.Scheme == public.Scheme && o.Host == public.Host
			} else {
				// No Origin: not a modern browser's cross-site request (e.g. curl), unless the
				// browser says otherwise.
				allowed = site == "" || site == "same-origin" || site == "none"
			}
			if !allowed {
				s.errs(func(http.ResponseWriter, *http.Request) error { return errForbidden })(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) limitWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !s.writes.allow(s.clientIP(r)) {
			s.errs(func(http.ResponseWriter, *http.Request) error { return errTooMany })(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) clientIP(r *http.Request) string {
	if h := s.Config.ClientIPHeader; h != "" {
		if v := strings.TrimSpace(strings.Split(r.Header.Get(h), ",")[0]); v != "" {
			return v
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logRequests logs method, path (never the query, which can carry OAuth codes), status and time.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			return
		}
		s.Log.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.Log.Error("panic", "path", r.URL.Path, "panic", v)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cacheFor(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(d.Seconds())))
		next.ServeHTTP(w, r)
	})
}
