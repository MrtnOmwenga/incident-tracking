package web

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/MrtnOmwenga/lighthouse/internal/auth"
	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`)

const maxSandboxMonitors = 10

// normalize fills defaults into a monitor and checks it. The database checks the same ranges;
// checking here too gives a useful message instead of a constraint name.
func normalize(in store.MonitorInput, id auth.Identity) (store.MonitorInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Slug == "" {
		in.Slug = slugify(in.Name)
	}
	defaults := map[*int]int{
		&in.IntervalSeconds: 60, &in.TimeoutMS: 10000, &in.ExpectedStatusMin: 200, &in.ExpectedStatusMax: 399,
		&in.FailureThreshold: 3, &in.RecoveryThreshold: 2,
	}
	for field, value := range defaults {
		if *field == 0 {
			*field = value
		}
	}
	if in.SimulatedMode == "" {
		in.SimulatedMode = "up"
	}
	if in.ExpectedText != nil && *in.ExpectedText == "" {
		in.ExpectedText = nil
	}

	switch {
	case in.Name == "" || utf8.RuneCountInString(in.Name) > 100:
		return in, invalid("name: 1 to 100 characters.")
	case !slugPattern.MatchString(in.Slug):
		return in, invalid("slug: lowercase letters, digits and dashes, up to 50.")
	case in.Kind != "http" && in.Kind != "simulated":
		return in, invalid("kind: http or simulated.")
	case !oneOf(in.SimulatedMode, "up", "slow", "flaky", "down"):
		return in, invalid("simulatedMode: up, slow, flaky or down.")
	case in.IntervalSeconds < 5 || in.IntervalSeconds > 3600:
		return in, invalid("intervalSeconds: 5 to 3600.")
	case in.TimeoutMS < 100 || in.TimeoutMS > 30000:
		return in, invalid("timeoutMs: 100 to 30000.")
	case in.ExpectedStatusMin < 100 || in.ExpectedStatusMax > 599 || in.ExpectedStatusMin > in.ExpectedStatusMax:
		return in, invalid("expected status: a range within 100 to 599.")
	case in.FailureThreshold < 1 || in.FailureThreshold > 20 || in.RecoveryThreshold < 1 || in.RecoveryThreshold > 20:
		return in, invalid("thresholds: 1 to 20.")
	case in.ExpectedText != nil && utf8.RuneCountInString(*in.ExpectedText) > 200:
		return in, invalid("expectedText: up to 200 characters.")
	}

	if in.Kind == "http" {
		if in.URL == nil {
			return in, invalid("url: required for http monitors.")
		}
		u, err := url.Parse(strings.TrimSpace(*in.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || len(*in.URL) > 2000 {
			return in, invalid("url: an absolute http(s) URL without credentials.")
		}
		clean := u.String()
		in.URL = &clean
	} else {
		in.URL = nil
	}

	// Sandboxes are for anyone on the internet: no real network traffic, no fast intervals.
	if !id.Owner() {
		if in.Kind != "simulated" {
			return in, invalid("The sandbox only runs simulated monitors: it doesn't send traffic to real sites.")
		}
		if in.AllowPrivateNetwork {
			return in, errForbidden
		}
		if in.IntervalSeconds < 10 {
			return in, invalid("intervalSeconds: at least 10 in the sandbox.")
		}
	}
	return in, nil
}

func oneOf(v string, options ...string) bool {
	for _, o := range options {
		if v == o {
			return true
		}
	}
	return false
}

func slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= 50 {
			break
		}
	}
	return strings.TrimRight(b.String(), "-")
}
