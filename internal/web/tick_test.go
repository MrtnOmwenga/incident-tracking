package web

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MrtnOmwenga/lighthouse/internal/oidc"
)

func TestTickRunsOnlyForTheScheduler(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "k", "kty": "RSA",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer keys.Close()

	const aud, caller = "https://lh.example/internal/tick", "scheduler@p.iam.gserviceaccount.com"
	token := func(email string) string {
		seg := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
		unsigned := seg(map[string]string{"alg": "RS256", "kid": "k"}) + "." + seg(map[string]any{
			"iss": "https://accounts.google.com", "aud": aud, "email": email, "email_verified": true,
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		})
		d := sha256.Sum256([]byte(unsigned))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, d[:])
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
	}

	ticks := 0
	s := &Server{
		Log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tick:         func(context.Context) (int, error) { ticks++; return 3, nil },
		TickVerifier: &oidc.Verifier{Audience: aud, Email: caller, KeysURL: keys.URL},
	}
	call := func(auth string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/internal/tick", nil)
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		s.tick(w, r)
		return w
	}

	for name, auth := range map[string]string{
		"no token":                "",
		"not a bearer token":      "Basic dXNlcjpwYXNz",
		"another service account": "Bearer " + token("someone@p.iam.gserviceaccount.com"),
		"garbage":                 "Bearer abc",
	} {
		if w := call(auth); w.Code != http.StatusNotFound {
			t.Errorf("%s: got %d, want 404", name, w.Code)
		}
	}
	if ticks != 0 {
		t.Fatalf("an unauthorized call ran the checks %d times", ticks)
	}

	w := call("Bearer " + token(caller))
	if w.Code != http.StatusOK || ticks != 1 {
		t.Fatalf("the scheduler's call: %d %s (ticks %d)", w.Code, w.Body, ticks)
	}
	var body map[string]int
	if json.Unmarshal(w.Body.Bytes(), &body) != nil || body["checked"] != 3 {
		t.Fatalf("body: %s", w.Body)
	}

	// Not configured for an external schedule: the endpoint doesn't exist.
	s.Tick, s.TickVerifier = nil, nil
	if w := call("Bearer " + token(caller)); w.Code != http.StatusNotFound {
		t.Fatalf("loop mode: got %d, want 404", w.Code)
	}
}

func TestEdgeOnly(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	s := &Server{}
	s.Config.EdgeSecret = strings.Repeat("s", 40)
	h := s.edgeOnly(ok)
	call := func(path, secret string) int {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if secret != "" {
			r.Header.Set("X-Edge-Secret", secret)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if got := call("/", ""); got != http.StatusNotFound {
		t.Errorf("direct request: %d, want 404", got)
	}
	if got := call("/", "wrong"); got != http.StatusNotFound {
		t.Errorf("wrong secret: %d, want 404", got)
	}
	if got := call("/", strings.Repeat("s", 40)); got != http.StatusTeapot {
		t.Errorf("through the edge: %d", got)
	}
	for _, p := range []string{"/healthz", "/readyz", "/internal/tick"} {
		if got := call(p, ""); got != http.StatusTeapot {
			t.Errorf("%s should be exempt: %d", p, got)
		}
	}
	// No secret configured: everything passes, as before.
	if got := (&Server{}).edgeOnly(ok); got == nil {
		t.Fatal("nil handler")
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	(&Server{}).edgeOnly(ok).ServeHTTP(w, r)
	if w.Code != http.StatusTeapot {
		t.Errorf("no edge configured: %d", w.Code)
	}
}
