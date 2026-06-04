import { describe, expect, it } from 'bun:test';
import { normalizeGame, normalizeStats } from '../src/store/riskStore';

describe('risk store normalization', () => {
  it('coerces nullable Firestore arrays and maps to render-safe defaults', () => {
    const game = normalizeGame({
      gameId: 'g1',
      status: 'playing',
      createdAt: '',
      startedAt: '',
      settings: { difficulty: 'normal', playerCount: 2 },
      players: [
        { id: 'human', name: 'You', kind: 'human', color: 'neonPink', alive: true, cards: null, cardSetsTraded: 0, eliminated: false },
        { id: 'ai-0', name: 'AI', kind: 'ai', color: 'electricBlue', alive: false, cards: null, cardSetsTraded: 0, eliminated: true },
      ],
      board: null,
      turn: { currentPlayerId: 'human', turnNumber: 1, phase: 'place', armiesToPlace: 3, conqueredThisTurn: false },
      events: null,
      deck: null,
      setupRemaining: null,
      lastEventSeq: 0,
    });

    expect(game?.players.map((p) => p.cards.length)).toEqual([0, 0]);
    expect(Object.keys(game?.board ?? {})).toHaveLength(0);
    expect(game?.events).toHaveLength(0);
    expect(game?.deck).toHaveLength(0);
    expect(game?.setupRemaining).toEqual({});
    expect(game?.turn.lastAttack).toBeNull();
    expect(game?.turn.postConquestPending).toBeNull();
  });

  it('coerces nullable stats maps to empty records', () => {
    const stats = normalizeStats({
      winsByDifficulty: null,
      lossesByDifficulty: null,
      surrendersByDifficulty: null,
    });

    expect(stats.winsByDifficulty).toEqual({});
    expect(stats.lossesByDifficulty).toEqual({});
    expect(stats.surrendersByDifficulty).toEqual({});
    expect(stats.totalGamesStarted).toBe(0);
  });
});
