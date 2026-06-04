import type { GameState, PlayerID, RiskEvent } from '../../types/risk';
import type { TerritoryID } from './board';
import type { RiskAvatarMood } from './RiskAvatar';

export function avatarEventForPlayer(game: GameState, playerId: PlayerID): { mood: RiskAvatarMood; key: number } {
  const events = game.events.slice(-8).reverse();
  for (const ev of events) {
    const mood = moodFromEvent(game, ev, playerId);
    if (mood !== 'none') return { mood, key: ev.seq };
  }
  return { mood: 'none', key: 0 };
}

function moodFromEvent(game: GameState, ev: RiskEvent, playerId: PlayerID): RiskAvatarMood {
  if (ev.kind === 'conquer' && ev.playerId === playerId) return 'nod';
  if (ev.kind === 'win' && ev.playerId === playerId) return 'nod';
  if (ev.kind === 'eliminate' && ev.playerId === playerId) return 'nod';

  if (ev.kind === 'attack') {
    if (ev.playerId === playerId) {
      const attackerLost = numericPayload(ev, 'attackerLost');
      const defenderLost = numericPayload(ev, 'defenderLost');
      return attackerLost > defenderLost ? 'shake' : 'nod';
    }
    const to = ev.payload.to;
    if (typeof to === 'string') {
      const defender = game.board[to as TerritoryID];
      if (defender?.ownerId === playerId) return 'startled';
    }
  }

  return 'none';
}

function numericPayload(ev: RiskEvent, key: string): number {
  const value = ev.payload[key];
  return typeof value === 'number' ? value : 0;
}
