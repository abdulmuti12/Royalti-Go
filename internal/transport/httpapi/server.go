package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/application"
	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/abdulmuti12/api-point-go/internal/platform"
)

type API struct {
	Service *application.Service
	Config  platform.Config
	Logger  *slog.Logger
	limiter *limiter
}
type inputKey struct{}

func New(s *application.Service, c platform.Config) http.Handler {
	a := &API{Service: s, Config: c, Logger: slog.Default(), limiter: &limiter{buckets: map[string]bucket{}}}
	mux := http.NewServeMux()
	for _, route := range Routes {
		route := route
		mux.HandleFunc(route.Method+" "+route.Path, func(w http.ResponseWriter, r *http.Request) { a.handle(w, r, route) })
	}
	mux.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, domain.Record{"status": "ok"}) })
	mux.Handle("GET /storage/", http.StripPrefix("/storage/", http.FileServer(http.Dir(c.Storage))))
	return a.middleware(mux)
}
func (a *API) handle(w http.ResponseWriter, r *http.Request, route Route) {
	if route.Action == "check" {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte("ok"))
		return
	}
	if route.Limit > 0 {
		host, _, e := net.SplitHostPort(r.RemoteAddr)
		if e != nil {
			host = r.RemoteAddr
		}
		limit := route.Limit
		// Allow overriding the per-route limit at runtime via RATE_LIMIT_MAX,
		// so the threshold can be raised without a rebuild when the default
		// (e.g. 8 logins / 60s) is too tight for dev / demo usage.
		if envLimit, err := strconv.Atoi(os.Getenv("RATE_LIMIT_MAX")); err == nil && envLimit > 0 {
			limit = envLimit
		}
		if !a.limiter.allow(host+":"+route.Path, limit, time.Duration(route.Minutes)*time.Minute) {
			w.Header().Set("Retry-After", "3600")
			writeJSON(w, 429, domain.Record{"message": "Too Many Attempts."})
			return
		}
	}
	ctx := r.Context()
	var principal domain.Principal
	var e error
	if route.Guard != "" {
		raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if raw == "" {
			failure(w, route, domain.Fail("Unauthorized", "Token not provided"))
			return
		}
		principal, e = a.Service.Authenticate(ctx, raw, route.Guard)
		if e != nil {
			failure(w, route, e)
			return
		}
		if e = a.Service.Authorize(ctx, principal, route.Menu); e != nil {
			failure(w, route, e)
			return
		}
	}
	in, _ := ctx.Value(inputKey{}).(domain.Record)
	if in == nil {
		in = domain.Record{}
	}
	if r.Method != "GET" && r.Method != "HEAD" && hasLegacySanitizer(route) {
		sanitizeLegacy(in)
	}
	if route.Table == "category" && route.Path == "/api/admins/get-categories" || route.Table == "brands" && route.Path == "/api/admins/get-brands" {
		in["sorted"] = true
	}
	saved, e := a.uploads(r, route, in)
	if e != nil {
		a.removeFiles(saved)
		failure(w, route, e)
		return
	}
	retired, e := a.retiredFiles(r, route, in)
	if e != nil {
		a.removeFiles(saved)
		failure(w, route, e)
		return
	}
	result, e := a.Service.Execute(ctx, domain.Command{Action: route.Action, Table: route.Table, ID: r.PathValue("id"), Input: in, Principal: principal})
	if e != nil {
		a.removeFiles(saved)
		failure(w, route, e)
		return
	}
	a.removeFiles(retired)
	respond(w, r, route, result, a.Config.AppURL)
}
func (a *API) middleware(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, origin := range a.Config.Origins {
		allowed[origin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.Logger.Error("request panic", "path", r.URL.Path, "panic", recovered)
				writeJSON(w, 500, domain.Record{"success": false, "message": "Terjadi kesalahan pada server", "status": 500})
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == "OPTIONS" {
			if origin != "" && !allowed[origin] {
				writeJSON(w, 403, domain.Record{"success": false, "message": "Forbidden: Invalid Origin or Referer.", "status": 403})
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Origin, Referer, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.WriteHeader(204)
			return
		}
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" || r.Method == "DELETE" {
			referer := r.Header.Get("Referer")
			ok := origin == "" && referer == "" || allowed[origin]
			if !ok && referer != "" {
				if u, e := url.Parse(referer); e == nil {
					ok = allowed[u.Scheme+"://"+u.Host]
				}
			}
			if !ok {
				writeJSON(w, 403, domain.Record{"success": false, "message": "Forbidden: Invalid Origin or Referer.", "status": 403})
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			in, e := decodeInput(w, r)
			if e != nil {
				var tooBig *http.MaxBytesError
				if errors.As(e, &tooBig) {
					writeJSON(w, 413, domain.Record{"message": "Request entity too large"})
				} else {
					writeJSON(w, 400, domain.Record{"message": "Invalid request body"})
				}
				return
			}
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
			if r.Method == "POST" {
				method := strings.ToUpper(domain.Text(in["_method"]))
				if method == "PUT" || method == "PATCH" || method == "DELETE" {
					r.Method = method
				}
			}
			delete(in, "_method")
			r = r.WithContext(context.WithValue(r.Context(), inputKey{}, in))
		}
		next.ServeHTTP(w, r)
	})
}

type bucket struct {
	Count int
	Until time.Time
}
type limiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
}

func (l *limiter) allow(key string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	// Prune expired buckets on every call so the map stays bounded and
	// stale keys can't push us over the 10000-key cap that would make
	// *new* requests fail open-closed unexpectedly.
	for k, b := range l.buckets {
		if now.After(b.Until) {
			delete(l.buckets, k)
		}
	}
	b := l.buckets[key]
	if now.After(b.Until) {
		b = bucket{Until: now.Add(window)}
	}
	if b.Count >= limit {
		return false
	}
	b.Count++
	l.buckets[key] = b
	return true
}
