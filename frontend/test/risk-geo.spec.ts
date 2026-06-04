import { describe, expect, it } from 'bun:test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import {
  buildTerritoryFeatures,
  RISK_COUNTRY_MAP,
  SUB_NATIONAL_POLYGONS,
  type CountriesGeoJson,
} from '../src/components/Risk/territoryGeo';
import { TERRITORIES } from '../src/components/Risk/board';

// Load the same GeoJSON the running app reads from /world-110m.geojson.
function loadCountries(): CountriesGeoJson {
  const path = resolve(__dirname, '../public/world-110m.geojson');
  const raw = readFileSync(path, 'utf8');
  return JSON.parse(raw) as CountriesGeoJson;
}

describe('risk territory geometries', () => {
  const countries = loadCountries();
  const fc = buildTerritoryFeatures(countries);

  it('produces exactly 42 features (one per Risk territory)', () => {
    expect(fc.features.length).toBe(42);
  });

  it('every Risk territory id is covered by either ISO codes or a custom polygon', () => {
    for (const t of TERRITORIES) {
      const codes = RISK_COUNTRY_MAP[t.id] ?? [];
      const custom = SUB_NATIONAL_POLYGONS[t.id];
      expect(
        codes.length > 0 || custom != null,
      ).toBe(true);
    }
  });

  it('every feature has non-empty geometry coordinates', () => {
    const empty = fc.features.filter((f) => {
      const c = f.geometry.coordinates as unknown[];
      return !Array.isArray(c) || c.length === 0;
    });
    if (empty.length) {
      // Surface which ones failed.
      throw new Error('empty geometries: ' + empty.map((f) => f.properties.id).join(', '));
    }
    expect(empty.length).toBe(0);
  });

  it('every ISO code in RISK_COUNTRY_MAP resolves to a country feature', () => {
    const known = new Set(countries.features.map((f) => String(f.properties.ADM0_A3 ?? '')));
    const missing: string[] = [];
    for (const [tid, codes] of Object.entries(RISK_COUNTRY_MAP)) {
      for (const code of codes) {
        if (!known.has(code)) missing.push(`${tid} → ${code}`);
      }
    }
    if (missing.length) {
      throw new Error('unknown country codes: ' + missing.join(', '));
    }
    expect(missing.length).toBe(0);
  });

  it('does not assign the same country geometry to multiple Risk territories', () => {
    const seen = new Map<string, string>();
    const duplicates: string[] = [];
    for (const [tid, codes] of Object.entries(RISK_COUNTRY_MAP)) {
      for (const code of codes) {
        const first = seen.get(code);
        if (first) duplicates.push(`${code}: ${first} + ${tid}`);
        else seen.set(code, tid);
      }
    }
    if (duplicates.length) {
      throw new Error('duplicate country assignments: ' + duplicates.join(', '));
    }
    expect(duplicates.length).toBe(0);
  });

  it('keeps hand-authored split polygons from overlapping interiors', () => {
    const entries = Object.entries(SUB_NATIONAL_POLYGONS) as Array<[string, Ring]>;
    const overlaps: string[] = [];
    for (let i = 0; i < entries.length; i++) {
      for (let j = i + 1; j < entries.length; j++) {
        const [aID, a] = entries[i];
        const [bID, b] = entries[j];
        if (ringsOverlap(a, b)) overlaps.push(`${aID} + ${bID}`);
      }
    }
    if (overlaps.length) {
      throw new Error('overlapping custom polygons: ' + overlaps.join(', '));
    }
    expect(overlaps.length).toBe(0);
  });
});

type Point = [number, number];
type Ring = Point[];

const EPS = 1e-9;

function closeRing(ring: Ring): Ring {
  if (ring.length === 0) return ring;
  const first = ring[0];
  const last = ring[ring.length - 1];
  if (samePoint(first, last)) return ring;
  return [...ring, first];
}

function ringsOverlap(aRaw: Ring, bRaw: Ring): boolean {
  const a = closeRing(aRaw);
  const b = closeRing(bRaw);

  for (const p of a.slice(0, -1)) {
    if (!pointOnRing(p, b) && pointInRing(p, b)) return true;
  }
  for (const p of b.slice(0, -1)) {
    if (!pointOnRing(p, a) && pointInRing(p, a)) return true;
  }

  for (let i = 0; i < a.length - 1; i++) {
    for (let j = 0; j < b.length - 1; j++) {
      if (segmentsProperlyIntersect(a[i], a[i + 1], b[j], b[j + 1])) return true;
    }
  }
  return false;
}

function pointInRing(point: Point, ring: Ring): boolean {
  let inside = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [xi, yi] = ring[i];
    const [xj, yj] = ring[j];
    const crosses = ((yi > point[1]) !== (yj > point[1])) &&
      point[0] < ((xj - xi) * (point[1] - yi)) / (yj - yi) + xi;
    if (crosses) inside = !inside;
  }
  return inside;
}

function pointOnRing(point: Point, ring: Ring): boolean {
  const closed = closeRing(ring);
  for (let i = 0; i < closed.length - 1; i++) {
    if (pointOnSegment(point, closed[i], closed[i + 1])) return true;
  }
  return false;
}

function pointOnSegment(p: Point, a: Point, b: Point): boolean {
  const cross = orientation(a, b, p);
  if (Math.abs(cross) > EPS) return false;
  return p[0] >= Math.min(a[0], b[0]) - EPS &&
    p[0] <= Math.max(a[0], b[0]) + EPS &&
    p[1] >= Math.min(a[1], b[1]) - EPS &&
    p[1] <= Math.max(a[1], b[1]) + EPS;
}

function segmentsProperlyIntersect(a: Point, b: Point, c: Point, d: Point): boolean {
  const o1 = orientation(a, b, c);
  const o2 = orientation(a, b, d);
  const o3 = orientation(c, d, a);
  const o4 = orientation(c, d, b);
  return o1 * o2 < -EPS && o3 * o4 < -EPS;
}

function orientation(a: Point, b: Point, c: Point): number {
  return (b[0] - a[0]) * (c[1] - a[1]) - (b[1] - a[1]) * (c[0] - a[0]);
}

function samePoint(a: Point, b: Point): boolean {
  return Math.abs(a[0] - b[0]) < EPS && Math.abs(a[1] - b[1]) < EPS;
}
