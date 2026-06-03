import { create } from 'zustand';
import { doc, onSnapshot } from 'firebase/firestore';
import { db } from '../firebase';
import * as api from '../api/risk';
import type { GameState, Stats, Difficulty, AttackResult } from '../types/risk';
import type { TerritoryID } from '../components/Risk/board';

// The Risk store mirrors mailStore's onSnapshot pattern. Writes go through
// the Go API; live updates arrive via the riskGames/{uid} Firestore doc.
// The animation queue (see queueEventsFrom + drainQueue) lets AI sub-step
// writes from the backend land one at a time visually rather than as a
// single batch.

interface RiskState {
  game: GameState | null;
  stats: Stats | null;
  loading: boolean;
  // Last seq we've fully rendered to the UI. New events with seq > this are
  // queued for animation by GameScreen.
  renderedSeq: number;
  unsubGame: (() => void) | null;
  unsubStats: (() => void) | null;

  // selection state — interaction with the globe
  selectedFrom: TerritoryID | null;
  hoveredTerritory: TerritoryID | null;

  // last attack rounds returned by the API (for the dice-roll modal)
  lastRounds: AttackResult[];

  subscribe: (uid: string) => void;
  unsubscribe: () => void;
  refresh: () => Promise<void>;

  startNew: (difficulty: Difficulty, playerCount: number) => Promise<void>;
  placeInitial: (t: TerritoryID) => Promise<void>;
  place: (t: TerritoryID, n: number) => Promise<void>;
  trade: (cardIds: string[]) => Promise<void>;
  attack: (from: TerritoryID, to: TerritoryID, mode?: 'single' | 'blitz') => Promise<AttackResult[]>;
  resolvePostConquest: (count: number) => Promise<void>;
  fortify: (from: TerritoryID, to: TerritoryID, count: number) => Promise<void>;
  endPhase: () => Promise<void>;
  skipFortify: () => Promise<void>;
  surrender: () => Promise<void>;

  selectFrom: (t: TerritoryID | null) => void;
  hover: (t: TerritoryID | null) => void;
  markRendered: (seq: number) => void;
}

function normalizeGame(raw: unknown): GameState | null {
  if (!raw || typeof raw !== 'object') return null;
  return raw as GameState;
}

export const useRiskStore = create<RiskState>((set, get) => ({
  game: null,
  stats: null,
  loading: true,
  renderedSeq: 0,
  unsubGame: null,
  unsubStats: null,
  selectedFrom: null,
  hoveredTerritory: null,
  lastRounds: [],

  subscribe: (uid) => {
    if (get().unsubGame) return;
    const gameRef = doc(db, 'riskGames', uid);
    const unsubGame = onSnapshot(
      gameRef,
      (snap) => {
        if (!snap.exists()) {
          set({ game: null, loading: false });
          return;
        }
        const g = normalizeGame(snap.data());
        set({ game: g, loading: false });
      },
      (err) => {
        console.error('risk game snapshot error', err);
        set({ loading: false });
      },
    );
    const statsRef = doc(db, 'riskStats', uid);
    const unsubStats = onSnapshot(
      statsRef,
      (snap) => {
        if (!snap.exists()) {
          set({ stats: {
            winsByDifficulty: {},
            lossesByDifficulty: {},
            surrendersByDifficulty: {},
            longestGameTurns: 0,
            currentWinStreak: 0,
            longestWinStreak: 0,
            totalGamesStarted: 0,
          }});
          return;
        }
        set({ stats: snap.data() as Stats });
      },
      (err) => console.error('risk stats snapshot error', err),
    );
    set({ unsubGame, unsubStats });
  },

  unsubscribe: () => {
    const { unsubGame, unsubStats } = get();
    if (unsubGame) unsubGame();
    if (unsubStats) unsubStats();
    set({
      unsubGame: null, unsubStats: null,
      game: null, stats: null,
      selectedFrom: null, hoveredTerritory: null,
      renderedSeq: 0, lastRounds: [],
    });
  },

  refresh: async () => {
    try {
      const g = await api.fetchGame();
      set({ game: g, loading: false });
    } catch (e) {
      console.error('risk refresh', e);
    }
  },

  startNew: async (difficulty, playerCount) => {
    const g = await api.newGame(difficulty, playerCount);
    set({ game: g, selectedFrom: null, renderedSeq: 0, lastRounds: [] });
  },

  placeInitial: async (t) => {
    const g = await api.placeInitial(t);
    set({ game: g });
  },

  place: async (t, n) => {
    const g = await api.placeArmies(t, n);
    set({ game: g });
  },

  trade: async (cardIds) => {
    const g = await api.tradeCards(cardIds);
    set({ game: g });
  },

  attack: async (from, to, mode = 'blitz') => {
    const { state, rounds } = await api.attack(from, to, mode);
    set({ game: state, lastRounds: rounds });
    return rounds;
  },

  resolvePostConquest: async (count) => {
    const g = await api.resolvePostConquest(count);
    set({ game: g, selectedFrom: null });
  },

  fortify: async (from, to, count) => {
    const g = await api.fortify(from, to, count);
    set({ game: g, selectedFrom: null });
  },

  endPhase: async () => {
    const g = await api.endPhase();
    set({ game: g, selectedFrom: null });
  },

  skipFortify: async () => {
    const g = await api.skipFortify();
    set({ game: g, selectedFrom: null });
  },

  surrender: async () => {
    const g = await api.surrender();
    set({ game: g });
  },

  selectFrom: (t) => set({ selectedFrom: t }),
  hover: (t) => set({ hoveredTerritory: t }),
  markRendered: (seq) => set({ renderedSeq: seq }),
}));

// ── Selectors / derived ───────────────────────────────────────────────────

export function humanPlayer(game: GameState | null) {
  if (!game) return null;
  return game.players.find((p) => p.kind === 'human') ?? null;
}

export function currentPlayer(game: GameState | null) {
  if (!game) return null;
  return game.players.find((p) => p.id === game.turn.currentPlayerId) ?? null;
}

export function ownedCount(game: GameState | null, playerId: string): number {
  if (!game) return 0;
  return Object.values(game.board).filter((t) => t.ownerId === playerId).length;
}

export function totalArmies(game: GameState | null, playerId: string): number {
  if (!game) return 0;
  return Object.values(game.board)
    .filter((t) => t.ownerId === playerId)
    .reduce((sum, t) => sum + t.armies, 0);
}

/**
 * Conquest probability lookup mirroring the Go AI's heuristic, used by the
 * frontend AttackModal to show "X%" when difficulty=easy.
 */
export function conquestProbability(att: number, def: number): number {
  if (att <= 1) return 0;
  if (def <= 0) return 1;
  const ratio = (att - 1) / def;
  const p = 1 / (1 + 1.5 * Math.pow(0.55, ratio * 1.6));
  return Math.max(0, Math.min(1, p));
}
