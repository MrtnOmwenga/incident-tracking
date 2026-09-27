-- Visit analytics without cookies or stored IP addresses.
--
-- A visitor is identified only by an HMAC of their IP address and user agent, keyed with a random
-- salt that exists for one day. The same person can't be linked across days, and the IP can't be
-- recovered from the hash once the salt is gone. Page views are keyed by an ID the browser makes
-- up per page load, so engaged-time heartbeats and events can be attached to them.

-- +goose Up
-- Salts are never readable by the API role; lighthouse_daily_salt hands out today's only.
CREATE TABLE analytics_salts (
  day date PRIMARY KEY,
  salt bytea NOT NULL CHECK (length(salt) = 32)
);
REVOKE ALL ON analytics_salts FROM PUBLIC;

CREATE TABLE page_views (
  id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  at timestamptz NOT NULL DEFAULT now(),
  day date NOT NULL,
  visitor text NOT NULL CHECK (visitor ~ '^[0-9a-f]{32}$'),
  path text NOT NULL CHECK (path ~ '^/[a-z0-9/_-]{0,120}$'),
  project text CHECK (project ~ '^[a-z0-9][a-z0-9-]{0,49}$'),
  -- A private campaign tag from ?ref= (e.g. one per job application).
  ref text CHECK (ref ~ '^[a-z0-9][a-z0-9-]{0,39}$'),
  referrer text CHECK (length(referrer) <= 100),
  device text NOT NULL CHECK (device IN ('mobile', 'tablet', 'desktop')),
  engaged_seconds integer NOT NULL DEFAULT 0 CHECK (engaged_seconds BETWEEN 0 AND 3600),
  last_ping timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE INDEX page_views_day ON page_views (tenant_id, day);
CREATE INDEX page_views_visitor ON page_views (tenant_id, day, visitor);

CREATE TABLE analytics_events (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id uuid NOT NULL,
  view_id uuid NOT NULL,
  at timestamptz NOT NULL DEFAULT now(),
  name text NOT NULL CHECK (name IN ('demo_ready', 'demo_open', 'intro_skip')),
  -- One of each per page view, so a script can't inflate the counts.
  UNIQUE (view_id, name),
  FOREIGN KEY (tenant_id, view_id) REFERENCES page_views (tenant_id, id) ON DELETE CASCADE
);

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['page_views', 'analytics_events'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format($p$CREATE POLICY tenant_isolation ON %I
      USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
      WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)$p$, t);
  END LOOP;
END $$;
-- +goose StatementEnd

GRANT SELECT, INSERT ON page_views, analytics_events TO lighthouse_app;
-- Heartbeats may only move the engaged time forward.
GRANT UPDATE (engaged_seconds, last_ping) ON page_views TO lighthouse_app;

-- +goose StatementBegin
-- Today's salt, created on first use from the server's strong random source. Older salts are
-- deleted as soon as a new day starts using its own, which is what makes yesterday's visitor
-- hashes unlinkable to anything.
CREATE FUNCTION lighthouse_daily_salt(p_day date)
  RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = public
  AS $$
  DECLARE s bytea;
  BEGIN
    INSERT INTO analytics_salts (day, salt)
      VALUES (p_day, uuid_send(gen_random_uuid()) || uuid_send(gen_random_uuid()))
      ON CONFLICT (day) DO NOTHING;
    DELETE FROM analytics_salts WHERE day < p_day;
    SELECT salt INTO s FROM analytics_salts WHERE day = p_day;
    RETURN s;
  END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION lighthouse_prune_analytics(p_keep interval)
  RETURNS bigint LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public
  AS $$
    WITH v AS (DELETE FROM page_views WHERE at < now() - p_keep RETURNING 1)
    SELECT count(*) FROM v
  $$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION lighthouse_daily_salt(date), lighthouse_prune_analytics(interval) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION lighthouse_daily_salt(date), lighthouse_prune_analytics(interval) TO lighthouse_app;

-- +goose Down
DROP FUNCTION lighthouse_prune_analytics(interval);
DROP FUNCTION lighthouse_daily_salt(date);
DROP TABLE analytics_events, page_views, analytics_salts;
