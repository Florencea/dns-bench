package web

import (
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Florencea/dns-bench/internal/config"
	"github.com/Florencea/dns-bench/internal/dns"
	"github.com/Florencea/dns-bench/internal/i18n"
)

const (
	maxFormBodyBytes = 64 << 10 // 64 KiB
	localeCookieAge  = 365 * 24 * 3600
)

// Server wires HTTP routes, security middleware, templates, and the DNS engine.
type Server struct {
	cfg       config.Config
	engine    *dns.Engine
	templates *TemplateSet
	handler   http.Handler
}

// NewServer initializes routes, templates, CSRF protection, and security middleware.
func NewServer(cfg config.Config, engine *dns.Engine) (*Server, error) {
	ts, err := NewTemplateSet()
	if err != nil {
		return nil, err
	}
	staticFS, err := StaticFS()
	if err != nil {
		return nil, err
	}
	if engine == nil {
		engine = dns.NewEngine(cfg.DNSTimeout, cfg.DoHTimeout)
	}

	s := &Server{
		cfg:       cfg,
		engine:    engine,
		templates: ts,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux, staticFS)

	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(s.renderCSRFDeny))

	s.handler = RecoverAndLogMiddleware(
		SecurityHeadersMiddleware(
			LoopbackHostMiddleware(cfg.DefaultLocale, csrf.Handler(mux)),
		),
	)
	return s, nil
}

// Handler returns the complete hardened HTTP handler chain.
func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) registerRoutes(mux *http.ServeMux, staticFS fs.FS) {
	fileServer := http.StripPrefix("/static/", http.FileServerFS(staticFS))

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fileServer.ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /locale", s.handleLocalePost)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	locale := i18n.ResolveLocale(r, s.cfg.DefaultLocale)
	t := i18n.Catalog(locale)
	switchLocale := i18n.LocaleEnUS
	if locale == i18n.LocaleEnUS {
		switchLocale = i18n.LocaleZhTW
	}

	q := r.URL.Query()
	mode := dns.ParseMode(q.Get("mode"))
	hasHostParam := q.Has("host")
	rawHost := q.Get("host")

	currentURI := buildCanonicalURI(mode, rawHost, hasHostParam)

	page := IndexPage{
		Lang:         locale,
		SwitchLocale: switchLocale,
		CurrentURI:   currentURI,
		T:            t,
		Mode:         string(mode),
		HostInput:    strings.TrimSpace(rawHost),
	}

	if hasHostParam {
		if !isAllowedQueryFetch(r) {
			http.Error(w, t.Errors.CrossSiteQueryRejected, http.StatusForbidden)
			return
		}

		domain, err := dns.ValidateDomain(rawHost)
		if err != nil {
			page.ErrorMessage = t.UI.DomainInvalid
			page.Cards = buildIdleCards(s.engine.FilteredServers(mode))
			s.templates.RenderIndex(w, http.StatusBadRequest, page)
			return
		}

		start := time.Now()
		results, qerr := s.engine.QueryAll(r.Context(), mode, domain)
		if qerr != nil {
			page.ErrorMessage = t.UI.DomainInvalid
			page.Cards = buildIdleCards(s.engine.FilteredServers(mode))
			s.templates.RenderIndex(w, http.StatusBadRequest, page)
			return
		}

		cards := make([]ServerCardView, len(results))
		for i, res := range results {
			cards[i] = ServerCardView{
				Index:  i,
				Server: res.Server,
				Output: res.Output,
			}
		}
		page.HostInput = domain
		page.QueriedHost = domain
		page.HasResult = true
		page.TotalDurationMS = time.Since(start).Milliseconds()
		page.Cards = cards
		s.templates.RenderIndex(w, http.StatusOK, page)
		return
	}

	page.Cards = buildIdleCards(s.engine.FilteredServers(mode))
	s.templates.RenderIndex(w, http.StatusOK, page)
}

func (s *Server) handleLocalePost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)
	if err := r.ParseForm(); err != nil {
		t := i18n.Catalog(i18n.ResolveLocale(r, s.cfg.DefaultLocale))
		http.Error(w, t.Errors.InvalidRequest, http.StatusBadRequest)
		return
	}

	if locale, ok := i18n.ParseLocale(r.FormValue("locale")); ok {
		http.SetCookie(w, &http.Cookie{
			Name:     i18n.LocaleCookieName,
			Value:    locale,
			Path:     "/",
			MaxAge:   localeCookieAge,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	target := SafeRedirectPath(r.FormValue("next"))
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) renderCSRFDeny(w http.ResponseWriter, r *http.Request) {
	t := i18n.Catalog(i18n.ResolveLocale(r, s.cfg.DefaultLocale))
	http.Error(w, t.Errors.CSRFRejected, http.StatusForbidden)
}

func buildIdleCards(servers []dns.Server) []ServerCardView {
	cards := make([]ServerCardView, len(servers))
	for i, srv := range servers {
		cards[i] = ServerCardView{
			Index:  i,
			Server: srv,
		}
	}
	return cards
}

func buildCanonicalURI(mode dns.Mode, rawHost string, hasHost bool) string {
	v := url.Values{}
	v.Set("mode", string(mode))
	if hasHost && strings.TrimSpace(rawHost) != "" {
		v.Set("host", strings.TrimSpace(rawHost))
	}
	return "/?" + v.Encode()
}

// isAllowedQueryFetch blocks cross-site or non-document GET requests from triggering local DNS queries.
func isAllowedQueryFetch(r *http.Request) bool {
	site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")))
	if site != "" && site != "none" && site != "same-origin" {
		return false
	}
	dest := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Dest")))
	if dest != "" && dest != "document" {
		return false
	}
	return true
}

// SafeRedirectPath validates redirect targets against open redirects while preserving query params.
func SafeRedirectPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, `/\`) {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "/"
	}
	return u.RequestURI()
}
