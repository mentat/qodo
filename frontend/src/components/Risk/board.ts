// Static Risk-map metadata for the frontend. Mirrors api/services/risk/board.go
// exactly — the Go file is the source of truth for the engine; this file
// drives the 3D globe placement, adjacency hints, and HUD labels.

export type TerritoryID =
  | 'alaska' | 'northwest-territory' | 'alberta' | 'ontario' | 'quebec'
  | 'greenland' | 'western-us' | 'eastern-us' | 'central-america'
  | 'venezuela' | 'brazil' | 'peru' | 'argentina'
  | 'iceland' | 'great-britain' | 'scandinavia' | 'ukraine'
  | 'northern-europe' | 'southern-europe' | 'western-europe'
  | 'north-africa' | 'egypt' | 'east-africa' | 'congo'
  | 'south-africa' | 'madagascar'
  | 'ural' | 'siberia' | 'yakutsk' | 'irkutsk' | 'kamchatka'
  | 'mongolia' | 'japan' | 'china' | 'afghanistan' | 'middle-east'
  | 'india' | 'siam'
  | 'indonesia' | 'new-guinea' | 'western-australia' | 'eastern-australia';

export type ContinentID =
  | 'north-america' | 'south-america' | 'europe'
  | 'africa' | 'asia' | 'australia';

export interface TerritoryDef {
  id: TerritoryID;
  name: string;
  continent: ContinentID;
  lat: number;
  lon: number;
  adjacent: TerritoryID[];
}

export interface ContinentDef {
  id: ContinentID;
  name: string;
  bonus: number;
  color: string; // Mantine palette key for shading owned territories
}

export const CONTINENTS: ContinentDef[] = [
  { id: 'north-america', name: 'North America', bonus: 5, color: 'electricBlue' },
  { id: 'south-america', name: 'South America', bonus: 2, color: 'hotYellow' },
  { id: 'europe', name: 'Europe', bonus: 5, color: 'synthPurple' },
  { id: 'africa', name: 'Africa', bonus: 3, color: 'neonPink' },
  { id: 'asia', name: 'Asia', bonus: 7, color: 'neonGreen' },
  { id: 'australia', name: 'Australia', bonus: 2, color: 'hotYellow' },
];

export const TERRITORIES: TerritoryDef[] = [
  // North America
  { id: 'alaska', name: 'Alaska', continent: 'north-america', lat: 65, lon: -150,
    adjacent: ['northwest-territory', 'alberta', 'kamchatka'] },
  { id: 'northwest-territory', name: 'NW Territory', continent: 'north-america', lat: 70, lon: -110,
    adjacent: ['alaska', 'alberta', 'ontario', 'greenland'] },
  { id: 'alberta', name: 'Alberta', continent: 'north-america', lat: 55, lon: -115,
    adjacent: ['alaska', 'northwest-territory', 'ontario', 'western-us'] },
  { id: 'ontario', name: 'Ontario', continent: 'north-america', lat: 55, lon: -90,
    adjacent: ['northwest-territory', 'alberta', 'western-us', 'eastern-us', 'quebec', 'greenland'] },
  { id: 'quebec', name: 'Quebec', continent: 'north-america', lat: 55, lon: -75,
    adjacent: ['ontario', 'eastern-us', 'greenland'] },
  { id: 'greenland', name: 'Greenland', continent: 'north-america', lat: 75, lon: -40,
    adjacent: ['northwest-territory', 'ontario', 'quebec', 'iceland'] },
  { id: 'western-us', name: 'Western US', continent: 'north-america', lat: 38, lon: -110,
    adjacent: ['alberta', 'ontario', 'eastern-us', 'central-america'] },
  { id: 'eastern-us', name: 'Eastern US', continent: 'north-america', lat: 38, lon: -85,
    adjacent: ['ontario', 'quebec', 'western-us', 'central-america'] },
  { id: 'central-america', name: 'Central America', continent: 'north-america', lat: 17, lon: -90,
    adjacent: ['western-us', 'eastern-us', 'venezuela'] },

  // South America
  { id: 'venezuela', name: 'Venezuela', continent: 'south-america', lat: 8, lon: -65,
    adjacent: ['central-america', 'brazil', 'peru'] },
  { id: 'brazil', name: 'Brazil', continent: 'south-america', lat: -10, lon: -55,
    adjacent: ['venezuela', 'peru', 'argentina', 'north-africa'] },
  { id: 'peru', name: 'Peru', continent: 'south-america', lat: -10, lon: -75,
    adjacent: ['venezuela', 'brazil', 'argentina'] },
  { id: 'argentina', name: 'Argentina', continent: 'south-america', lat: -35, lon: -65,
    adjacent: ['peru', 'brazil'] },

  // Europe
  { id: 'iceland', name: 'Iceland', continent: 'europe', lat: 65, lon: -19,
    adjacent: ['greenland', 'great-britain', 'scandinavia'] },
  { id: 'great-britain', name: 'Great Britain', continent: 'europe', lat: 54, lon: -3,
    adjacent: ['iceland', 'scandinavia', 'northern-europe', 'western-europe'] },
  { id: 'scandinavia', name: 'Scandinavia', continent: 'europe', lat: 62, lon: 15,
    adjacent: ['iceland', 'great-britain', 'northern-europe', 'ukraine'] },
  { id: 'ukraine', name: 'Ukraine', continent: 'europe', lat: 50, lon: 30,
    adjacent: ['scandinavia', 'northern-europe', 'southern-europe', 'ural', 'afghanistan', 'middle-east'] },
  { id: 'northern-europe', name: 'Northern Europe', continent: 'europe', lat: 52, lon: 15,
    adjacent: ['great-britain', 'scandinavia', 'ukraine', 'southern-europe', 'western-europe'] },
  { id: 'southern-europe', name: 'Southern Europe', continent: 'europe', lat: 44, lon: 12,
    adjacent: ['northern-europe', 'ukraine', 'western-europe', 'north-africa', 'egypt', 'middle-east'] },
  { id: 'western-europe', name: 'Western Europe', continent: 'europe', lat: 46, lon: 2,
    adjacent: ['great-britain', 'northern-europe', 'southern-europe', 'north-africa'] },

  // Africa
  { id: 'north-africa', name: 'North Africa', continent: 'africa', lat: 22, lon: 8,
    adjacent: ['brazil', 'western-europe', 'southern-europe', 'egypt', 'east-africa', 'congo'] },
  { id: 'egypt', name: 'Egypt', continent: 'africa', lat: 27, lon: 30,
    adjacent: ['southern-europe', 'north-africa', 'east-africa', 'middle-east'] },
  { id: 'east-africa', name: 'East Africa', continent: 'africa', lat: 5, lon: 35,
    adjacent: ['egypt', 'north-africa', 'congo', 'south-africa', 'madagascar', 'middle-east'] },
  { id: 'congo', name: 'Congo', continent: 'africa', lat: -2, lon: 22,
    adjacent: ['north-africa', 'east-africa', 'south-africa'] },
  { id: 'south-africa', name: 'South Africa', continent: 'africa', lat: -28, lon: 25,
    adjacent: ['congo', 'east-africa', 'madagascar'] },
  { id: 'madagascar', name: 'Madagascar', continent: 'africa', lat: -20, lon: 47,
    adjacent: ['east-africa', 'south-africa'] },

  // Asia
  { id: 'ural', name: 'Ural', continent: 'asia', lat: 60, lon: 60,
    adjacent: ['ukraine', 'siberia', 'china', 'afghanistan'] },
  { id: 'siberia', name: 'Siberia', continent: 'asia', lat: 65, lon: 95,
    adjacent: ['ural', 'yakutsk', 'irkutsk', 'mongolia', 'china'] },
  { id: 'yakutsk', name: 'Yakutsk', continent: 'asia', lat: 65, lon: 130,
    adjacent: ['siberia', 'irkutsk', 'kamchatka'] },
  { id: 'irkutsk', name: 'Irkutsk', continent: 'asia', lat: 58, lon: 110,
    adjacent: ['siberia', 'yakutsk', 'kamchatka', 'mongolia'] },
  { id: 'kamchatka', name: 'Kamchatka', continent: 'asia', lat: 55, lon: 160,
    adjacent: ['yakutsk', 'irkutsk', 'mongolia', 'japan', 'alaska'] },
  { id: 'mongolia', name: 'Mongolia', continent: 'asia', lat: 47, lon: 105,
    adjacent: ['siberia', 'irkutsk', 'kamchatka', 'china', 'japan'] },
  { id: 'japan', name: 'Japan', continent: 'asia', lat: 36, lon: 138,
    adjacent: ['kamchatka', 'mongolia'] },
  { id: 'china', name: 'China', continent: 'asia', lat: 35, lon: 105,
    adjacent: ['ural', 'siberia', 'mongolia', 'afghanistan', 'india', 'siam'] },
  { id: 'afghanistan', name: 'Afghanistan', continent: 'asia', lat: 35, lon: 65,
    adjacent: ['ural', 'ukraine', 'china', 'india', 'middle-east'] },
  { id: 'middle-east', name: 'Middle East', continent: 'asia', lat: 30, lon: 50,
    adjacent: ['ukraine', 'southern-europe', 'egypt', 'east-africa', 'afghanistan', 'india'] },
  { id: 'india', name: 'India', continent: 'asia', lat: 22, lon: 78,
    adjacent: ['china', 'afghanistan', 'middle-east', 'siam'] },
  { id: 'siam', name: 'Siam', continent: 'asia', lat: 13, lon: 100,
    adjacent: ['china', 'india', 'indonesia'] },

  // Australia
  { id: 'indonesia', name: 'Indonesia', continent: 'australia', lat: -3, lon: 115,
    adjacent: ['siam', 'new-guinea', 'western-australia'] },
  { id: 'new-guinea', name: 'New Guinea', continent: 'australia', lat: -5, lon: 142,
    adjacent: ['indonesia', 'western-australia', 'eastern-australia'] },
  { id: 'western-australia', name: 'Western Australia', continent: 'australia', lat: -25, lon: 122,
    adjacent: ['indonesia', 'new-guinea', 'eastern-australia'] },
  { id: 'eastern-australia', name: 'Eastern Australia', continent: 'australia', lat: -25, lon: 145,
    adjacent: ['new-guinea', 'western-australia'] },
];

const byId = new Map(TERRITORIES.map((t) => [t.id, t]));

export function territoryById(id: TerritoryID): TerritoryDef | undefined {
  return byId.get(id);
}

export function territoriesByContinent(c: ContinentID): TerritoryDef[] {
  return TERRITORIES.filter((t) => t.continent === c);
}

export function adjacent(a: TerritoryID, b: TerritoryID): boolean {
  const t = byId.get(a);
  return !!t && t.adjacent.includes(b);
}

export function continentById(id: ContinentID): ContinentDef | undefined {
  return CONTINENTS.find((c) => c.id === id);
}

/** Convert (lat, lon) on a unit sphere to a 3D position. Used by the globe. */
export function latLonToVec3(lat: number, lon: number, radius = 1): [number, number, number] {
  const phi = (90 - lat) * (Math.PI / 180);
  const theta = (lon + 180) * (Math.PI / 180);
  const x = -radius * Math.sin(phi) * Math.cos(theta);
  const y = radius * Math.cos(phi);
  const z = radius * Math.sin(phi) * Math.sin(theta);
  return [x, y, z];
}
