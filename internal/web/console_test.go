package web_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestConsoleIsServedWithClientSideRoutes(t *testing.T) {
	t.Parallel()
	e := start(t, nil)
	b := e.browser()
	resp, index := b.do("GET", "/console/", nil)
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("/console/: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	// A client-side route loads the same page, so reloading /console/incidents/… works.
	for _, path := range []string{"/console/monitors", "/console/incidents/0b9e7c4e-0000-0000-0000-000000000000"} {
		if resp, body := b.do("GET", path, nil); resp.StatusCode != 200 || body != index {
			t.Errorf("%s: %d, not the console's page", path, resp.StatusCode)
		}
	}
	if resp, _ := b.do("GET", "/console/assets/missing.js", nil); resp.StatusCode != 404 {
		t.Errorf("a missing asset: %d, want 404", resp.StatusCode)
	}
	if resp, _ := b.do("GET", "/console", nil); resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("/console: %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "script-src 'self'") {
		t.Error("the console gets the same strict CSP")
	}
}

func TestSessionAndSignInOptions(t *testing.T) {
	t.Parallel()
	e := start(t, nil)
	b := e.browser()
	if got := decode[map[string]any](t, b.expect(200, "GET", "/api/session", nil)); got["signedIn"] != false {
		t.Fatalf("signed out: %v", got)
	}
	opts := decode[map[string]bool](t, b.expect(200, "GET", "/api/sign-in-options", nil))
	if !opts["github"] || opts["dev"] || !opts["sandbox"] {
		t.Fatalf("options: %v", opts)
	}
	b.expect(201, "POST", "/api/sandbox", nil)
	got := decode[map[string]any](t, b.expect(200, "GET", "/api/session", nil))
	if got["signedIn"] != true || got["role"] != "sandbox" || got["expiresAt"] == nil {
		t.Fatalf("signed in: %v", got)
	}
}
