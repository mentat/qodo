// Geo data for the Mapbox-backed Risk globe.
//
// Each of the 42 Risk territories is rendered as a single GeoJSON Polygon or
// MultiPolygon feature. Most map cleanly to one or more real-world countries
// (looked up via the ADM0_A3 3-letter ISO code in Natural Earth's
// /public/world-110m.geojson). The cases where Risk carves a real country
// into sub-regions (Russia, USA, Canada, Australia) are handled by hand-authored
// polygons under SUB_NATIONAL_POLYGONS — accurate enough to read as the
// real region without shipping the multi-megabyte admin-1 (states +
// provinces) Natural Earth file.
//
// Output: buildTerritoryFeatures(countriesGeoJson) → FeatureCollection
// where each feature has `properties: { id, name, continent }` and a
// geometry. Mapbox uses the `id` for click + data-driven styling.

import { TERRITORIES, type TerritoryID, type ContinentID } from './board';

// ── 1. Country-code mapping ───────────────────────────────────────────────
//
// Maps each Risk territory to the list of ADM0_A3 (3-letter ISO) country
// codes whose Natural Earth geometries should be unioned to form the
// territory's shape. Set to an empty array for territories handled
// instead via SUB_NATIONAL_POLYGONS below.
export const RISK_COUNTRY_MAP: Record<TerritoryID, string[]> = {
  // North America — 7 of 9 are sub-national; only Greenland + Central Am map to countries.
  alaska: [],                                              // sub-national (USA-AK)
  'northwest-territory': [],                               // sub-national (CAN-YT/NT/NU)
  alberta: [],                                             // sub-national (CAN-BC/AB/SK/MB)
  ontario: [],                                             // sub-national (CAN-ON)
  quebec: [],                                              // sub-national (CAN-QC + Labrador)
  greenland: ['GRL'],
  'western-us': [],                                        // sub-national (USA west of ~100°W)
  'eastern-us': [],                                        // sub-national (USA east of ~100°W)
  'central-america': ['MEX', 'GTM', 'HND', 'SLV', 'NIC', 'CRI', 'PAN', 'BLZ', 'CUB', 'JAM', 'HTI', 'DOM', 'BHS'],

  // South America — clean country mappings.
  venezuela: ['VEN', 'GUY', 'SUR', 'TTO'],
  brazil: ['BRA'],
  peru: ['PER', 'ECU', 'COL', 'BOL', 'CHL'], // Peru territory absorbs Colombia/Ecuador/Bolivia/Chile in classic Risk
  argentina: ['ARG', 'URY', 'PRY', 'FLK'],

  // Europe — multi-country aggregates.
  iceland: ['ISL'],
  'great-britain': ['GBR', 'IRL'],
  scandinavia: ['SWE', 'NOR', 'FIN', 'DNK'],
  ukraine: ['UKR', 'BLR', 'MDA', 'LTU', 'LVA', 'EST'], // Ural handles western Russia separately
  'northern-europe': ['DEU', 'POL', 'CZE', 'SVK', 'AUT', 'CHE', 'NLD', 'BEL', 'LUX'],
  // Microstates omitted: Malta, Andorra, Monaco — Natural Earth 1:110m
  // drops features smaller than ~2k km². They get absorbed visually into the
  // surrounding territory at globe zoom.
  'southern-europe': ['ITA', 'GRC', 'HRV', 'SVN', 'BIH', 'SRB', 'MNE', 'MKD', 'ALB', 'KOS', 'BGR', 'ROU', 'HUN', 'CYP'],
  'western-europe': ['FRA', 'ESP', 'PRT'],

  // Africa — multi-country aggregates.
  'north-africa': ['MAR', 'DZA', 'TUN', 'LBY', 'SAH', 'MRT', 'MLI', 'NER', 'TCD', 'SDN', 'SEN', 'GMB', 'GNB', 'GIN', 'SLE', 'LBR', 'CIV', 'BFA', 'GHA', 'TGO', 'BEN', 'NGA', 'CMR', 'CAF'],
  egypt: ['EGY'],
  'east-africa': ['ETH', 'ERI', 'DJI', 'SOM', 'SOL', 'KEN', 'UGA', 'TZA', 'RWA', 'BDI', 'SDS'],
  congo: ['COD', 'COG', 'GAB', 'GNQ', 'AGO', 'ZMB'],
  'south-africa': ['ZAF', 'NAM', 'BWA', 'ZWE', 'MOZ', 'MWI', 'LSO', 'SWZ'],
  madagascar: ['MDG'],

  // Asia — multi-country aggregates + sub-national Russian splits.
  ural: [],          // sub-national (RUS, west of ~60°E)
  siberia: [],       // sub-national (RUS, 60°–110°E)
  yakutsk: [],       // sub-national (RUS, 110°–145°E north)
  irkutsk: [],       // sub-national (RUS, 90°–130°E south)
  kamchatka: [],     // sub-national (RUS, 145°–180°E peninsula)
  mongolia: ['MNG'],
  japan: ['JPN'],
  china: ['CHN', 'TWN', 'KOR', 'PRK'],
  afghanistan: ['AFG', 'TJK', 'KGZ', 'UZB', 'TKM', 'KAZ'],
  'middle-east': ['IRN', 'IRQ', 'SAU', 'ARE', 'KWT', 'QAT', 'YEM', 'OMN', 'SYR', 'LBN', 'JOR', 'ISR', 'PSX', 'TUR', 'GEO', 'ARM', 'AZE'],
  india: ['IND', 'PAK', 'BGD', 'LKA', 'BTN', 'NPL'],
  // Singapore omitted (microstate not in 110m).
  siam: ['THA', 'LAO', 'KHM', 'VNM', 'MMR', 'MYS', 'BRN', 'PHL'],

  // Australia — Indonesia + AU split.
  indonesia: ['IDN', 'TLS'],         // Indonesia includes western New Guinea through IDN.
  'new-guinea': ['PNG'],             // Natural Earth gives the eastern half + PNG islands cleanly.
  'western-australia': [],           // sub-national (AUS west of ~140°E)
  'eastern-australia': [],           // sub-national (AUS east of ~140°E + NZL + Pacific)
};

// ── 2. Hand-authored sub-national polygons ────────────────────────────────
//
// Reasonably-detailed polygon outlines for the 12 Risk territories that
// don't map to whole countries. Coordinates are [lon, lat], drawn clockwise
// (Mapbox/GeoJSON convention for outer rings). Vertex counts are chosen so
// each region is recognizable without being surveyor-grade.

type PolyRing = [number, number][];

const ALASKA: PolyRing = [
  // Bering coast → Gulf of Alaska → Canada border → Arctic slope.
  [-170.5, 63.2], [-168.1, 61.5], [-165.6, 60.1], [-162.2, 59.7],
  [-159.6, 58.9], [-156.2, 56.8], [-153.2, 57.1], [-151.7, 59.4],
  [-149.2, 60.1], [-146.0, 60.4], [-141.0, 60.0], [-141.0, 69.7],
  [-145.0, 70.1], [-150.5, 70.4], [-155.1, 71.2], [-160.1, 70.5],
  [-164.2, 68.9], [-167.3, 66.1], [-170.5, 63.2],
];

const NORTHWEST_TERRITORY: PolyRing = [
  // Yukon/NWT/Nunavut mass. South edge is shared with Alberta/Ontario.
  [-141.0, 60.0], [-141.0, 69.7], [-132.0, 69.4], [-124.0, 70.4],
  [-116.0, 72.2], [-105.0, 73.7], [-94.0, 72.2], [-86.0, 69.2],
  [-80.0, 65.5], [-79.0, 59.0], [-83.0, 57.0], [-88.0, 58.0],
  [-95.0, 60.0], [-112.0, 60.0], [-128.0, 60.0], [-141.0, 60.0],
];

const ALBERTA: PolyRing = [
  // Western Canada: BC coast plus prairie provinces to the Ontario split.
  [-141.0, 60.0], [-136.0, 59.2], [-132.0, 56.0], [-130.0, 53.0],
  [-128.2, 51.2], [-124.9, 49.5], [-122.8, 49.0], [-110.0, 49.0],
  [-95.0, 49.0], [-95.0, 60.0], [-112.0, 60.0], [-128.0, 60.0],
  [-141.0, 60.0],
];

const ONTARIO: PolyRing = [
  // Hudson Bay shore → Manitoba/Ontario border → Great Lakes → Quebec line.
  [-95.0, 60.0], [-95.0, 49.0], [-91.0, 48.2], [-88.5, 47.6],
  [-84.8, 46.6], [-82.4, 44.2], [-79.2, 43.2], [-77.8, 44.6],
  [-79.0, 50.8], [-79.0, 56.0], [-83.0, 57.0], [-88.0, 58.0],
  [-95.0, 60.0],
];

const QUEBEC: PolyRing = [
  // Quebec + Labrador, with the St. Lawrence/Labrador coast pulled east.
  [-79.0, 56.0], [-79.0, 50.8], [-77.8, 44.6], [-74.7, 45.0],
  [-71.7, 46.2], [-69.5, 47.6], [-67.2, 49.0], [-64.0, 49.8],
  [-60.0, 50.2], [-56.0, 52.2], [-55.5, 54.0], [-58.5, 55.4],
  [-62.5, 56.7], [-67.5, 58.4], [-73.0, 59.0], [-77.0, 58.0],
  [-79.0, 56.0],
];

const WESTERN_US: PolyRing = [
  // Pacific coast → US/Mexico border → 100W split → Canadian border.
  [-124.8, 48.6], [-124.4, 46.2], [-124.0, 43.2], [-123.0, 40.5],
  [-122.0, 38.0], [-121.0, 36.0], [-119.5, 34.5], [-117.1, 32.5],
  [-114.8, 32.7], [-111.0, 31.3], [-106.5, 31.8], [-104.5, 29.6],
  [-100.0, 28.8], [-100.0, 49.0], [-117.0, 49.0], [-124.8, 48.6],
];

const EASTERN_US: PolyRing = [
  // Shared 100W split → Gulf/Atlantic coast → Great Lakes back west.
  [-100.0, 49.0], [-100.0, 28.8], [-97.0, 27.8], [-95.0, 28.8],
  [-91.0, 29.1], [-88.0, 30.3], [-84.5, 29.9], [-81.1, 25.2],
  [-80.0, 27.5], [-78.5, 32.5], [-76.0, 36.0], [-74.2, 40.0],
  [-70.7, 42.0], [-67.0, 44.7], [-75.0, 44.4],
  [-79.2, 43.2], [-82.4, 44.2], [-84.8, 46.6], [-91.0, 48.2],
  [-95.0, 49.0], [-100.0, 49.0],
];

// Russia splits. Adjacent territories share boundary segments so Mapbox fill
// layers do not visibly double-paint or flicker on hover.
const URAL: PolyRing = [
  [30.0, 75.0], [60.0, 75.0], [60.0, 55.0], [55.0, 50.0],
  [45.0, 47.0], [35.0, 50.0], [30.0, 55.0], [30.0, 75.0],
];

const SIBERIA: PolyRing = [
  [60.0, 75.0], [110.0, 78.0], [110.0, 62.0], [100.0, 58.0],
  [95.0, 50.0], [80.0, 50.0], [60.0, 55.0], [60.0, 75.0],
];

const IRKUTSK: PolyRing = [
  [95.0, 50.0], [100.0, 58.0], [110.0, 62.0], [120.0, 58.0],
  [120.0, 50.0], [105.0, 49.4], [95.0, 50.0],
];

const YAKUTSK: PolyRing = [
  [110.0, 78.0], [145.0, 76.0], [145.0, 62.0], [132.0, 60.0],
  [120.0, 58.0], [110.0, 62.0], [110.0, 78.0],
];

const KAMCHATKA: PolyRing = [
  [145.0, 62.0], [156.0, 64.5], [165.0, 63.0], [171.0, 59.5],
  [166.0, 53.5], [160.0, 51.0], [155.0, 54.0], [145.0, 56.0],
  [145.0, 62.0],
];

const WESTERN_AUSTRALIA: PolyRing = [
  // WA coast with the classic Risk split near 138E.
  [113.2, -22.0], [114.0, -29.0], [115.2, -34.8], [121.5, -34.2],
  [128.0, -31.7], [134.0, -32.7], [138.0, -34.0], [138.0, -25.5],
  [134.5, -18.0], [129.0, -13.0], [124.5, -14.2], [119.0, -18.0],
  [113.2, -22.0],
];

const EASTERN_AUSTRALIA: PolyRing = [
  // Eastern AU coast, Bass Strait edge, and the shared 138E interior split.
  [138.0, -34.0], [141.0, -38.4], [146.5, -39.0], [151.5, -33.0],
  [153.5, -27.0], [150.8, -22.0], [146.0, -16.0], [142.0, -10.5],
  [138.0, -16.0], [138.0, -25.5], [138.0, -34.0],
];

export const SUB_NATIONAL_POLYGONS: Partial<Record<TerritoryID, PolyRing>> = {
  alaska: ALASKA,
  'northwest-territory': NORTHWEST_TERRITORY,
  alberta: ALBERTA,
  ontario: ONTARIO,
  quebec: QUEBEC,
  'western-us': WESTERN_US,
  'eastern-us': EASTERN_US,
  ural: URAL,
  siberia: SIBERIA,
  yakutsk: YAKUTSK,
  irkutsk: IRKUTSK,
  kamchatka: KAMCHATKA,
  'western-australia': WESTERN_AUSTRALIA,
  'eastern-australia': EASTERN_AUSTRALIA,
};

// ── 3. GeoJSON types (light) ──────────────────────────────────────────────

type PolygonCoords = number[][][];     // [ring, ...]; ring = [[lon, lat], ...]
type MultiPolygonCoords = number[][][][]; // [polygon, ...]; polygon = [ring, ...]

type PolygonGeom = { type: 'Polygon'; coordinates: PolygonCoords };
type MultiPolygonGeom = { type: 'MultiPolygon'; coordinates: MultiPolygonCoords };
export type Geom = PolygonGeom | MultiPolygonGeom;

export interface CountryFeature {
  type: 'Feature';
  properties: { ADM0_A3: string; NAME: string; [k: string]: unknown };
  geometry: Geom;
}

export interface CountriesGeoJson {
  type: 'FeatureCollection';
  features: CountryFeature[];
}

export interface TerritoryFeature {
  type: 'Feature';
  id: TerritoryID;
  properties: {
    id: TerritoryID;
    name: string;
    continent: ContinentID;
    ownerId: string;
    armies: number;
    color: string;
  };
  geometry: Geom;
}

export interface TerritoryFeatureCollection {
  type: 'FeatureCollection';
  features: TerritoryFeature[];
}

// ── 4. Building features ──────────────────────────────────────────────────

/**
 * Concatenate one Polygon or MultiPolygon's outer-ring coordinates into a
 * single MultiPolygon's coordinates array. We don't attempt true geometric
 * union (no spatial library) — countries that share a border just sit next
 * to each other as separate polygons in a MultiPolygon, which renders
 * identically for fill layers.
 */
function appendGeometryRings(out: MultiPolygonCoords, geom: Geom) {
  if (geom.type === 'Polygon') {
    out.push(geom.coordinates);
  } else {
    for (const poly of geom.coordinates) {
      out.push(poly);
    }
  }
}

/**
 * Build the 42 Risk territory features from the loaded Natural Earth
 * country GeoJSON. The output is a single FeatureCollection ready to feed
 * to a Mapbox <Source>.
 *
 * The returned features' `properties` are placeholders for owner/color/
 * armies — the Globe component overwrites these on every render so Mapbox's
 * data-driven styling picks them up.
 */
export function buildTerritoryFeatures(countries: CountriesGeoJson): TerritoryFeatureCollection {
  // Index the country features by ADM0_A3 for fast lookup.
  const byCode = new Map<string, CountryFeature>();
  for (const f of countries.features) {
    const code = String(f.properties?.ADM0_A3 ?? '');
    if (code) byCode.set(code, f);
  }

  const out: TerritoryFeature[] = [];
  for (const def of TERRITORIES) {
    const tid = def.id;
    let geometry: Geom;

    // 1) Hand-authored polygons take precedence.
    const custom = SUB_NATIONAL_POLYGONS[tid];
    if (custom) {
      geometry = { type: 'Polygon', coordinates: [closeRing(custom)] };
    } else {
      // 2) Look up country codes; concat into a MultiPolygon.
      const codes = RISK_COUNTRY_MAP[tid] ?? [];
      const coords: MultiPolygonCoords = [];
      for (const code of codes) {
        const f = byCode.get(code);
        if (!f) {
          // eslint-disable-next-line no-console
          console.warn(`risk territory ${tid}: missing country ${code} in GeoJSON`);
          continue;
        }
        appendGeometryRings(coords, f.geometry);
      }
      geometry = coords.length === 1
        ? { type: 'Polygon', coordinates: coords[0] }
        : { type: 'MultiPolygon', coordinates: coords };
    }

    out.push({
      type: 'Feature',
      id: tid,
      properties: {
        id: tid,
        name: def.name,
        continent: def.continent,
        ownerId: '',
        armies: 0,
        color: '#666',
      },
      geometry,
    });
  }
  return { type: 'FeatureCollection', features: out };
}

/** Ensure a ring is closed (first vertex === last vertex per GeoJSON spec). */
function closeRing(ring: PolyRing): PolyRing {
  if (ring.length === 0) return ring;
  const a = ring[0];
  const b = ring[ring.length - 1];
  if (a[0] === b[0] && a[1] === b[1]) return ring;
  return [...ring, [a[0], a[1]]];
}

// ── 5. Lazy loader ────────────────────────────────────────────────────────

let cached: TerritoryFeatureCollection | null = null;

/**
 * Fetches the Natural Earth countries file from /world-110m.geojson and
 * returns the built Risk territory features. Idempotent — the result is
 * cached in module scope so re-renders don't re-fetch.
 */
export async function loadTerritoryFeatures(): Promise<TerritoryFeatureCollection> {
  if (cached) return cached;
  const res = await fetch('/world-110m.geojson');
  if (!res.ok) {
    throw new Error(`failed to load world-110m.geojson: ${res.status}`);
  }
  const data = (await res.json()) as CountriesGeoJson;
  cached = buildTerritoryFeatures(data);
  return cached;
}
