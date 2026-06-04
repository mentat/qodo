// Client-side display info for the AI generals. The backend (api/services/risk/cast.go)
// is the source of truth for strategy weights; this file only mirrors the
// fields needed to render avatars + blurbs in the lobby and HUD.

export interface General {
  id: string;
  name: string;
  title: string;
  blurb: string;
  color: string; // Mantine palette key (display tint, not the in-game player color)
  emoji: string; // fallback avatar when dotLottie cannot render
  avatarSrc: string;
  avatarAlt: string;
  artDirection: string[];
}

export const GENERALS: General[] = [
  {
    id: 'maxine-voltage',
    name: 'Maxine Voltage',
    title: 'Field Marshal',
    blurb: 'A neon-soaked cavalry commander who treats the world like a synth solo — loud, fast, impossible to ignore.',
    color: 'neonPink',
    emoji: '⚡',
    avatarSrc: '/risk-avatars/maxine-voltage.lottie',
    avatarAlt: 'Animated neon-pink avatar for Maxine Voltage',
    artDirection: ['neon cavalry', 'lightning insignia', 'hot pink command glow'],
  },
  {
    id: 'general-static',
    name: 'General Static',
    title: 'Commander',
    blurb: 'A patient tactician whose silences are louder than her artillery. She finishes continents others abandon.',
    color: 'electricBlue',
    emoji: '📺',
    avatarSrc: '/risk-avatars/general-static.lottie',
    avatarAlt: 'Animated electric-blue avatar for General Static',
    artDirection: ['crt visor', 'blue scanlines', 'quiet tactician'],
  },
  {
    id: 'vice-admiral-vector',
    name: 'Vice-Admiral Vector',
    title: 'Vice-Admiral',
    blurb: 'Naval strategist with a vendetta against any continent that touches the sea. Especially Australia.',
    color: 'neonGreen',
    emoji: '⚓',
    avatarSrc: '/risk-avatars/vice-admiral-vector.lottie',
    avatarAlt: 'Animated green admiral avatar for Vice-Admiral Vector',
    artDirection: ['naval cap', 'anchor insignia', 'green vector glow'],
  },
  {
    id: 'commodore-cassette',
    name: 'Commodore Cassette',
    title: 'Commodore',
    blurb: 'Rewinds, replays, and refuses to flip sides. Predictable but relentless.',
    color: 'hotYellow',
    emoji: '📼',
    avatarSrc: '/risk-avatars/commodore-cassette.lottie',
    avatarAlt: 'Animated yellow avatar for Commodore Cassette',
    artDirection: ['cassette badge', 'tape-loop energy', 'yellow deck lights'],
  },
  {
    id: 'captain-coral',
    name: 'Captain Coral',
    title: 'Captain',
    blurb: 'Coastal raider who hops across oceans. Reads the wrap-around adjacencies like nobody else.',
    color: 'synthPurple',
    emoji: '🐚',
    avatarSrc: '/risk-avatars/captain-coral.lottie',
    avatarAlt: 'Animated purple coastal avatar for Captain Coral',
    artDirection: ['shell insignia', 'coral raider', 'violet sea glow'],
  },
  {
    id: 'field-marshal-neon',
    name: 'Field Marshal Neon',
    title: 'Field Marshal',
    blurb: 'Old-school continental conquest doctrine in a chrome-pink uniform.',
    color: 'neonPink',
    emoji: '🎖',
    avatarSrc: '/risk-avatars/field-marshal-neon.lottie',
    avatarAlt: 'Animated chrome-pink avatar for Field Marshal Neon',
    artDirection: ['field marshal cap', 'medal insignia', 'old-school neon doctrine'],
  },
  {
    id: 'colonel-chrome',
    name: 'Colonel Chrome',
    title: 'Colonel',
    blurb: 'Polished to a mirror. Plays a textbook board, opens with Asia, dares you to stop him.',
    color: 'electricBlue',
    emoji: '🔱',
    avatarSrc: '/risk-avatars/colonel-chrome.lottie',
    avatarAlt: 'Animated chrome-blue avatar for Colonel Chrome',
    artDirection: ['chrome visor', 'trident insignia', 'mirror-polished uniform'],
  },
  {
    id: 'lieutenant-laser',
    name: 'Lt. Laser',
    title: 'Lieutenant',
    blurb: 'Junior officer with a bright pink targeting grid. Over-commits, but lands the first kill.',
    color: 'hotYellow',
    emoji: '🎯',
    avatarSrc: '/risk-avatars/lieutenant-laser.lottie',
    avatarAlt: 'Animated yellow target-grid avatar for Lieutenant Laser',
    artDirection: ['targeting grid', 'junior laser officer', 'hot yellow lock-on'],
  },
];

export function generalById(id: string | undefined | null): General | undefined {
  if (!id) return undefined;
  return GENERALS.find((g) => g.id === id);
}
