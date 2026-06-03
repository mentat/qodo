// Shared types mirroring api/services/risk/types.go. The Firestore docs and
// REST payloads use these shapes.

import type { TerritoryID } from '../components/Risk/board';

export type PlayerID = string; // 'human' | 'ai-1'..'ai-5' | 'neutral'
export type PlayerKind = 'human' | 'ai' | 'neutral';
export type Difficulty = 'easy' | 'normal' | 'hard';
export type Phase = 'place' | 'attack' | 'fortify' | 'awaiting_ai';
export type Status = 'setup' | 'playing' | 'won' | 'lost' | 'surrendered';
export type CardType = 'inf' | 'cav' | 'art' | 'wild';

export interface Card {
  id: string;
  type: CardType;
  territoryId: TerritoryID | '';
}

export interface Player {
  id: PlayerID;
  name: string;
  kind: PlayerKind;
  color: string;
  generalId?: string;
  alive: boolean;
  cards: Card[];
  cardSetsTraded: number;
  eliminated: boolean;
}

export interface TerritoryState {
  ownerId: PlayerID;
  armies: number;
}

export interface AttackResult {
  from: TerritoryID;
  to: TerritoryID;
  attackerDice: number[];
  defenderDice: number[];
  attackerLost: number;
  defenderLost: number;
  conquered: boolean;
}

export interface PostConquest {
  from: TerritoryID;
  to: TerritoryID;
  minArmies: number;
  maxArmies: number;
}

export interface Turn {
  currentPlayerId: PlayerID;
  turnNumber: number;
  phase: Phase;
  armiesToPlace: number;
  conqueredThisTurn: boolean;
  lastAttack?: AttackResult | null;
  postConquestPending?: PostConquest | null;
}

export interface RiskEvent {
  seq: number;
  ts: string | { toDate(): Date }; // Firestore Timestamp on the live socket; ISO from REST
  playerId: PlayerID;
  kind: string;
  payload: Record<string, unknown>;
}

export interface Settings {
  difficulty: Difficulty;
  playerCount: number;
}

export interface GameState {
  gameId: string;
  status: Status;
  createdAt: string;
  startedAt: string;
  endedAt?: string | null;
  settings: Settings;
  players: Player[];
  board: Record<TerritoryID, TerritoryState>;
  turn: Turn;
  events: RiskEvent[];
  deck: Card[];
  setupRemaining: Record<PlayerID, number>;
  lastEventSeq: number;
}

export interface Stats {
  winsByDifficulty: Record<string, number>;
  lossesByDifficulty: Record<string, number>;
  surrendersByDifficulty: Record<string, number>;
  longestGameTurns: number;
  currentWinStreak: number;
  longestWinStreak: number;
  totalGamesStarted: number;
}
