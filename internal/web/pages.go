package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MrtnOmwenga/lighthouse/internal/hub"
	"github.com/MrtnOmwenga/lighthouse/internal/status"
	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

var funcs = template.FuncMap{
	"sparkline": func(m status.Monitor, now time.Time) template.HTML {
		return status.Sparkline(m.Latency, 24*time.Hour, now)
	},
	"pct": func(p *float64) string {
		if p == nil {
			return "–"
		}
		if *p == 100 {
			return "100%"
		}
		return fmt.Sprintf("%.2f%%", *p)
	},
	"ms": func(v *float64) string {
		if v == nil {
			return "–"
		}
		return fmt.Sprintf("%.0f ms", *v)
	},
	"when": func(t time.Time) string { return t.UTC().Format("2 Jan 2006, 15:04 UTC") },
	"ago":  ago,
	"duration": func(i store.Incident, now time.Time) string {
		end := now
		if i.ResolvedAt != nil {
			end = *i.ResolvedAt
		}
		return humanDuration(end.Sub(i.StartedAt))
	},
	"inc":   func(i int) int { return i + 1 },
	"words": func(s string) string { return strings.ReplaceAll(s, "_", " ") },
	// latest is the most recent update written for people (not a status or severity change).
	"latest": func(events []store.Event) *store.Event {
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].Kind == "comment" || events[i].Kind == "opened" {
				return &events[i]
			}
		}
		return nil
	},
}

func ago(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	return humanDuration(d) + " ago"
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "under a minute"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute")
	case d < 48*time.Hour:
		h, m := int(d.Hours()), int(d.Minutes())%60
		if m == 0 {
			return plural(h, "hour")
		}
		return plural(h, "hour") + " " + plural(m, "minute")
	default:
		return plural(int(d.Hours()/24), "day")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

type pageData struct {
	Owner   string
	Section string // highlighted in the navigation
	Now     time.Time
	Page    status.Page
	// incident page
	Incident *status.Incident
	// error page
	Title, Message string
	// hub and launch pages
	Catalog *hub.Catalog
	Cards   []projectCard
	Project *hub.Project
}

func (s *Server) render(w http.ResponseWriter, status int, name string, data pageData) {
	data.Owner = s.Config.OwnerName
	if data.Now.IsZero() {
		data.Now = s.Now()
	}
	var buf bytes.Buffer
	if err := s.pages.ExecuteTemplate(&buf, name, data); err != nil {
		s.Log.Error("rendering", "template", name, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func (s *Server) renderError(w http.ResponseWriter, code int, title, message string) {
	s.render(w, code, "error.html", pageData{Title: title, Message: message})
}

func (s *Server) statusPage(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	page, err := status.Build(r.Context(), s.Pool, s.OwnerTenant, now)
	if err != nil {
		s.Log.Error("status page", "err", err)
		s.renderError(w, http.StatusServiceUnavailable, "Status unavailable", "The status page couldn't be built just now. Please try again in a minute.")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=15")
	s.render(w, http.StatusOK, "status.html", pageData{Section: "status", Now: now, Page: page})
}

func (s *Server) publicStatus(w http.ResponseWriter, r *http.Request) {
	s.errs(func(w http.ResponseWriter, r *http.Request) error {
		page, err := status.Build(r.Context(), s.Pool, s.OwnerTenant, s.Now())
		if err != nil {
			return err
		}
		w.Header().Set("Cache-Control", "public, max-age=15")
		w.Header().Set("Access-Control-Allow-Origin", "*") // public data, readable by the hub and badges
		return writeJSON(w, http.StatusOK, page)
	})(w, r)
}

// incidentPage shows one public incident and its public timeline. Private incidents are
// indistinguishable from missing ones.
func (s *Server) incidentPage(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Not found", "There's no public incident here.")
		return
	}
	var view status.Incident
	err = store.WithTenant(r.Context(), s.Pool, s.OwnerTenant, func(tx pgx.Tx) error {
		inc, err := store.GetIncident(r.Context(), tx, id)
		if err != nil {
			return err
		}
		if !inc.Public {
			return store.ErrNotFound
		}
		events, err := store.Events(r.Context(), tx, id, true)
		view = status.Incident{Incident: inc, Events: events}
		return err
	})
	if err != nil {
		if err != store.ErrNotFound {
			s.Log.Error("incident page", "err", err)
		}
		s.renderError(w, http.StatusNotFound, "Not found", "There's no public incident here.")
		return
	}
	s.render(w, http.StatusOK, "incident.html", pageData{Section: "status", Incident: &view})
}
