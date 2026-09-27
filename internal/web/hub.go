package web

import (
	"net/http"

	"github.com/MrtnOmwenga/lighthouse/internal/hub"
	"github.com/MrtnOmwenga/lighthouse/internal/status"
)

// projectCard is a project as the hub shows it, with its live status when it has a monitor.
type projectCard struct {
	hub.Project
	Health    string   // up, down, unknown; empty without a public monitor
	Uptime90d *float64 // percent
}

func (s *Server) hubPage(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	// Status is a nicety on the hub: if it can't be built, the projects still show.
	page, err := status.Build(r.Context(), s.Pool, s.OwnerTenant, now)
	if err != nil {
		s.Log.Warn("hub: status unavailable", "err", err)
	}
	monitors := map[string]status.Monitor{}
	for _, m := range page.Monitors {
		monitors[m.Slug] = m
	}
	cards := make([]projectCard, len(s.Catalog.Projects))
	for i, p := range s.Catalog.Projects {
		cards[i] = projectCard{Project: p}
		if m, ok := monitors[p.Monitor]; ok && p.Monitor != "" {
			cards[i].Health, cards[i].Uptime90d = m.Health, m.Uptime90d
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=30")
	s.render(w, http.StatusOK, "hub.html", pageData{Section: "projects", Now: now, Catalog: &s.Catalog, Cards: cards, Page: page})
}

// launchPage introduces a demo while it starts, then opens it (or offers a guided tour).
func (s *Server) launchPage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.Catalog.Find(r.PathValue("slug"))
	if !ok || p.Demo == "" {
		s.renderError(w, http.StatusNotFound, "Not found", "There's no demo by that name.")
		return
	}
	s.render(w, http.StatusOK, "launch.html", pageData{Section: "projects", Project: &p})
}

// publicProject is what the API says about a project: never its health address, which usually
// points inside the cluster.
type publicProject struct {
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Tagline     string      `json:"tagline"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags"`
	Repo        string      `json:"repo,omitempty"`
	Demo        string      `json:"demo,omitempty"`
	Launch      string      `json:"launch,omitempty"` // the launch page, when there is a demo
	Tour        string      `json:"tour,omitempty"`
	Intro       []hub.Slide `json:"intro,omitempty"`
}

func (s *Server) listProjects(w http.ResponseWriter, _ *http.Request) {
	out := make([]publicProject, len(s.Catalog.Projects))
	for i, p := range s.Catalog.Projects {
		out[i] = publicProject{Slug: p.Slug, Name: p.Name, Tagline: p.Tagline, Description: p.Description,
			Tags: p.Tags, Repo: p.Repo, Demo: p.Demo, Tour: p.Tour, Intro: p.Intro}
		if out[i].Tags == nil {
			out[i].Tags = []string{}
		}
		if p.Demo != "" {
			out[i].Launch = "/go/" + p.Slug
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = writeJSON(w, http.StatusOK, out)
}

// projectReady reports whether a demo is up, waking it if it sleeps. Launch pages poll it.
func (s *Server) projectReady(w http.ResponseWriter, r *http.Request) {
	s.errs(func(w http.ResponseWriter, r *http.Request) error {
		p, ok := s.Catalog.Find(r.PathValue("slug"))
		if !ok || p.Demo == "" {
			return errNotFound
		}
		w.Header().Set("Cache-Control", "no-store")
		return writeJSON(w, http.StatusOK, map[string]bool{"ready": s.Readiness.Ready(r.Context(), p)})
	})(w, r)
}
