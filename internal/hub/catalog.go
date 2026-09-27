// Package hub is the portfolio front door: the list of projects, and the launch page that
// introduces a demo while it wakes up.
package hub

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// Catalog is read from a YAML file (PROJECTS_FILE) that the owner maintains. It is trusted
// configuration, not user input, but it is still validated so a typo fails at startup rather than
// on the page.
type Catalog struct {
	Headline string    `yaml:"headline"`
	About    string    `yaml:"about"`
	Links    []Link    `yaml:"links"`
	Projects []Project `yaml:"projects"`
}

type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

type Project struct {
	Slug        string   `yaml:"slug"`
	Name        string   `yaml:"name"`
	Tagline     string   `yaml:"tagline"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Repo        string   `yaml:"repo"`
	// Demo is where the live demo is. Empty for projects without one (a CLI, a device app).
	Demo string `yaml:"demo"`
	// Health is polled until it answers 2xx/3xx, which also wakes a demo that sleeps when idle.
	// Usually an address inside the cluster. Empty: the demo is always ready.
	Health string `yaml:"health"`
	// Monitor is the slug of the Lighthouse monitor whose status the card shows.
	Monitor string `yaml:"monitor"`
	// Tour, when set, is offered at the end of the introduction as a guided alternative to
	// opening the demo directly.
	Tour string `yaml:"tour"`
	// NoDemoNote explains, for projects without a demo, how to try them instead.
	NoDemoNote string  `yaml:"no_demo_note"`
	Intro      []Slide `yaml:"intro"`
}

type Slide struct {
	Title string `yaml:"title"`
	Body  string `yaml:"body"`
}

func (c Catalog) Find(slug string) (Project, bool) {
	for _, p := range c.Projects {
		if p.Slug == slug {
			return p, true
		}
	}
	return Project{}, false
}

// Load reads and validates a catalog. An empty path is an empty catalog.
func Load(path string) (Catalog, error) {
	if path == "" {
		return Catalog{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	return Parse(raw)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`)

// Parse decodes a catalog strictly (unknown keys are errors) and reports every problem at once.
func Parse(raw []byte) (Catalog, error) {
	var c Catalog
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) { // an empty file is an empty catalog
		return Catalog{}, fmt.Errorf("projects file: %w", err)
	}
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	length := func(what, v string, max int, required bool) {
		n := utf8.RuneCountInString(strings.TrimSpace(v))
		if (required && n == 0) || n > max {
			add("%s: required, at most %d characters", what, max)
		}
	}
	link := func(what, v string, allowRelative bool) {
		if v == "" {
			return
		}
		if allowRelative && strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "//") {
			return
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			add("%s: an absolute http(s) URL without credentials", what)
		}
	}

	length("headline", c.Headline, 120, false)
	length("about", c.About, 600, false)
	for i, l := range c.Links {
		length(fmt.Sprintf("links[%d].label", i), l.Label, 30, true)
		if !strings.HasPrefix(l.URL, "mailto:") {
			link(fmt.Sprintf("links[%d].url", i), l.URL, false)
		}
		if l.URL == "" {
			add("links[%d].url: required", i)
		}
	}
	seen := map[string]bool{}
	for i, p := range c.Projects {
		where := fmt.Sprintf("projects[%d]", i)
		if p.Slug != "" {
			where = "project " + p.Slug
		}
		if !slugPattern.MatchString(p.Slug) {
			add("%s: slug must be lowercase letters, digits and dashes", where)
		}
		if seen[p.Slug] {
			add("%s: duplicate slug", where)
		}
		seen[p.Slug] = true
		length(where+": name", p.Name, 60, true)
		length(where+": tagline", p.Tagline, 140, true)
		length(where+": description", p.Description, 600, false)
		length(where+": no_demo_note", p.NoDemoNote, 200, false)
		if len(p.Tags) > 8 {
			add("%s: at most 8 tags", where)
		}
		for _, t := range p.Tags {
			length(where+": tag", t, 24, true)
		}
		link(where+": repo", p.Repo, false)
		link(where+": demo", p.Demo, true)
		link(where+": health", p.Health, false)
		link(where+": tour", p.Tour, true)
		if p.Monitor != "" && !slugPattern.MatchString(p.Monitor) {
			add("%s: monitor must be a monitor's slug", where)
		}
		if p.Demo == "" && (p.Health != "" || p.Tour != "" || len(p.Intro) > 0) {
			add("%s: health, tour and intro need a demo", where)
		}
		if p.Demo != "" && len(p.Intro) == 0 {
			add("%s: a demo needs an intro (1 to 6 slides) to show while it starts", where)
		}
		if len(p.Intro) > 6 {
			add("%s: at most 6 intro slides", where)
		}
		for j, s := range p.Intro {
			length(fmt.Sprintf("%s: intro[%d].title", where, j), s.Title, 80, true)
			length(fmt.Sprintf("%s: intro[%d].body", where, j), s.Body, 400, true)
		}
	}
	if len(problems) > 0 {
		return Catalog{}, errors.New("projects file:\n  " + strings.Join(problems, "\n  "))
	}
	return c, nil
}
