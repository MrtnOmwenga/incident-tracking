-- Lighthouse's schema. Every tenant table has forced row-level security keyed on the
-- transaction-local setting app.tenant_id, so a query that forgets its tenant filter still can't
-- see or change another tenant's rows. The API connects as a member of lighthouse_app, which owns
-- nothing; migrations run as the owner (which needs BYPASSRLS, e.g. the Postgres superuser).
--
-- Tenants: one "owner" tenant (the real monitors and incidents) and throwaway "sandbox" tenants
-- that let visitors use the full console on sample data without touching the real thing.

-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'lighthouse_app') THEN CREATE ROLE lighthouse_app NOLOGIN; END IF;
END $$;
-- +goose StatementEnd

CREATE TABLE tenants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  kind text NOT NULL CHECK (kind IN ('owner', 'sandbox')),
  name text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX tenants_one_owner ON tenants (kind) WHERE kind = 'owner';

CREATE TABLE monitors (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  slug text NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,49}$'),
  kind text NOT NULL CHECK (kind IN ('http', 'simulated')),
  url text,
  simulated_mode text NOT NULL DEFAULT 'up' CHECK (simulated_mode IN ('up', 'slow', 'flaky', 'down')),
  interval_seconds integer NOT NULL CHECK (interval_seconds BETWEEN 5 AND 3600),
  timeout_ms integer NOT NULL DEFAULT 10000 CHECK (timeout_ms BETWEEN 100 AND 30000),
  expected_status_min integer NOT NULL DEFAULT 200 CHECK (expected_status_min BETWEEN 100 AND 599),
  expected_status_max integer NOT NULL DEFAULT 399 CHECK (expected_status_max BETWEEN 100 AND 599),
  expected_text text CHECK (length(expected_text) <= 200),
  failure_threshold integer NOT NULL DEFAULT 3 CHECK (failure_threshold BETWEEN 1 AND 20),
  recovery_threshold integer NOT NULL DEFAULT 2 CHECK (recovery_threshold BETWEEN 1 AND 20),
  is_public boolean NOT NULL DEFAULT false,
  paused boolean NOT NULL DEFAULT false,
  -- Only the owner may point a monitor at private addresses (e.g. services inside the cluster).
  allow_private_network boolean NOT NULL DEFAULT false,
  -- State kept by the scheduler.
  health text NOT NULL DEFAULT 'unknown' CHECK (health IN ('unknown', 'up', 'down')),
  consecutive_failures integer NOT NULL DEFAULT 0,
  consecutive_successes integer NOT NULL DEFAULT 0,
  open_incident_id uuid,
  last_checked_at timestamptz,
  next_check_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, slug),
  UNIQUE (tenant_id, id),
  CHECK ((kind = 'http') = (url IS NOT NULL)),
  CHECK (expected_status_min <= expected_status_max)
);
CREATE INDEX monitors_due ON monitors (next_check_at) WHERE NOT paused;

CREATE TABLE checks (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id uuid NOT NULL,
  monitor_id uuid NOT NULL,
  at timestamptz NOT NULL DEFAULT now(),
  ok boolean NOT NULL,
  status_code integer,
  latency_ms integer NOT NULL CHECK (latency_ms >= 0),
  -- A category, never the raw error: raw errors can reveal internal hosts and addresses.
  failure text CHECK (failure IN ('timeout', 'connection', 'status', 'content', 'tls', 'blocked')),
  tls_expires_at timestamptz,
  FOREIGN KEY (tenant_id, monitor_id) REFERENCES monitors (tenant_id, id) ON DELETE CASCADE,
  CHECK (ok = (failure IS NULL))
);
CREATE INDEX checks_monitor_at ON checks (monitor_id, at DESC);

CREATE TABLE incidents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  monitor_id uuid,
  title text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
  severity text NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high')),
  status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved')),
  automatic boolean NOT NULL DEFAULT false,
  -- Shown on the public status page. Fixed when the incident is created (automatic incidents copy
  -- their monitor's setting), so deleting a private monitor can't make its incidents public.
  public boolean NOT NULL DEFAULT false,
  started_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  UNIQUE (tenant_id, id),
  -- Deleting a monitor keeps its incident history.
  FOREIGN KEY (tenant_id, monitor_id) REFERENCES monitors (tenant_id, id) ON DELETE SET NULL (monitor_id),
  CHECK ((status = 'resolved') = (resolved_at IS NOT NULL))
);
CREATE INDEX incidents_started ON incidents (tenant_id, started_at DESC, id DESC);
ALTER TABLE monitors ADD FOREIGN KEY (tenant_id, open_incident_id) REFERENCES incidents (tenant_id, id) ON DELETE SET NULL (open_incident_id);

-- The incident timeline: what happened, and what the owner said about it. Internal notes stay off
-- the public status page.
CREATE TABLE incident_events (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id uuid NOT NULL,
  incident_id uuid NOT NULL,
  at timestamptz NOT NULL DEFAULT now(),
  kind text NOT NULL CHECK (kind IN ('opened', 'resolved', 'status', 'severity', 'comment')),
  message text NOT NULL CHECK (length(message) BETWEEN 1 AND 2000),
  public boolean NOT NULL DEFAULT true,
  author text NOT NULL,
  FOREIGN KEY (tenant_id, incident_id) REFERENCES incidents (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX incident_events_incident ON incident_events (incident_id, at);

CREATE TABLE sessions (
  token_hash text PRIMARY KEY,
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('owner', 'sandbox')),
  github_login text,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
  ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
  CREATE POLICY tenant_isolation ON tenants
    USING (id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (id = nullif(current_setting('app.tenant_id', true), '')::uuid);
  FOREACH t IN ARRAY ARRAY['monitors', 'checks', 'incidents', 'incident_events', 'sessions'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format($p$CREATE POLICY tenant_isolation ON %I
      USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
      WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)$p$, t);
  END LOOP;
END $$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO lighthouse_app;
GRANT SELECT, INSERT ON tenants TO lighthouse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON monitors, incidents, sessions TO lighthouse_app;
GRANT SELECT, INSERT ON checks, incident_events TO lighthouse_app;

-- The few questions asked before a tenant is known, each answering only what it must.

-- +goose StatementBegin
-- Monitors due for a check, across tenants: the scheduler then works inside each one's tenant.
CREATE FUNCTION lighthouse_due_monitors(p_limit integer)
  RETURNS TABLE (id uuid, tenant_id uuid) LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public
  AS $$ SELECT id, tenant_id FROM monitors WHERE NOT paused AND next_check_at <= now() ORDER BY next_check_at LIMIT p_limit $$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Which tenant a session cookie belongs to.
CREATE FUNCTION lighthouse_session(p_token_hash text)
  RETURNS TABLE (tenant_id uuid, role text, github_login text, expires_at timestamptz)
  LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public
  AS $$ SELECT tenant_id, role, github_login, expires_at FROM sessions WHERE token_hash = p_token_hash $$;
-- +goose StatementEnd

-- +goose StatementBegin
-- The owner tenant, created on first use.
CREATE FUNCTION lighthouse_owner_tenant(p_name text)
  RETURNS uuid LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = public
  AS $$
  DECLARE found uuid;
  BEGIN
    SELECT id INTO found FROM tenants WHERE kind = 'owner';
    IF found IS NULL THEN
      INSERT INTO tenants (kind, name) VALUES ('owner', p_name) ON CONFLICT DO NOTHING;
      SELECT id INTO found FROM tenants WHERE kind = 'owner';
    END IF;
    RETURN found;
  END $$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Housekeeping: old check results, expired sessions and sandboxes.
CREATE FUNCTION lighthouse_prune(p_keep_checks interval, p_keep_sandboxes interval)
  RETURNS TABLE (checks bigint, sandboxes bigint, sessions bigint)
  LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public
  AS $$
    WITH c AS (DELETE FROM checks WHERE at < now() - p_keep_checks RETURNING 1),
         t AS (DELETE FROM tenants WHERE kind = 'sandbox' AND created_at < now() - p_keep_sandboxes RETURNING 1),
         s AS (DELETE FROM sessions WHERE expires_at < now() RETURNING 1)
    SELECT (SELECT count(*) FROM c), (SELECT count(*) FROM t), (SELECT count(*) FROM s)
  $$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION lighthouse_due_monitors(integer), lighthouse_session(text), lighthouse_owner_tenant(text),
  lighthouse_prune(interval, interval) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION lighthouse_due_monitors(integer), lighthouse_session(text), lighthouse_owner_tenant(text),
  lighthouse_prune(interval, interval) TO lighthouse_app;

-- +goose Down
DROP FUNCTION lighthouse_prune(interval, interval);
DROP FUNCTION lighthouse_owner_tenant(text);
DROP FUNCTION lighthouse_session(text);
DROP FUNCTION lighthouse_due_monitors(integer);
DROP TABLE sessions, incident_events;
ALTER TABLE monitors DROP CONSTRAINT monitors_tenant_id_open_incident_id_fkey;
DROP TABLE incidents, checks, monitors, tenants;
