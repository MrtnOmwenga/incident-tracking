package web

import (
	"testing"

	"github.com/MrtnOmwenga/lighthouse/internal/auth"
	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"GhostChat": "ghostchat", "Checkout API": "checkout-api", "  --Ünïcode!! site--  ": "n-code-site", "!!!": "",
	} {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// Whatever comes in, normalize either rejects it or returns something the database accepts, and
// never lets a sandbox create a monitor that sends real traffic.
func FuzzNormalize(f *testing.F) {
	f.Add("GhostChat", "", "http", "https://ghostchat.example/health", 60, 0, 0, false, true)
	f.Add("x", "x", "simulated", "", 5, 200, 399, true, false)
	f.Add("A", "a-", "http", "http://user:pw@host/", 3601, 600, 100, false, true)
	f.Fuzz(func(t *testing.T, name, slug, kind, url string, interval, min, max int, private, owner bool) {
		id := auth.Identity{Role: "sandbox"}
		if owner {
			id.Role = "owner"
		}
		in := store.MonitorInput{Name: name, Slug: slug, Kind: kind, IntervalSeconds: interval,
			ExpectedStatusMin: min, ExpectedStatusMax: max, AllowPrivateNetwork: private}
		if url != "" {
			in.URL = &url
		}
		out, err := normalize(in, id)
		if err != nil {
			return
		}
		if !slugPattern.MatchString(out.Slug) || out.Name == "" || out.ExpectedStatusMin > out.ExpectedStatusMax ||
			out.IntervalSeconds < 5 || out.IntervalSeconds > 3600 || (out.Kind == "http") != (out.URL != nil) {
			t.Fatalf("accepted an invalid monitor: %+v", out)
		}
		if !owner && (out.Kind != "simulated" || out.AllowPrivateNetwork) {
			t.Fatalf("a sandbox got a real monitor: %+v", out)
		}
	})
}
