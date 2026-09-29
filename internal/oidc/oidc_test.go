package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	aud   = "https://lighthouse.example/internal/tick"
	email = "scheduler@proj.iam.gserviceaccount.com"
)

var now = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

type fixture struct {
	key     *rsa.PrivateKey
	server  *httptest.Server
	fetches atomic.Int32
	kid     string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{key: key, kid: "k1"}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f.fetches.Add(1)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		pub := f.key.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": f.kid, "kty": "RSA", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fixture) verifier() *Verifier {
	return &Verifier{Audience: aud, Email: email, KeysURL: f.server.URL, Now: func() time.Time { return now }}
}

func (f *fixture) token(t *testing.T, header map[string]any, claims map[string]any, signWith *rsa.PrivateKey) string {
	t.Helper()
	seg := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	unsigned := seg(header) + "." + seg(claims)
	digest := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, signWith, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func goodClaims() map[string]any {
	return map[string]any{
		"iss": "https://accounts.google.com", "aud": aud, "email": email, "email_verified": true,
		"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(59 * time.Minute).Unix(),
	}
}

func TestVerifyAcceptsAGoogleToken(t *testing.T) {
	f := newFixture(t)
	v := f.verifier()
	tok := f.token(t, map[string]any{"alg": "RS256", "kid": "k1"}, goodClaims(), f.key)
	if err := v.Verify(context.Background(), tok); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	// The key set is cached: a second token doesn't refetch it.
	if err := v.Verify(context.Background(), tok); err != nil {
		t.Fatal(err)
	}
	if n := f.fetches.Load(); n != 1 {
		t.Fatalf("fetched the keys %d times, want 1", n)
	}
}

func TestVerifyRejects(t *testing.T) {
	f := newFixture(t)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	with := func(k string, v any) map[string]any { c := goodClaims(); c[k] = v; return c }
	rs256 := map[string]any{"alg": "RS256", "kid": "k1"}
	cases := map[string]string{
		"signed by another key":         f.token(t, rs256, goodClaims(), other),
		"wrong audience":                f.token(t, rs256, with("aud", "https://elsewhere"), f.key),
		"another service account":       f.token(t, rs256, with("email", "attacker@proj.iam.gserviceaccount.com"), f.key),
		"unverified email":              f.token(t, rs256, with("email_verified", false), f.key),
		"not Google":                    f.token(t, rs256, with("iss", "https://evil.example"), f.key),
		"expired":                       f.token(t, rs256, with("exp", now.Add(-2*time.Minute).Unix()), f.key),
		"issued in the future":          f.token(t, rs256, with("iat", now.Add(5*time.Minute).Unix()), f.key),
		"unknown key":                   f.token(t, map[string]any{"alg": "RS256", "kid": "nope"}, goodClaims(), f.key),
		"algorithm none":                strings.Join(strings.Split(f.token(t, map[string]any{"alg": "none", "kid": "k1"}, goodClaims(), f.key), ".")[:2], ".") + ".",
		"HS256 (key confusion attempt)": f.token(t, map[string]any{"alg": "HS256", "kid": "k1"}, goodClaims(), f.key),
		"garbage":                       "not.a.jwt.at.all",
		"tampered claims":               tamper(f.token(t, rs256, goodClaims(), f.key)),
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if err := f.verifier().Verify(context.Background(), tok); !errors.Is(err, ErrInvalid) {
				t.Fatalf("got %v, want ErrInvalid", err)
			}
		})
	}
}

// tamper swaps the claims for different ones, keeping the original signature.
func tamper(tok string) string {
	parts := strings.Split(tok, ".")
	b, _ := json.Marshal(map[string]any{"iss": "https://accounts.google.com", "aud": aud, "email": email, "email_verified": true, "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "extra": 1})
	parts[1] = base64.RawURLEncoding.EncodeToString(b)
	return strings.Join(parts, ".")
}

func TestAudienceMayBeAList(t *testing.T) {
	f := newFixture(t)
	c := goodClaims()
	c["aud"] = []string{"https://other", aud}
	if err := f.verifier().Verify(context.Background(), f.token(t, map[string]any{"alg": "RS256", "kid": "k1"}, c, f.key)); err != nil {
		t.Fatal(err)
	}
}

func TestRotatedKeyIsFetched(t *testing.T) {
	f := newFixture(t)
	v := f.verifier()
	rs := func(kid string) map[string]any { return map[string]any{"alg": "RS256", "kid": kid} }
	if err := v.Verify(context.Background(), f.token(t, rs("k1"), goodClaims(), f.key)); err != nil {
		t.Fatal(err)
	}
	// Google publishes a new key; a token signed with it must be accepted without waiting for the
	// cached set to expire.
	f.key, _ = rsa.GenerateKey(rand.Reader, 2048)
	f.kid = "k2"
	if err := v.Verify(context.Background(), f.token(t, rs("k2"), goodClaims(), f.key)); err != nil {
		t.Fatalf("rotated key rejected: %v", err)
	}
}

func TestMisconfiguredVerifierRejectsEverything(t *testing.T) {
	f := newFixture(t)
	v := f.verifier()
	v.Email = ""
	if err := v.Verify(context.Background(), f.token(t, map[string]any{"alg": "RS256", "kid": "k1"}, goodClaims(), f.key)); !errors.Is(err, ErrInvalid) {
		t.Fatal("a verifier with no allowed caller accepted a token")
	}
}
