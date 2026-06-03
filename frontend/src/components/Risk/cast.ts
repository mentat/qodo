// Client-side display info for the AI generals. The backend (api/services/risk/cast.go)
// is the source of truth for strategy weights; this file only mirrors the
// fields needed to render avatars + blurbs in the lobby and HUD.

export interface General {
  id: string;
  name: string;
  title: string;
  blurb: string;
  color: string; // Mantine palette key (display tint, not the in-game player color)
  emoji: string; // quick-render avatar before any SVGs ship
}

export const GENERALS: General[] = [
  {
    id: 'maxine-voltage',
    name: 'Maxine Voltage',
    title: 'Field Marshal',
    blurb: 'A neon-soaked cavalry commander who treats the world like a synth solo — loud, fast, impossible to ignore.',
    color: 'neonPink',
    emoji: '⚡',
  },
  {
    id: 'general-static',
    name: 'General Static',
    title: 'Commander',
    blurb: 'A patient tactician whose silences are louder than her artillery. She finishes continents others abandon.',
    color: 'electricBlue',
    emoji: '📺',
  },
  {
    id: 'vice-admiral-vector',
    name: 'Vice-Admiral Vector',
    title: 'Vice-Admiral',
    blurb: 'Naval strategist with a vendetta against any continent that touches the sea. Especially Australia.',
    color: 'neonGreen',
    emoji: '⚓',
  },
  {
    id: 'commodore-cassette',
    name: 'Commodore Cassette',
    title: 'Commodore',
    blurb: 'Rewinds, replays, and refuses to flip sides. Predictable but relentless.',
    color: 'hotYellow',
    emoji: '📼',
  },
  {
    id: 'captain-coral',
    name: 'Captain Coral',
    title: 'Captain',
    blurb: 'Coastal raider who hops across oceans. Reads the wrap-around adjacencies like nobody else.',
    color: 'synthPurple',
    emoji: '🐚',
  },
  {
    id: 'field-marshal-neon',
    name: 'Field Marshal Neon',
    title: 'Field Marshal',
    blurb: 'Old-school continental conquest doctrine in a chrome-pink uniform.',
    color: 'neonPink',
    emoji: '🎖',
  },
  {
    id: 'colonel-chrome',
    name: 'Colonel Chrome',
    title: 'Colonel',
    blurb: 'Polished to a mirror. Plays a textbook board, opens with Asia, dares you to stop him.',
    color: 'electricBlue',
    emoji: '🔱',
  },
  {
    id: 'lieutenant-laser',
    name: 'Lt. Laser',
    title: 'Lieutenant',
    blurb: 'Junior officer with a bright pink targeting grid. Over-commits, but lands the first kill.',
    color: 'hotYellow',
    emoji: '🎯',
  },
];

export function generalById(id: string | undefined | null): General | undefined {
  if (!id) return undefined;
  return GENERALS.find((g) => g.id === id);
}
