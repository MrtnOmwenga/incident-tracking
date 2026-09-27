package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MrtnOmwenga/lighthouse/internal/monitor"
)

func TestTheShippedCatalogIsValid(t *testing.T) {
	c, err := Load("../../deploy/projects.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Projects) == 0 {
		t.Fatal("no projects")
	}
	if _, ok := c.Find("redacted"); !ok {
		t.Fatal("redacted missing")
	}
}

func TestParse(t *testing.T) {
	if c, err := Parse(nil); err != nil || len(c.Projects) != 0 {
		t.Fatalf("an empty file is an empty catalog: %v", err)
	}
	if _, err := Parse([]byte("projects:\n  - slug: x\n    nmae: typo\n")); err == nil || !strings.Contains(err.Error(), "nmae") {
		t.Fatalf("unknown keys are errors: %v", err)
	}

	_, err := Parse([]byte(`
links: [{label: Mail, url: "mailto:me@example.com"}, {label: Bad, url: "javascript:alert(1)"}]
projects:
  - slug: Bad Slug
    name: ""
    tagline: fine
    demo: "//evil.example"
  - slug: dup
    name: A
    tagline: t
    demo: https://a.example
  - slug: dup
    name: B
    tagline: t
    health: http://b.internal/health
`))
	if err == nil {
		t.Fatal("accepted an invalid catalog")
	}
	for _, want := range []string{
		"links[1].url", "slug must be", "name: required", "project Bad Slug: demo", "duplicate slug",
		"project dup: a demo needs an intro", "health, tour and intro need a demo",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "links[0]") {
		t.Error("mailto links are allowed")
	}
}

func TestReadinessSharesProbesAndCaches(t *testing.T) {
	var hits atomic.Int32
	var up atomic.Bool
	demo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		time.Sleep(50 * time.Millisecond)
		if !up.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer demo.Close()
	r := NewReadiness(monitor.NewProber())
	r.TTL = 200 * time.Millisecond
	p := Project{Slug: "demo", Health: demo.URL}

	var wg sync.WaitGroup
	for range 25 {
		wg.Go(func() {
			if r.Ready(context.Background(), p) {
				t.Error("a 503 isn't ready")
			}
		})
	}
	wg.Wait()
	if n := hits.Load(); n != 1 {
		t.Fatalf("25 simultaneous visitors caused %d probes, want 1", n)
	}
	up.Store(true)
	if r.Ready(context.Background(), p) {
		t.Fatal("the cached answer holds until it expires")
	}
	time.Sleep(250 * time.Millisecond)
	if !r.Ready(context.Background(), p) || hits.Load() != 2 {
		t.Fatalf("after the TTL it asks again: hits %d", hits.Load())
	}
	if !r.Ready(context.Background(), Project{Slug: "static"}) {
		t.Fatal("a demo without a health address is always ready")
	}
}
