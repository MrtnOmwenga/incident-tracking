package web_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/MrtnOmwenga/lighthouse/internal/config"
	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

const firefox = "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"

// hit posts to a collection endpoint the way a.js does, with the headers a browser would send.
func (b *browser) hit(path string, body any, headers map[string]string) int {
	b.e.t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", b.e.url+path, strings.NewReader(string(raw)))
	req.Header.Set("Origin", b.e.url)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", firefox)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode
}

type report struct {
	Pages []struct {
		Path     string `json:"path"`
		Views    int    `json:"views"`
		Visitors int    `json:"visitors"`
	} `json:"pages"`
	Projects []struct {
		Project string `json:"project"`
		Views   int    `json:"views"`
		Opens   int    `json:"opens"`
	} `json:"projects"`
	Refs []struct {
		Ref         string   `json:"ref"`
		Views       int      `json:"views"`
		Pages       []string `json:"pages"`
		DemosOpened int      `json:"demosOpened"`
	} `json:"refs"`
	Referrers []struct {
		Label    string `json:"label"`
		Visitors int    `json:"visitors"`
	} `json:"referrers"`
}

func (e *env) report() report {
	e.t.Helper()
	owner := e.browser()
	owner.expect(200, "POST", "/auth/dev", nil)
	return decode[report](e.t, owner.expect(200, "GET", "/api/analytics", nil))
}

func (e *env) views() int {
	e.t.Helper()
	var n int
	if err := e.db.Owner.QueryRow(context.Background(), "SELECT count(*) FROM page_views").Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestVisitsAreCountedWithTagsAndReferrers(t *testing.T) {
	t.Parallel()
	e := start(t, func(c *config.Config) { c.DevLogin = true })
	b := e.browser()
	first, second := uuid.NewString(), uuid.NewString()
	b.hit("/api/a/view", map[string]string{"id": first, "path": "/", "ref": "acme-backend", "referrer": "https://www.linkedin.com/in/x"}, nil)
	b.hit("/api/a/view", map[string]string{"id": second, "path": "/go/redacted"}, nil)
	b.hit("/api/a/event", map[string]string{"id": second, "name": "demo_open"}, nil)
	b.hit("/api/a/event", map[string]string{"id": second, "name": "demo_open"}, nil) // counted once

	rep := e.report()
	if len(rep.Refs) != 1 || rep.Refs[0].Ref != "acme-backend" || rep.Refs[0].Views != 2 || rep.Refs[0].DemosOpened != 1 {
		t.Fatalf("the tagged visitor's whole visit is attributed to the tag: %+v", rep.Refs)
	}
	if strings.Join(rep.Refs[0].Pages, ",") != "/,/go/redacted" {
		t.Fatalf("pages: %v", rep.Refs[0].Pages)
	}
	if len(rep.Projects) != 1 || rep.Projects[0].Project != "redacted" || rep.Projects[0].Opens != 1 {
		t.Fatalf("projects: %+v", rep.Projects)
	}
	found := false
	for _, r := range rep.Referrers {
		found = found || r.Label == "linkedin.com"
	}
	if !found {
		t.Fatalf("referrers keep the host only: %+v", rep.Referrers)
	}
	// Nothing identifying is stored: the visitor column is a 32-character pseudonym.
	var visitor string
	if err := e.db.Owner.QueryRow(context.Background(), "SELECT visitor FROM page_views LIMIT 1").Scan(&visitor); err != nil || len(visitor) != 32 || strings.Contains(visitor, "127.0.0.1") {
		t.Fatalf("visitor %q %v", visitor, err)
	}
}

func TestWhatIsNeverCounted(t *testing.T) {
	t.Parallel()
	e := start(t, func(c *config.Config) { c.DevLogin = true })
	anon := e.browser()
	cases := map[string]map[string]string{
		"global privacy control": {"Sec-GPC": "1"},
		"do not track":           {"DNT": "1"},
		"a bot":                  {"User-Agent": "Googlebot/2.1 (+http://www.google.com/bot.html)"},
		"no user agent":          {"User-Agent": ""},
	}
	for name, headers := range cases {
		if code := anon.hit("/api/a/view", map[string]string{"id": uuid.NewString(), "path": "/"}, headers); code != 204 {
			t.Errorf("%s: %d (always 204, whatever happens)", name, code)
		}
	}
	owner := e.browser()
	owner.expect(200, "POST", "/auth/dev", nil)
	owner.hit("/api/a/view", map[string]string{"id": uuid.NewString(), "path": "/"}, nil)
	anon.hit("/api/a/view", map[string]string{"id": uuid.NewString(), "path": "/admin/secret"}, nil) // not a page of this site
	if n := e.views(); n != 0 {
		t.Fatalf("%d views recorded that shouldn't have been", n)
	}
	if code := anon.hit("/api/a/view", map[string]string{"id": "not-a-uuid", "path": "/"}, nil); code != 422 {
		t.Fatalf("a malformed id: %d", code)
	}
	if code := anon.hit("/api/a/view", map[string]string{"id": uuid.NewString(), "path": "/"}, map[string]string{"Origin": "https://evil.example"}); code != 403 {
		t.Fatalf("another site can't post views: %d", code)
	}
}

// Engaged time is measured on the server, one capped heartbeat at a time, so a client can't claim
// more time than has passed.
func TestEngagedTimeIsCappedByTheServer(t *testing.T) {
	t.Parallel()
	e := start(t, nil)
	b := e.browser()
	id := uuid.NewString()
	b.hit("/api/a/view", map[string]string{"id": id, "path": "/projects/redacted"}, nil)
	for range 5 {
		b.hit("/api/a/ping", map[string]string{"id": id}, nil) // a burst of pings adds nothing
	}
	engaged := func() int {
		var s int
		if err := e.db.Owner.QueryRow(context.Background(), "SELECT engaged_seconds FROM page_views WHERE id = $1", id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if s := engaged(); s > 1 {
		t.Fatalf("pings in a burst credited %d s", s)
	}
	// Ten minutes pass with no heartbeat (the reader went away); the next ping adds at most the cap.
	if _, err := e.db.Owner.Exec(context.Background(), "UPDATE page_views SET last_ping = now() - interval '10 minutes' WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	b.hit("/api/a/ping", map[string]string{"id": id}, nil)
	if s := engaged(); s < store.MaxPingCredit-1 || s > store.MaxPingCredit+1 {
		t.Fatalf("after a long gap, one ping credited %d s, want about %d", s, store.MaxPingCredit)
	}
}

// Public readership hides any project read by fewer than five people.
func TestPublicReadershipHidesSmallNumbers(t *testing.T) {
	t.Parallel()
	e := start(t, nil, testSite(t, "http://127.0.0.1:1"))
	b := e.browser()
	visit := func(i int, path string) {
		b.hit("/api/a/view", map[string]string{"id": uuid.NewString(), "path": path},
			map[string]string{"User-Agent": fmt.Sprintf("%s reader-%d", firefox, i)}) // distinct visitors
	}
	for i := range 4 {
		visit(i, "/projects/redacted")
	}
	_, page := b.do("GET", "/status", nil)
	if !strings.Contains(page, "fewer than 5 readers") || strings.Contains(page, ">4<") {
		t.Fatalf("four readers must not be published:\n%s", page)
	}
	visit(4, "/projects/redacted")
	_, page = b.do("GET", "/status", nil)
	if strings.Contains(page, "fewer than 5 readers") || !strings.Contains(page, ">5<") {
		t.Fatal("five readers are published")
	}
}

func TestTheReportIsTheOwnersAlone(t *testing.T) {
	t.Parallel()
	e := start(t, nil)
	e.browser().expect(401, "GET", "/api/analytics", nil)
	visitor := e.browser()
	visitor.expect(201, "POST", "/api/sandbox", nil)
	visitor.expect(403, "GET", "/api/analytics", nil)
	visitor.hit("/api/a/view", map[string]string{"id": uuid.NewString(), "path": "/"}, nil)
	// A sandbox tenant can't read the owner's visit records, even directly.
	var n int
	if err := visitor.e.db.App.QueryRow(context.Background(), "SELECT count(*) FROM page_views").Scan(&n); err != nil || n != 0 {
		t.Fatalf("visible without the owner's tenant: %d %v", n, err)
	}
}

func TestSaltsRotateDaily(t *testing.T) {
	t.Parallel()
	e := start(t, nil)
	ctx := context.Background()
	day1 := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	a, err := store.DailySalt(ctx, e.db.App, day1)
	if err != nil || len(a) != 32 {
		t.Fatalf("salt %x %v", a, err)
	}
	again, _ := store.DailySalt(ctx, e.db.App, day1)
	b, _ := store.DailySalt(ctx, e.db.App, day1.AddDate(0, 0, 1))
	if string(a) != string(again) || string(a) == string(b) {
		t.Fatal("one salt per day, a new one each day")
	}
	var old int
	if err := e.db.Owner.QueryRow(ctx, "SELECT count(*) FROM analytics_salts WHERE day = $1", day1.Format(time.DateOnly)).Scan(&old); err != nil || old != 0 {
		t.Fatalf("yesterday's salt must be gone: %d %v", old, err)
	}
	if _, err := e.db.App.Exec(ctx, "SELECT salt FROM analytics_salts"); err == nil {
		t.Fatal("the app role must not read salts directly")
	}
}
