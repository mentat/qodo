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
});
