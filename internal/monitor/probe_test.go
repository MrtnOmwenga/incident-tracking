package monitor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func target(url string) Target {
	return Target{URL: url, Timeout: 2 * time.Second, StatusMin: 200, StatusMax: 399, AllowPrivate: true}
}

func TestHTTPChecks(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("<h1>all systems go</h1>"))
		case "/broken":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/slow":
			time.Sleep(300 * time.Millisecond)
		case "/redirect-loop":
			http.Redirect(w, r, "/redirect-loop", http.StatusFound)
		}
	}))
	defer site.Close()
	p := NewProber()
	ctx := context.Background()

	if r := p.HTTP(ctx, target(site.URL+"/ok")); !r.OK || r.StatusCode != 200 || r.Latency <= 0 {
		t.Fatalf("a healthy page should pass: %+v", r)
	}
	if r := p.HTTP(ctx, target(site.URL+"/broken")); r.OK || r.Failure != FailStatus || r.StatusCode != 503 {
		t.Fatalf("a 503 should fail on status: %+v", r)
	}
	wanted := target(site.URL + "/ok")
	wanted.ExpectedText = "all systems go"
	if r := p.HTTP(ctx, wanted); !r.OK {
		t.Fatalf("the expected text is there: %+v", r)
	}
	wanted.ExpectedText = "maintenance"
	if r := p.HTTP(ctx, wanted); r.OK || r.Failure != FailContent {
		t.Fatalf("missing text should fail on content: %+v", r)
	}
	slow := target(site.URL + "/slow")
	slow.Timeout = 50 * time.Millisecond
	if r := p.HTTP(ctx, slow); r.OK || r.Failure != FailTimeout {
		t.Fatalf("a slow page past the timeout should fail as a timeout: %+v", r)
	}
	if r := p.HTTP(ctx, target(site.URL+"/redirect-loop")); r.OK || r.Failure != FailStatus {
		t.Fatalf("redirects stop after five hops: %+v", r)
	}
	if r := p.HTTP(ctx, target("http://127.0.0.1:1/")); r.OK || r.Failure != FailConnection {
		t.Fatalf("a closed port is a connection failure: %+v", r)
	}
}

func TestPrivateAddressesAreBlockedUnlessAllowed(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer site.Close()
	p := NewProber()
	blocked := target(site.URL)
	blocked.AllowPrivate = false
	if r := p.HTTP(context.Background(), blocked); r.OK || r.Failure != FailBlocked {
		t.Fatalf("a loopback target must be blocked: %+v", r)
	}
	// The hostname resolves to loopback: the check happens on the resolved address.
	byName := target(strings.Replace(site.URL, "127.0.0.1", "localhost", 1))
	byName.AllowPrivate = false
	if r := p.HTTP(context.Background(), byName); r.Failure != FailBlocked {
		t.Fatalf("a name resolving to loopback must be blocked too: %+v", r)
	}
}

// A permitted page that redirects to a forbidden address is caught at the redirect, because the
// guard runs on every dial. Test servers all live on loopback, so here 127.0.0.1 plays the public
// internet and 127.0.0.2 the internal network.
func TestRedirectToBlockedAddressIsBlocked(t *testing.T) {
	internal := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("secret"))
	}))
	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("127.0.0.2 isn't available here: %v", err)
	}
	internal.Listener = listener
	internal.Start()
	defer internal.Close()
	redirector := httptest.NewServer(http.RedirectHandler(internal.URL, http.StatusFound))
	defer redirector.Close()

	p := newProber(func(ip netip.Addr) bool { return ip == netip.MustParseAddr("127.0.0.2") })
	direct := target(redirector.URL)
	direct.AllowPrivate = false
	direct.ExpectedText = "secret"
	if r := p.HTTP(context.Background(), direct); r.OK || r.Failure != FailBlocked {
		t.Fatalf("following a redirect into the internal network must be blocked: %+v", r)
	}
	// Control: without the redirect, the "public" server is reachable through the same guard.
	plain := target(redirector.URL + "/")
	plain.AllowPrivate = false
	plain.StatusMax = 302
	noFollow := newProber(func(netip.Addr) bool { return false })
	if r := noFollow.HTTP(context.Background(), plain); !r.OK {
		t.Fatalf("with nothing blocked the chain succeeds: %+v", r)
	}
}

func TestBlockedClassification(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1": true, "10.1.2.3": true, "172.16.0.9": true, "192.168.1.1": true, "169.254.169.254": true,
		"100.64.0.1": true, "0.0.0.0": true, "::1": true, "fd00::1": true, "fe80::1": true, "::ffff:10.0.0.1": true,
		"8.8.8.8": false, "1.1.1.1": false, "2606:4700:4700::1111": false, "100.128.0.1": false,
	} {
		if got := Blocked(netip.MustParseAddr(addr)); got != want {
			t.Errorf("Blocked(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestTLSExpiryIsRecorded(t *testing.T) {
	site := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer site.Close()
	p := NewProber()
	p.private.Transport = site.Client().Transport // trust the test certificate
	r := p.HTTP(context.Background(), target(site.URL))
	if !r.OK || r.TLSExpiresAt == nil || r.TLSExpiresAt.Before(time.Now()) {
		t.Fatalf("expected a passing check with the certificate's expiry: %+v", r)
	}
	p2 := NewProber()
	if r := p2.HTTP(context.Background(), target(site.URL)); r.OK || r.Failure != FailTLS {
		t.Fatalf("an untrusted certificate is a TLS failure: %+v", r)
	}
}

func TestSimulated(t *testing.T) {
	p := NewProber()
	for mode, ok := range map[string]bool{"up": true, "slow": true, "down": false} {
		if r := p.Simulated(mode); r.OK != ok {
			t.Errorf("mode %s: ok = %v", mode, r.OK)
		}
	}
	if r := p.Simulated("slow"); r.Latency < 1500*time.Millisecond {
		t.Errorf("slow should be slow: %v", r.Latency)
	}
	p.rand = func() float64 { return 0.1 }
	if r := p.Simulated("flaky"); r.OK {
		t.Error("flaky fails when the dice say so")
	}
	p.rand = func() float64 { return 0.9 }
	if r := p.Simulated("flaky"); !r.OK {
		t.Error("flaky passes otherwise")
	}
}
