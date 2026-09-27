import { describe, expect, it } from 'vitest';
import { ago, duration, pct, slugify } from '../format';

describe('format', () => {
  it('turns a company or role into a ?ref= tag', () => {
    expect(slugify('Acme, backend role')).toBe('acme-backend-role');
    expect(slugify('  Zürich GmbH!! ')).toBe('zurich-gmbh');
    expect(slugify('---')).toBe('');
    expect(slugify('x'.repeat(80))).toHaveLength(40);
    // Tags are what the server accepts: lowercase letters, digits and dashes, no dash at the end.
    for (const text of ['Acme Corp', 'a--b', 'ÆØÅ Ltd', 'Role (senior)', `${'a'.repeat(39)}-b`]) {
      expect(slugify(text)).toMatch(/^([a-z0-9][a-z0-9-]{0,38}[a-z0-9]|[a-z0-9]?)$/);
    }
  });

  it('never rounds an imperfect uptime up to 100%', () => {
    expect(pct(100)).toBe('100%');
    expect(pct(99.99)).toBe('99.99%');
    expect(pct(null)).toBe('–');
  });

  it('says how long ago, and how long', () => {
    const now = Date.parse('2026-09-28T12:00:00Z');
    expect(ago('2026-09-28T11:59:58Z', now)).toBe('just now');
    expect(ago('2026-09-28T11:55:00Z', now)).toBe('5 min ago');
    expect(ago(null, now)).toBe('never');
    expect(duration(45)).toBe('45 s');
    expect(duration(3725)).toBe('1 h 2 min');
  });
});
