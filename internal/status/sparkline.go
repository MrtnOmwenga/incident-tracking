package status

import (
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

const (
	sparkWidth  = 300.0
	sparkHeight = 48.0
	sparkPad    = 3.0
)

// Sparkline draws a latency series as a small inline SVG: a line for latency (scaled to the
// series' own maximum) and a red tick for every bucket containing a failed check. It is rendered
// on the server, so the page needs no JavaScript. Only numbers are written into the markup.
func Sparkline(points []store.Point, window time.Duration, now time.Time) template.HTML {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="spark" viewBox="0 0 %g %g" preserveAspectRatio="none" role="img" aria-label="Response time, last 24 hours">`, sparkWidth, sparkHeight)
	valid := func(p store.Point) bool {
		return p.LatencyMS != nil && *p.LatencyMS >= 0 && !math.IsInf(*p.LatencyMS, 0) && !math.IsNaN(*p.LatencyMS)
	}
	peak := 0.0
	for _, p := range points {
		if valid(p) {
			peak = max(peak, *p.LatencyMS)
		}
	}
	start := now.Add(-window)
	x := func(t time.Time) float64 {
		f := float64(t.Sub(start)) / float64(window)
		return sparkPad + clamp(f, 0, 1)*(sparkWidth-2*sparkPad)
	}
	var line [][2]float64 // a run of consecutive buckets with passing checks
	flush := func() {
		switch len(line) {
		case 0:
		case 1:
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="1.5"/>`, line[0][0], line[0][1])
		default:
			b.WriteString(`<polyline points="`)
			for i, pt := range line {
				if i > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%.1f,%.1f", pt[0], pt[1])
			}
			b.WriteString(`"/>`)
		}
		line = line[:0]
	}
	for _, p := range points {
		if p.Failed > 0 {
			fmt.Fprintf(&b, `<rect class="fail" x="%.1f" y="0" width="2" height="%g"/>`, x(p.At)-1, sparkHeight)
		}
		if !valid(p) || peak == 0 {
			flush() // a gap: no passing checks in this bucket
			continue
		}
		y := sparkHeight - sparkPad - (*p.LatencyMS/peak)*(sparkHeight-2*sparkPad)
		line = append(line, [2]float64{x(p.At), y})
	}
	flush()
	b.WriteString(`</svg>`)
	return template.HTML(b.String()) //nolint:gosec // numbers only, formatted above
}

func clamp(v, lo, hi float64) float64 {
	return max(lo, min(hi, v))
}
