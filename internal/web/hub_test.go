package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MrtnOmwenga/lighthouse/internal/hub"
	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

func TestHub(t *testing.T) {
	t.Parallel()
	var up atomic.Bool
	demo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !up.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer demo.Close()
	catalog, err := hub.Parse([]byte(`
headline: Builds things.
projects:
  - slug: redacted
    name: Redacted
    tagline: Words black out.
    tags: [NestJS]
    repo: https://github.com/example/redacted
    demo: https://redacted.example
    health: ` + demo.URL + `/internal-health
    monitor: redacted
    intro:
      - { title: A briefing room, body: Three screens at once. }
      - { title: Live permissions, body: Demote someone mid-sentence. }
  - slug: tool
    name: Tool
    tagline: Runs locally.
    no_demo_note: Install it yourself.
`))
	if err != nil {
		t.Fatal(err)
	}
	e := start(t, nil, catalog)
	e.ownerData(func(tx pgx.Tx) error {
		m, err := store.CreateMonitor(context.Background(), tx, e.owner, store.MonitorInput{Name: "Redacted", Slug: "redacted", Kind: "simulated",
			SimulatedMode: "up", IntervalSeconds: 60, TimeoutMS: 1000, ExpectedStatusMin: 200, ExpectedStatusMax: 299,
			FailureThreshold: 1, RecoveryThreshold: 1, Public: true})
		if err != nil {
			return err
		}
		m.Health = "up"
		return store.SaveMonitorState(context.Background(), tx, m)
	})
	b := e.browser()

	_, page := b.do("GET", "/", nil)
	for _, want := range []string{"Builds things.", "Words black out.", `href="/go/redacted"`, "Live", "Install it yourself."} {
		if !strings.Contains(page, want) {
			t.Errorf("hub page lacks %q", want)
		}
	}
	if strings.Contains(page, `href="/go/tool"`) {
		t.Error("a project without a demo has no launch link")
	}

	_, launch := b.do("GET", "/go/redacted", nil)
	for _, want := range []string{"Starting Redacted…", "A briefing room", "Live permissions", `data-demo="https://redacted.example"`, "/static/launch.js"} {
		if !strings.Contains(launch, want) {
			t.Errorf("launch page lacks %q", want)
		}
	}
	for _, path := range []string{"/go/tool", "/go/nope"} {
		if resp, _ := b.do("GET", path, nil); resp.StatusCode != 404 {
			t.Errorf("%s: %d", path, resp.StatusCode)
		}
	}

	// The health address is internal: it never appears in public output.
	_, api := b.do("GET", "/api/projects", nil)
	for _, body := range []string{page, launch, api} {
		if strings.Contains(body, "internal-health") {
			t.Error("public output reveals the health address")
		}
	}
	if !strings.Contains(api, `"launch":"/go/redacted"`) {
		t.Errorf("api: %s", api)
	}

	if body := b.expect(200, "GET", "/api/projects/redacted/ready", nil); !strings.Contains(body, `"ready":false`) {
		t.Fatalf("a demo answering 503 isn't ready: %s", body)
	}
	up.Store(true)
	e.waitReady(b, "redacted")
	b.expect(404, "GET", "/api/projects/tool/ready", nil)
}

func (e *env) waitReady(b *browser, slug string) {
	e.t.Helper()
	for range 50 {
		if strings.Contains(b.expect(200, "GET", "/api/projects/"+slug+"/ready", nil), `"ready":true`) {
			return
		}
		sleep()
	}
	e.t.Fatal("never became ready")
}

func sleep() { <-time.After(100 * time.Millisecond) }
