// Package testdb gives each test its own migrated Postgres database, cloned from a template in one
// container per test binary. Cloning takes milliseconds, so every test starts clean and tests can
// run in parallel.
//
// Set LIGHTHOUSE_TEST_DATABASE_URL (a superuser URL) to use an existing server instead of a
// container.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/MrtnOmwenga/lighthouse/internal/store"
)

const (
	template    = "lighthouse_template"
	appUser     = "lighthouse_test"
	appPassword = "test-only-password"
)

var (
	once      sync.Once
	serverURL *url.URL // superuser, database left blank
	setupErr  error
	container testcontainers.Container
	counter   atomic.Int64
)

// Main runs a package's tests and then stops the container. Call it from TestMain.
func Main(m *testing.M) {
	code := m.Run()
	if container != nil {
		_ = container.Terminate(context.Background())
	}
	os.Exit(code)
}

// DB is one test's database: App connects as the API's least-privileged role, so row-level
// security applies; Owner is a superuser, for setting up and inspecting state behind RLS's back.
type DB struct {
	App, Owner *pgxpool.Pool
	AppURL     string
}

func New(t testing.TB) DB {
	t.Helper()
	once.Do(setup)
	if setupErr != nil {
		t.Fatalf("test database: %v", setupErr)
	}
	ctx := context.Background()
	name := fmt.Sprintf("lighthouse_test_%d_%d", os.Getpid(), counter.Add(1))
	admin, err := pgxpool.New(ctx, dbURL(serverURL, "postgres", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, template)); err != nil {
		t.Fatalf("clone template: %v", err)
	}
	db := DB{AppURL: dbURL(serverURL, name, url.UserPassword(appUser, appPassword))}
	if db.Owner, err = pgxpool.New(ctx, dbURL(serverURL, name, nil)); err != nil {
		t.Fatal(err)
	}
	if db.App, err = pgxpool.New(ctx, db.AppURL); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.App.Close()
		db.Owner.Close()
		if admin, err := pgxpool.New(context.Background(), dbURL(serverURL, "postgres", nil)); err == nil {
			_, _ = admin.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
			admin.Close()
		}
	})
	return db
}

func setup() {
	ctx := context.Background()
	raw := os.Getenv("LIGHTHOUSE_TEST_DATABASE_URL")
	if raw == "" {
		if strings.Contains(os.Getenv("DOCKER_HOST"), "podman") {
			// Ryuk (the reaper container) needs a privileged Docker socket that rootless Podman
			// doesn't provide; Main terminates the container instead.
			os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		}
		pg, err := postgres.Run(ctx, "docker.io/library/postgres:17-alpine",
			postgres.WithUsername("postgres"), postgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
			// A throwaway database: durability only slows the tests down.
			testcontainers.WithCmd("postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"),
		)
		if err != nil {
			setupErr = fmt.Errorf("start postgres: %w", err)
			return
		}
		container = pg
		if raw, err = pg.ConnectionString(ctx, "sslmode=disable"); err != nil {
			setupErr = err
			return
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		setupErr = err
		return
	}
	serverURL = u

	admin, err := pgxpool.New(ctx, dbURL(u, "postgres", nil))
	if err != nil {
		setupErr = err
		return
	}
	defer admin.Close()
	for _, stmt := range []string{
		"DROP DATABASE IF EXISTS " + template + " WITH (FORCE)",
		"CREATE DATABASE " + template,
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			setupErr = err
			return
		}
	}
	setupErr = store.Migrate(ctx, dbURL(u, template, nil), dbURL(u, template, url.UserPassword(appUser, appPassword)), "")
}

// dbURL points the server URL at another database, and optionally another user.
func dbURL(server *url.URL, database string, user *url.Userinfo) string {
	u := *server
	u.Path = "/" + database
	if user != nil {
		u.User = user
	}
	return u.String()
}
