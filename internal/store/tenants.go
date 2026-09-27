package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OwnerTenant returns the owner tenant's ID, creating it on first use.
func OwnerTenant(ctx context.Context, pool *pgxpool.Pool, name string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `SELECT lighthouse_owner_tenant($1)`, name).Scan(&id)
	return id, err
}

// CreateSandbox creates a sandbox tenant, then runs seed inside it.
func CreateSandbox(ctx context.Context, pool *pgxpool.Pool, id string, seed func(pgx.Tx) error) error {
	return WithTenant(ctx, pool, id, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO tenants (id, kind, name) VALUES ($1, 'sandbox', 'Sandbox')`, id); err != nil {
			return err
		}
		return seed(tx)
	})
}

type Session struct {
	TenantID    string
	Role        string // owner or sandbox
	GitHubLogin *string
	ExpiresAt   time.Time
}

// LookupSession finds the session for a cookie's token hash, in any tenant.
func LookupSession(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (Session, error) {
	var s Session
	err := pool.QueryRow(ctx, `SELECT tenant_id, role, github_login, expires_at FROM lighthouse_session($1)`, tokenHash).
		Scan(&s.TenantID, &s.Role, &s.GitHubLogin, &s.ExpiresAt)
	return s, notFound(err)
}

func CreateSession(ctx context.Context, tx pgx.Tx, tokenHash string, s Session) error {
	_, err := tx.Exec(ctx, `INSERT INTO sessions (token_hash, tenant_id, role, github_login, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		tokenHash, s.TenantID, s.Role, s.GitHubLogin, s.ExpiresAt)
	return err
}

func DeleteSession(ctx context.Context, tx pgx.Tx, tokenHash string) error {
	_, err := tx.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

type Pruned struct{ Checks, Sandboxes, Sessions int64 }

// Prune deletes old checks, expired sessions and sandboxes older than keepSandboxes.
func Prune(ctx context.Context, pool *pgxpool.Pool, keepChecks, keepSandboxes time.Duration) (Pruned, error) {
	var p Pruned
	err := pool.QueryRow(ctx, `SELECT checks, sandboxes, sessions FROM lighthouse_prune($1, $2)`, keepChecks, keepSandboxes).
		Scan(&p.Checks, &p.Sandboxes, &p.Sessions)
	return p, err
}
