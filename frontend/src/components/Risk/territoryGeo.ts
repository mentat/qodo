// Geo data for the Mapbox-backed Risk globe.
//
// Each of the 42 Risk territories is rendered as a single GeoJSON Polygon or
// MultiPolygon feature. Most map cleanly to one or more real-world countries
// (looked up via the ADM0_A3 3-letter ISO code in Natural Earth's
// /public/world-110m.geojson). The cases where Risk carves a real country
// into sub-regions (Russia, USA, Canada) are handled by hand-authored
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
// territory's shape. Set to an empty array for the 12 territories handled
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
  indonesia: ['IDN', 'TLS', 'PNG'],  // PNG joins Indonesia in some Risk editions, but here it's in NewGuinea
  'new-guinea': [],                  // sub-national: eastern half of PNG (we draw it explicitly)
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
  [-168, 60], [-162, 54.5], [-156, 56], [-152, 60], [-148, 60.5],
  [-141, 60], [-141, 69.6], [-156, 71.4], [-165, 68.5], [-168, 65.5], [-168, 60],
];

const NORTHWEST_TERRITORY: PolyRing = [
  [-141, 60], [-141, 70.5], [-128, 70], [-115, 73], [-100, 73.5],
  [-85, 70.5], [-77, 67], [-75, 62], [-85, 60], [-102, 60], [-120, 60], [-141, 60],
];

const ALBERTA: PolyRing = [
  // BC + AB + SK + MB
  [-141, 60], [-141, 56.5], [-132, 54.5], [-130, 51], [-123, 49],
  [-114, 49], [-95, 49], [-95, 60], [-141, 60],
];

const ONTARIO: PolyRing = [
  // ON province incl. Hudson Bay south shore
  [-95, 60], [-95, 49], [-89, 48], [-84.5, 46.5], [-82, 43],
  [-79, 43], [-79, 51], [-83, 55], [-87, 56], [-95, 60],
];

const QUEBEC: PolyRing = [
  // QC + Labrador
  [-79, 51], [-79, 45.5], [-74, 45], [-69, 47.5], [-67, 49],
  [-64, 50], [-57, 53.5], [-55.5, 52], [-58, 50], [-65, 50],
  [-69, 52], [-72, 55], [-77, 56], [-79, 56], [-79, 51],
];

const WESTERN_US: PolyRing = [
  // Pacific coast → Mexico border → ~100°W meridian → Canadian border
  [-124.5, 48.5], [-124, 42], [-122, 36], [-120, 34],
  [-117, 32.5], [-110, 31.3], [-106, 31.8], [-103, 28.7],
  [-100, 28.5], [-100, 49], [-117, 49], [-124.5, 48.5],
];

const EASTERN_US: PolyRing = [
  // 100°W meridian → Gulf → Florida → Atlantic → Maine → Great Lakes back to 100°W
  [-100, 49], [-100, 28.5], [-97, 27.5], [-95, 29], [-90, 29.2],
  [-87, 30.4], [-83.5, 30], [-81, 25.2], [-80, 27], [-78, 33.7],
  [-75.5, 36.9], [-74, 39.5], [-71, 41.5], [-67, 44.6], [-69, 47.5],
  [-75, 45], [-79, 43], [-82, 43], [-84.5, 46.5], [-89, 48], [-95, 49], [-100, 49],
];

// Russia splits — rough rectangles bounded by recognizable lat/lon ranges
const URAL: PolyRing = [
  [30, 75], [60, 75], [60, 50], [55, 47], [40, 47], [30, 50], [30, 75],
];

const SIBERIA: PolyRing = [
  [60, 75], [110, 78], [110, 65], [105, 55], [95, 50], [80, 50], [60, 50], [60, 75],
];

const YAKUTSK: PolyRing = [
  [110, 78], [145, 78], [145, 65], [140, 60], [120, 60], [110, 65], [110, 78],
];

const IRKUTSK: PolyRing = [
  [95, 65], [120, 65], [120, 50], [105, 50], [95, 50], [95, 65],
];

const KAMCHATKA: PolyRing = [
  [145, 65], [165, 65], [170, 60], [165, 53], [157, 51], [155, 56], [145, 58], [145, 65],
];

// New Guinea split — east half of Papua New Guinea (Indonesia handles west).
const NEW_GUINEA: PolyRing = [
  [141, -2], [151, -2], [156, -7], [150, -11], [141, -9], [141, -2],
];

const WESTERN_AUSTRALIA: PolyRing = [
  [113, -22], [113, -34], [126, -32], [135, -34], [138, -34], [138, -26], [135, -16], [130, -12], [126, -14], [113, -22],
];

const EASTERN_AUSTRALIA: PolyRing = [
  [138, -26], [138, -39], [147, -43], [153, -28], [145, -16], [142, -10], [138, -16], [138, -26],
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
  'new-guinea': NEW_GUINEA,
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
