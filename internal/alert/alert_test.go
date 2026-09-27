package alert

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MrtnOmwenga/lighthouse/internal/monitor"
)

// fakeSMTP is just enough of an SMTP server to receive messages. It never offers STARTTLS.
type fakeSMTP struct {
	addr     string
	mu       sync.Mutex
	messages []string
	auth     []string
}

func startSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	f := &fakeSMTP{addr: ln.Addr().String()}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(conn)
		}
	}()
	return f
}

func (f *fakeSMTP) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	say := func(s string) { _, _ = io.WriteString(conn, s+"\r\n") }
	say("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"):
			say("250-fake")
			say("250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH"):
			f.mu.Lock()
			f.auth = append(f.auth, strings.TrimSpace(line))
			f.mu.Unlock()
			say("235 ok")
		case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
			say("250 ok")
		case cmd == "DATA":
			say("354 go on")
			var msg strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				msg.WriteString(l)
			}
			f.mu.Lock()
			f.messages = append(f.messages, msg.String())
			f.mu.Unlock()
			say("250 queued")
		case cmd == "QUIT":
			say("221 bye")
			return
		default:
			say("250 ok")
		}
	}
}

func (f *fakeSMTP) mailer(allowPlain bool) *Mailer {
	host, port, _ := net.SplitHostPort(f.addr)
	p := 0
	for _, ch := range port {
		p = p*10 + int(ch-'0')
	}
	return &Mailer{Host: host, Port: p, Username: "u", Password: "p", From: "lighthouse@example.com",
		To: []string{"me@example.com"}, Timeout: 3 * time.Second, allowPlain: allowPlain,
		now: func() time.Time { return time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC) }}
}

func TestSendsAPlainTextMessage(t *testing.T) {
	f := startSMTP(t)
	if err := f.mailer(true).Send(context.Background(), "Café is down\r\nBcc: attacker@evil.example", "Line one\nLine two"); err != nil {
		t.Fatal(err)
	}
	if len(f.messages) != 1 {
		t.Fatalf("%d messages", len(f.messages))
	}
	msg := f.messages[0]
	for _, want := range []string{"From: lighthouse@example.com\r\n", "To: me@example.com\r\n", "Content-Type: text/plain; charset=utf-8", "Line one\r\nLine two"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	// A title can't smuggle in headers: line breaks are removed, non-ASCII is encoded.
	if strings.Contains(msg, "\r\nBcc:") || !strings.Contains(msg, "Subject: =?utf-8?q?") {
		t.Errorf("header injection or unencoded subject:\n%s", msg)
	}
}

// Credentials never go over an unencrypted connection: a server without STARTTLS is refused
// before any AUTH is sent.
func TestRefusesToSendWithoutTLS(t *testing.T) {
	f := startSMTP(t)
	err := f.mailer(false).Send(context.Background(), "s", "b")
	if !errors.Is(err, ErrNoTLS) {
		t.Fatalf("want ErrNoTLS, got %v", err)
	}
	if len(f.auth) != 0 || len(f.messages) != 0 {
		t.Fatalf("credentials or mail sent in the clear: %v %d", f.auth, len(f.messages))
	}
}

func TestNotifierEmailsTheOwnerOnly(t *testing.T) {
	f := startSMTP(t)
	n := &Notifier{Mailer: f.mailer(true), Tenant: "owner", PublicURL: "https://status.example", Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	at := time.Date(2026, 9, 28, 9, 30, 0, 0, time.UTC)
	n.Notify(context.Background(), monitor.Change{TenantID: "sandbox-123", Opened: true, Title: "Storefront is down"})
	if len(f.messages) != 0 {
		t.Fatal("a sandbox's incident must never send email")
	}
	n.Notify(context.Background(), monitor.Change{TenantID: "owner", Opened: true, IncidentID: "i-1", Title: "GhostChat is down",
		Monitor: "GhostChat", Public: true, StartedAt: at, At: at, Failure: monitor.FailTimeout})
	n.Notify(context.Background(), monitor.Change{TenantID: "owner", IncidentID: "i-1", Title: "GhostChat is down",
		Monitor: "GhostChat", Public: true, StartedAt: at, At: at.Add(7 * time.Minute)})
	if len(f.messages) != 2 {
		t.Fatalf("%d messages, want 2", len(f.messages))
	}
	opened, resolved := f.messages[0], f.messages[1]
	for _, want := range []string{"Subject: [Lighthouse] GhostChat is down", "(timeout)", "https://status.example/console/incidents/i-1", "https://status.example/status/incidents/i-1"} {
		if !strings.Contains(opened, want) {
			t.Errorf("opened email lacks %q:\n%s", want, opened)
		}
	}
	if !strings.Contains(resolved, "Subject: [Lighthouse] Resolved: GhostChat is down") || !strings.Contains(resolved, "lasted 7m0s") {
		t.Errorf("resolved email:\n%s", resolved)
	}
}

func TestPrivateIncidentsDontLinkToThePublicPage(t *testing.T) {
	_, body := Compose(monitor.Change{Opened: true, IncidentID: "i-2", Title: "Internal API is down", Monitor: "Internal API", At: time.Now()}, "https://s.example")
	if strings.Contains(body, "/status/incidents/") {
		t.Fatalf("a private incident has no public page:\n%s", body)
	}
}
