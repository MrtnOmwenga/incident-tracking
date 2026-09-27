package web

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/MrtnOmwenga/lighthouse/internal/analytics"
	"github.com/MrtnOmwenga/lighthouse/internal/auth"
)

// counted reports whether this request's visit may be counted: not when the browser asks not to
// be tracked (Global Privacy Control or Do Not Track), and not for the signed-in owner.
func counted(r *http.Request) bool {
	if r.Header.Get("Sec-GPC") == "1" || r.Header.Get("DNT") == "1" {
		return false
	}
	if id, ok := auth.FromContext(r.Context()); ok && id.Owner() {
		return false
	}
	return true
}

func validView(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// The collection endpoints answer 204 whether or not the hit was counted, so they reveal nothing
// about why a visit was skipped.

func (s *Server) analyticsView(w http.ResponseWriter, r *http.Request) {
	s.errs(func(w http.ResponseWriter, r *http.Request) error {
		var in struct {
			ID       string `json:"id"`
			Path     string `json:"path"`
			Ref      string `json:"ref"`
			Referrer string `json:"referrer"`
		}
		if err := readJSON(w, r, &in); err != nil {
			return err
		}
		if !validView(in.ID) {
			return invalid("id: a UUID for this page view.")
		}
		if counted(r) {
			if err := s.Analytics.View(r.Context(), analytics.Hit{
				ID: in.ID, Path: in.Path, Ref: in.Ref, Referrer: in.Referrer, IP: s.clientIP(r), UserAgent: r.UserAgent(),
			}); err != nil {
				return err
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})(w, r)
}

func (s *Server) analyticsPing(w http.ResponseWriter, r *http.Request) {
	s.errs(func(w http.ResponseWriter, r *http.Request) error {
		var in struct {
			ID string `json:"id"`
		}
		if err := readJSON(w, r, &in); err != nil {
			return err
		}
		if !validView(in.ID) {
			return invalid("id: a UUID for this page view.")
		}
		if counted(r) {
			if err := s.Analytics.Ping(r.Context(), in.ID); err != nil {
				return err
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})(w, r)
}

func (s *Server) analyticsEvent(w http.ResponseWriter, r *http.Request) {
	s.errs(func(w http.ResponseWriter, r *http.Request) error {
		var in struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := readJSON(w, r, &in); err != nil {
			return err
		}
		if !validView(in.ID) {
			return invalid("id: a UUID for this page view.")
		}
		if counted(r) {
			if err := s.Analytics.Event(r.Context(), in.ID, in.Name); err != nil {
				return err
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})(w, r)
}

// analyticsReport is the owner's private report: pages, projects, referral tags, referrers and
// devices over the last ?days= days (default 30, at most 90).
func (s *Server) analyticsReport(w http.ResponseWriter, r *http.Request, id auth.Identity) error {
	if !id.Owner() {
		return errForbidden
	}
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		n, err := strconv.Atoi(d)
		if err != nil || n < 1 || n > 90 {
			return invalid("days: 1 to 90.")
		}
		days = n
	}
	rep, err := s.Analytics.Report(r.Context(), days)
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	return writeJSON(w, http.StatusOK, rep)
}

func (s *Server) privacyPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	s.render(w, http.StatusOK, "privacy.html", pageData{Section: "privacy"})
}

// readershipRow is one project's public readership on the systems page.
type readershipRow struct {
	Name string
	analytics.Readership
}

func (s *Server) readership(r *http.Request) []readershipRow {
	rows, err := s.Analytics.Public(r.Context(), 30)
	if err != nil {
		s.Log.Warn("readership unavailable", "err", err)
		return nil
	}
	var out []readershipRow
	for _, row := range rows {
		if p, ok := s.Site.Find(row.Project); ok {
			out = append(out, readershipRow{Name: p.Name, Readership: row})
		}
	}
	return out
}
