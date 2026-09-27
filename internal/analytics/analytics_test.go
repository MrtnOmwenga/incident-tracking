package analytics

import (
	"regexp"
	"testing"
)

func TestVisitorIsAPseudonymPerDay(t *testing.T) {
	day1, day2 := []byte("0123456789abcdef0123456789abcdef"), []byte("fedcba9876543210fedcba9876543210")
	a := Visitor(day1, "203.0.113.7", "Firefox")
	if a != Visitor(day1, "203.0.113.7", "Firefox") {
		t.Fatal("the same visitor on the same day must hash the same")
	}
	if a == Visitor(day2, "203.0.113.7", "Firefox") {
		t.Fatal("a new day's salt must make the visitor unrecognisable")
	}
	if a == Visitor(day1, "203.0.113.8", "Firefox") || a == Visitor(day1, "203.0.113.7", "Chrome") {
		t.Fatal("different visitors must hash differently")
	}
	// The separator stops "ab"+"c" from colliding with "a"+"bc".
	if Visitor(day1, "ab", "c") == Visitor(day1, "a", "bc") {
		t.Fatal("ambiguous concatenation")
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(a) {
		t.Fatalf("visitor %q doesn't fit the column", a)
	}
}

func TestBotsAndDevices(t *testing.T) {
	for ua, bot := range map[string]bool{
		"": true, "Googlebot/2.1": true, "Mozilla/5.0 HeadlessChrome/120": true, "curl/8.5": true, "Slackbot-LinkExpanding": true,
		"Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0": false,
	} {
		if IsBot(ua) != bot {
			t.Errorf("IsBot(%q) = %v", ua, !bot)
		}
	}
	for ua, want := range map[string]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile/15E148": "mobile",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) Mobile Safari/537.36":        "mobile",
		"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)":                        "tablet",
		"Mozilla/5.0 (Linux; Android 14; Xiaomi Pad 6) Safari/537.36":          "tablet",
		"Mozilla/5.0 (X11; Linux x86_64) Firefox/130.0":                        "desktop",
	} {
		if got := Device(ua); got != want {
			t.Errorf("Device(%q) = %s, want %s", ua, got, want)
		}
	}
}

func TestPages(t *testing.T) {
	for path, want := range map[string]string{
		"/": "/", "": "/", "/projects": "/projects", "/projects/": "/projects", "/projects/redacted": "/projects/redacted",
		"/go/ghostchat": "/go/ghostchat", "/status/incidents/7f0c": "/status/incidents", "/privacy": "/privacy",
	} {
		if got, _, ok := Page(path); !ok || got != want {
			t.Errorf("Page(%q) = %q %v, want %q", path, got, ok, want)
		}
	}
	for _, path := range []string{"/admin", "/projects/../etc", "/go/UPPER", "/projects/a/b", "/api/status", "//evil"} {
		if _, _, ok := Page(path); ok {
			t.Errorf("Page(%q) should be rejected", path)
		}
	}
	if _, p, _ := Page("/go/redacted"); p == nil || *p != "redacted" {
		t.Error("a launch page belongs to its project")
	}
}

func TestRefAndReferrer(t *testing.T) {
	if r := Ref(" Acme-Backend "); r == nil || *r != "acme-backend" {
		t.Errorf("Ref normalizes: %v", r)
	}
	for _, bad := range []string{"", "a b", "<script>", "x@y", "-leading"} {
		if Ref(bad) != nil {
			t.Errorf("Ref(%q) should be dropped", bad)
		}
	}
	r := New(nil, "", "https://www.omwenga.example")
	if h := r.Referrer("https://www.linkedin.com/feed/?trk=abc"); h == nil || *h != "linkedin.com" {
		t.Errorf("referrer keeps the host only: %v", h)
	}
	for _, same := range []string{"https://omwenga.example/projects", "https://www.omwenga.example/", "javascript:alert(1)", "", "not a url"} {
		if r.Referrer(same) != nil {
			t.Errorf("Referrer(%q) should be dropped", same)
		}
	}
}

var pathColumn = regexp.MustCompile(`^/[a-z0-9/_-]{0,120}$`)

// Whatever a browser (or anyone) sends, an accepted page fits the database column, so a crafted
// path can never turn into a server error.
func FuzzPage(f *testing.F) {
	for _, seed := range []string{"/", "/projects/redacted", "/go/x", "/status/incidents/abc", "/../", "/projects/%00"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, path string) {
		page, project, ok := Page(path)
		if !ok {
			return
		}
		if !pathColumn.MatchString(page) {
			t.Fatalf("Page(%q) accepted %q", path, page)
		}
		if project != nil && !slugPattern.MatchString(*project) {
			t.Fatalf("Page(%q) accepted project %q", path, *project)
		}
	})
}
