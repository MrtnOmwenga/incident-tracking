package status

import (
	"encoding/xml"
	"strconv"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

func TestPercentNeverRoundsUpToPerfect(t *testing.T) {
	if Percent(0, 0) != nil {
		t.Fatal("no checks: no number")
	}
	if p := *Percent(99999, 100000); p != 99.99 {
		t.Fatalf("99.999%% shows as %v", p)
	}
	rapid.Check(t, func(t *rapid.T) {
		total := rapid.IntRange(1, 1_000_000).Draw(t, "total")
		ok := rapid.IntRange(0, total).Draw(t, "ok")
		p := *Percent(ok, total)
		if p < 0 || p > 100 || (p == 100) != (ok == total) {
			t.Fatalf("Percent(%d, %d) = %v", ok, total, p)
		}
	})
}

func TestOverall(t *testing.T) {
	up, down := Monitor{Health: "up"}, Monitor{Health: "down"}
	for _, c := range []struct {
		monitors []Monitor
		active   int
		want     Overall
	}{
		{nil, 0, NoMonitorsYet},
		{[]Monitor{up, up}, 0, Operational},
		{[]Monitor{up, {Health: "unknown"}}, 0, Operational},
		{[]Monitor{up, down}, 0, Degraded},
		{[]Monitor{up}, 1, Degraded},
		{nil, 1, Degraded},
		{[]Monitor{down, down}, 2, MajorOutage},
	} {
		if got := overall(c.monitors, c.active); got != c.want {
			t.Errorf("overall(%v, %d) = %s, want %s", c.monitors, c.active, got, c.want)
		}
	}
}

func TestFillDays(t *testing.T) {
	now := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	days := fillDays([]store.Day{
		{Date: time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC), Checks: 10, OK: 9},
		{Date: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), Checks: 4, OK: 4},
	}, now)
	if len(days) != Days || days[Days-1].Date != "2026-09-27" || days[0].Date != "2026-06-30" {
		t.Fatalf("range %s..%s (%d)", days[0].Date, days[len(days)-1].Date, len(days))
	}
	if *days[Days-1].Uptime != 90 || days[Days-1].Bar() != "bad" || (status99()).Bar() != "fair" || *days[0].Uptime != 100 || days[0].Bar() != "good" {
		t.Fatalf("values: %+v %+v", days[0], days[Days-1])
	}
	if days[1].Uptime != nil || days[1].Bar() != "none" {
		t.Fatalf("a day without checks has no uptime: %+v", days[1])
	}
}

// Whatever the series, the sparkline is well-formed SVG with coordinates inside the viewBox.
func TestSparklineIsAlwaysValidSVG(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 60).Draw(t, "n")
		points := make([]store.Point, n)
		for i := range points {
			points[i].At = now.Add(-time.Duration(rapid.Int64Range(-int64(time.Hour), int64(26*time.Hour)).Draw(t, "age")))
			if rapid.Bool().Draw(t, "passed") {
				l := rapid.Float64Range(0, 1e6).Draw(t, "latency")
				points[i].LatencyMS = &l
			}
			points[i].Failed = rapid.IntRange(0, 3).Draw(t, "failed")
		}
		svg := string(Sparkline(points, 24*time.Hour, now))
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
			t.Fatalf("non-finite number: %s", svg)
		}
		dec := xml.NewDecoder(strings.NewReader(svg))
		for {
			tok, err := dec.Token()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				t.Fatalf("invalid SVG: %v\n%s", err, svg)
			}
			if el, ok := tok.(xml.StartElement); ok {
				for _, a := range el.Attr {
					if a.Name.Local == "cx" || a.Name.Local == "cy" || a.Name.Local == "x" {
						v, err := strconv.ParseFloat(a.Value, 64)
						if err != nil || v < -1 || v > sparkWidth {
							t.Fatalf("%s=%s out of bounds", a.Name.Local, a.Value)
						}
					}
				}
			}
		}
	})
}

func FuzzSparkline(f *testing.F) {
	f.Add(int64(0), 12.5, 0, int64(3600))
	f.Fuzz(func(t *testing.T, ageNanos int64, latency float64, failed int, age2 int64) {
		now := time.Now()
		l := latency
		points := []store.Point{{At: now.Add(-time.Duration(ageNanos)), LatencyMS: &l, Failed: failed}, {At: now.Add(-time.Duration(age2))}}
		svg := string(Sparkline(points, 24*time.Hour, now))
		if err := xml.Unmarshal([]byte(svg), new(any)); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
			t.Fatalf("non-finite number: %s", svg)
		}
	})
}

func status99() Day { p := 99.0; return Day{Uptime: &p} }
