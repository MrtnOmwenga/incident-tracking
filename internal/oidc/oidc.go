// Package oidc verifies the identity tokens Google attaches to requests it makes on a service
// account's behalf (Cloud Scheduler calling Lighthouse, for example): an RS256-signed JWT whose
// keys Google publishes. It checks the signature, issuer, audience, expiry and the service
// account's email, and nothing else, so it needs no client library.
package oidc

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GoogleKeysURL is where Google publishes the keys that sign its identity tokens.
const GoogleKeysURL = "https://www.googleapis.com/oauth2/v3/certs"

var googleIssuers = map[string]bool{"accounts.google.com": true, "https://accounts.google.com": true}

// Verifier accepts tokens for one audience, from one service account.
type Verifier struct {
	Audience string // the token's aud: what the caller was told to ask a token for
	Email    string // the service account allowed to call
	KeysURL  string // defaults to GoogleKeysURL; tests point it at a fake
	Client   *http.Client
	Now      func() time.Time

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
}

var ErrInvalid = errors.New("invalid identity token")

// leeway allows for clocks that disagree by a little.
const leeway = 60 * time.Second

type claims struct {
	Iss           string          `json:"iss"`
	Aud           json.RawMessage `json:"aud"` // a string, or (per the JWT spec) a list
	Exp           int64           `json:"exp"`
	Iat           int64           `json:"iat"`
	Email         string          `json:"email"`
	EmailVerified bool            `json:"email_verified"`
}

// Verify checks a bearer token ("Authorization: Bearer <token>" minus the prefix).
func (v *Verifier) Verify(ctx context.Context, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("%w: not a JWT", ErrInvalid)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeSegment(parts[0], &header); err != nil {
		return fmt.Errorf("%w: header: %v", ErrInvalid, err)
	}
	// Only RS256: never let the token choose a weaker algorithm (or "none").
	if header.Alg != "RS256" {
		return fmt.Errorf("%w: algorithm %q", ErrInvalid, header.Alg)
	}
	key, err := v.key(ctx, header.Kid)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("%w: signature encoding", ErrInvalid)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return fmt.Errorf("%w: bad signature", ErrInvalid)
	}

	var c claims
	if err := decodeSegment(parts[1], &c); err != nil {
		return fmt.Errorf("%w: claims: %v", ErrInvalid, err)
	}
	now := v.now()
	switch {
	case !googleIssuers[c.Iss]:
		return fmt.Errorf("%w: issuer %q", ErrInvalid, c.Iss)
	case !audienceMatches(c.Aud, v.Audience):
		return fmt.Errorf("%w: audience", ErrInvalid)
	case now.After(time.Unix(c.Exp, 0).Add(leeway)):
		return fmt.Errorf("%w: expired", ErrInvalid)
	case time.Unix(c.Iat, 0).After(now.Add(leeway)):
		return fmt.Errorf("%w: issued in the future", ErrInvalid)
	case v.Email == "" || !strings.EqualFold(c.Email, v.Email) || !c.EmailVerified:
		return fmt.Errorf("%w: caller %q", ErrInvalid, c.Email)
	}
	return nil
}

func audienceMatches(raw json.RawMessage, want string) bool {
	if want == "" {
		return false
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one == want
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		for _, a := range many {
			if a == want {
				return true
			}
		}
	}
	return false
}

func decodeSegment(seg string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (v *Verifier) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

// key returns the public key for kid, fetching Google's key set when it's missing or stale.
// Google rotates keys roughly daily and says how long a set may be cached (Cache-Control max-age);
// an unknown kid triggers one refetch, since a new key may have been published since.
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok && v.now().Before(v.expires) {
		return k, nil
	}
	if err := v.fetch(ctx); err != nil {
		return nil, err
	}
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("%w: unknown key %q", ErrInvalid, kid)
}

func (v *Verifier) fetch(ctx context.Context) error {
	url := v.KeysURL
	if url == "" {
		url = GoogleKeysURL
	}
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching signing keys: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching signing keys: %s", res.Status)
	}
	var set struct {
		Keys []struct {
			Kid, Kty, Alg, N, E string
		} `json:"keys"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, res.Body, 1<<20)).Decode(&set); err != nil {
		return fmt.Errorf("decoding signing keys: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, errN := base64.RawURLEncoding.DecodeString(k.N)
		e, errE := base64.RawURLEncoding.DecodeString(k.E)
		if errN != nil || errE != nil || len(e) == 0 || len(e) > 4 {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if len(keys) == 0 {
		return errors.New("signing keys: none usable")
	}
	v.keys = keys
	v.expires = v.now().Add(maxAge(res.Header.Get("Cache-Control")))
	return nil
}

// maxAge reads max-age from Cache-Control, between 5 minutes and a day (default an hour).
func maxAge(cc string) time.Duration {
	for _, part := range strings.Split(cc, ",") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(part), "max-age="); ok {
			if s, err := strconv.Atoi(v); err == nil {
				return min(max(time.Duration(s)*time.Second, 5*time.Minute), 24*time.Hour)
			}
		}
	}
	return time.Hour
}
