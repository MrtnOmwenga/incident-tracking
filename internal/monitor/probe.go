package monitor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"
)

// Failure is why a check failed, as a category. Raw errors are never stored or shown: they can
// name internal hosts and addresses.
type Failure string

const (
	FailTimeout    Failure = "timeout"
	FailConnection Failure = "connection"
	FailStatus     Failure = "status"
	FailContent    Failure = "content"
	FailTLS        Failure = "tls"
	FailBlocked    Failure = "blocked" // the target resolved to a private address the monitor may not reach
)

type Result struct {
	OK           bool
	StatusCode   int
	Latency      time.Duration
	Failure      Failure
	TLSExpiresAt *time.Time
}

// Target is one HTTP check.
type Target struct {
	URL          string
	Timeout      time.Duration
	StatusMin    int
	StatusMax    int
	ExpectedText string
	// AllowPrivate lets the check reach private, loopback and link-local addresses. Off by default:
	// without it, a monitor could be pointed at internal services or the cloud metadata endpoint.
	AllowPrivate bool
}

var (
	errBlocked          = errors.New("destination address is not allowed")
	errTooManyRedirects = errors.New("too many redirects")
)

// cgnat is 100.64.0.0/10, shared address space that netip doesn't classify as private.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// Blocked reports whether a check without AllowPrivate must not connect to ip.
func Blocked(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() || cgnat.Contains(ip)
}

// guard runs after DNS resolution, on the address actually being dialled, so a public hostname
// that resolves to a private address (DNS rebinding included) is caught too, as are redirects.
func guard(blocked func(netip.Addr) bool) func(_, address string, _ syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return errBlocked
		}
		ip, err := netip.ParseAddr(host)
		if err != nil || blocked(ip) {
			return errBlocked
		}
		return nil
	}
}

// Prober runs checks. Its two HTTP clients differ only in whether private addresses are allowed.
type Prober struct {
	public, private *http.Client
	rand            func() float64
}

const maxBody = 1 << 20 // content checks read at most 1 MiB

func NewProber() *Prober { return newProber(Blocked) }

func newProber(blocked func(netip.Addr) bool) *Prober {
	client := func(allowPrivate bool) *http.Client {
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		if !allowPrivate {
			dialer.Control = guard(blocked)
		}
		return &http.Client{
			Transport: &http.Transport{
				DialContext:           dialer.DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				DisableKeepAlives:     true, // every check measures a fresh connection
			},
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errTooManyRedirects
				}
				return nil
			},
		}
	}
	return &Prober{public: client(false), private: client(true), rand: rand.Float64}
}

// HTTP checks a target: reachable, a status in range, and the expected text if any.
func (p *Prober) HTTP(ctx context.Context, t Target) Result {
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()
	client := p.public
	if t.AllowPrivate {
		client = p.private
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		return Result{Failure: FailConnection}
	}
	req.Header.Set("User-Agent", "Lighthouse/1 (uptime monitor)")
	resp, err := client.Do(req)
	if err != nil {
		return Result{Latency: time.Since(start), Failure: classify(ctx, err)}
	}
	defer resp.Body.Close()
	res := Result{StatusCode: resp.StatusCode, Latency: time.Since(start)}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		expires := resp.TLS.PeerCertificates[0].NotAfter
		res.TLSExpiresAt = &expires
	}
	if resp.StatusCode < t.StatusMin || resp.StatusCode > t.StatusMax {
		res.Failure = FailStatus
		return res
	}
	if t.ExpectedText != "" {
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
		if err != nil {
			res.Failure = classify(ctx, err)
			return res
		}
		if !strings.Contains(string(body), t.ExpectedText) {
			res.Failure = FailContent
			return res
		}
	}
	res.OK = true
	return res
}

func classify(ctx context.Context, err error) Failure {
	var certInvalid x509.CertificateInvalidError
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var tlsRecord tls.RecordHeaderError
	var netErr net.Error
	switch {
	case errors.Is(err, errBlocked):
		return FailBlocked
	case errors.Is(err, errTooManyRedirects):
		return FailStatus
	case errors.As(err, &certInvalid), errors.As(err, &unknownAuthority), errors.As(err, &hostname), errors.As(err, &tlsRecord):
		return FailTLS
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return FailTimeout
	default:
		return FailConnection
	}
}

// Simulated produces a result for a sample target without any network traffic. Sandbox monitors
// use it, so visitors can break and fix a "site" without Lighthouse ever reaching a real address.
func (p *Prober) Simulated(mode string) Result {
	jitter := func(lo, hi float64) time.Duration {
		return time.Duration((lo + p.rand()*(hi-lo)) * float64(time.Millisecond))
	}
	switch mode {
	case "slow":
		return Result{OK: true, StatusCode: 200, Latency: jitter(1500, 2500)}
	case "flaky":
		if p.rand() < 0.5 {
			return Result{StatusCode: 503, Latency: jitter(40, 120), Failure: FailStatus}
		}
		return Result{OK: true, StatusCode: 200, Latency: jitter(40, 160)}
	case "down":
		return Result{StatusCode: 503, Latency: jitter(20, 60), Failure: FailStatus}
	default:
		return Result{OK: true, StatusCode: 200, Latency: jitter(40, 120)}
	}
}
