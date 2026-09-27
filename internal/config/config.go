// Package config reads Lighthouse's settings from the environment and refuses to start with an
// invalid or unsafe combination, listing every problem at once.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env         string // development, test or production
	Addr        string // listen address, e.g. ":8080"
	PublicURL   string // how browsers reach Lighthouse; used for OAuth redirects and origin checks
	DatabaseURL string // the API's least-privilege role (row-level security applies)

	GitHubClientID     string
	GitHubClientSecret string
	OwnerGitHubID      int64  // the only GitHub account that may sign in as the owner
	GitHubOAuthBase    string // overridable so tests can stand in for GitHub
	GitHubAPIBase      string

	// DevLogin enables POST /auth/dev, an owner login without GitHub, for local development only.
	DevLogin bool

	// ClientIPHeader names the header a trusted reverse proxy puts the visitor's IP in (e.g.
	// CF-Connecting-IP behind Cloudflare), for rate limiting. Empty: use the connection's address.
	// Only set it when every request passes through that proxy, or the header can be forged.
	ClientIPHeader string

	// SiteDir holds the portfolio's content: site.yaml, stories/ and media/. Empty: no content.
	SiteDir string

	OwnerName     string        // shown on the public status page
	CheckWorkers  int           // concurrent checks
	SandboxTTL    time.Duration // how long a visitor's sandbox lives
	RetentionDays int           // how long check results are kept
}

func (c Config) Production() bool { return c.Env == "production" }

// Load reads the configuration from getenv (os.Getenv in production, a map in tests).
func Load(getenv func(string) string) (Config, error) {
	var problems []string
	get := func(key, fallback string) string {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v
		}
		return fallback
	}
	integer := func(key string, fallback, lo, hi int) int {
		raw := get(key, strconv.Itoa(fallback))
		n, err := strconv.Atoi(raw)
		if err != nil || n < lo || n > hi {
			problems = append(problems, fmt.Sprintf("%s must be a whole number between %d and %d", key, lo, hi))
		}
		return n
	}

	c := Config{
		Env:                get("LIGHTHOUSE_ENV", "development"),
		Addr:               get("ADDR", ":8080"),
		PublicURL:          strings.TrimRight(get("PUBLIC_URL", "http://localhost:8080"), "/"),
		DatabaseURL:        get("DATABASE_URL", ""),
		GitHubClientID:     get("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: get("GITHUB_CLIENT_SECRET", ""),
		GitHubOAuthBase:    strings.TrimRight(get("GITHUB_OAUTH_BASE", "https://github.com"), "/"),
		GitHubAPIBase:      strings.TrimRight(get("GITHUB_API_BASE", "https://api.github.com"), "/"),
		DevLogin:           get("DEV_LOGIN", "false") == "true",
		ClientIPHeader:     get("CLIENT_IP_HEADER", ""),
		SiteDir:            get("SITE_DIR", ""),
		OwnerName:          get("OWNER_NAME", "Lighthouse"),
		CheckWorkers:       integer("CHECK_WORKERS", 8, 1, 256),
		SandboxTTL:         time.Duration(integer("SANDBOX_TTL_MINUTES", 120, 5, 24*60)) * time.Minute,
		RetentionDays:      integer("RETENTION_DAYS", 90, 1, 3650),
	}
	if owner := get("OWNER_GITHUB_ID", "0"); owner != "0" {
		id, err := strconv.ParseInt(owner, 10, 64)
		if err != nil || id <= 0 {
			problems = append(problems, "OWNER_GITHUB_ID must be your numeric GitHub user ID")
		}
		c.OwnerGitHubID = id
	}

	switch c.Env {
	case "development", "test", "production":
	default:
		problems = append(problems, "LIGHTHOUSE_ENV must be development, test or production")
	}
	if c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if u, err := url.Parse(c.PublicURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		problems = append(problems, "PUBLIC_URL must be an absolute http(s) URL")
	}
	if c.DevLogin && c.Production() {
		problems = append(problems, "DEV_LOGIN must not be enabled in production")
	}
	if c.Production() {
		if !strings.HasPrefix(c.PublicURL, "https://") {
			problems = append(problems, "PUBLIC_URL must use https in production")
		}
		if c.GitHubClientID == "" || c.GitHubClientSecret == "" || c.OwnerGitHubID == 0 {
			problems = append(problems, "production needs GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET and OWNER_GITHUB_ID for the owner login")
		}
	}
	if len(problems) > 0 {
		return Config{}, errors.New("invalid configuration:\n  " + strings.Join(problems, "\n  "))
	}
	return c, nil
}

// SecureCookies reports whether cookies should be marked Secure (served over HTTPS).
func (c Config) SecureCookies() bool { return strings.HasPrefix(c.PublicURL, "https://") }
