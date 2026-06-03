import { useState, useEffect } from 'react';
import { Flex, Stack, Box } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useRiskStore, humanPlayer } from '../../store/riskStore';
import { Globe } from './Globe';
import { TurnPanel, PlayersPanel, EventLog } from './HUD';
import { CardFan } from './CardFan';
import {
  AttackModal, PostConquestModal, FortifyModal, SurrenderConfirm, GameOverBanner,
} from './Modals';
import { adjacent, type TerritoryID } from './board';

// GameScreen: the playing layout. Left HUD, center globe + modals, right HUD.
// Handles the territory-click → action routing based on the current phase.
export function GameScreen() {
  const game = useRiskStore((s) => s.game);
  const selectedFrom = useRiskStore((s) => s.selectedFrom);
  const selectFrom = useRiskStore((s) => s.selectFrom);
  const place = useRiskStore((s) => s.place);
  const placeInitial = useRiskStore((s) => s.placeInitial);
  const surrender = useRiskStore((s) => s.surrender);
  const startNew = useRiskStore((s) => s.startNew);

  const [attackTarget, setAttackTarget] = useState<TerritoryID | null>(null);
  const [fortifyTarget, setFortifyTarget] = useState<TerritoryID | null>(null);
  const [surrenderOpen, setSurrenderOpen] = useState(false);

  const human = humanPlayer(game);
  const isHumanTurn = game && human && game.turn.currentPlayerId === human.id;

  // Clear selection if the phase changes or the turn flips.
  useEffect(() => {
    selectFrom(null);
    setAttackTarget(null);
    setFortifyTarget(null);
  }, [game?.turn.phase, game?.turn.currentPlayerId, selectFrom]);

  if (!game || !human) return null;

  const onTerritoryClick = async (t: TerritoryID) => {
    if (!isHumanTurn || game.status !== 'playing' && game.status !== 'setup') return;
    const ts = game.board[t];
    if (!ts) return;

    // Setup phase: place 1 army on any owned territory.
    if (game.status === 'setup') {
      if (ts.ownerId !== human.id) {
        notifications.show({ message: 'Place only on your territories during setup.', color: 'neonPink' });
        return;
      }
      try {
        await placeInitial(t);
      } catch (e) {
        notifications.show({ message: (e as Error).message, color: 'red' });
      }
      return;
    }

    // Place phase: must own + add 1 army.
    if (game.turn.phase === 'place') {
      if (ts.ownerId !== human.id) {
        notifications.show({ message: 'Place only on your territories.', color: 'neonPink' });
        return;
      }
      if (game.turn.armiesToPlace <= 0) {
        notifications.show({ message: 'No armies left to place — end the phase.' });
        return;
      }
      try {
        await place(t, 1);
      } catch (e) {
        notifications.show({ message: (e as Error).message, color: 'red' });
      }
      return;
    }

    // Attack phase: pick attacker, then defender.
    if (game.turn.phase === 'attack') {
      if (!selectedFrom) {
        if (ts.ownerId !== human.id) {
          notifications.show({ message: 'Pick one of your territories to attack from.' });
          return;
        }
        if (ts.armies < 2) {
          notifications.show({ message: 'Need ≥ 2 armies to launch an attack.' });
          return;
        }
        selectFrom(t);
        return;
      }
      // Already have a selectedFrom.
      if (t === selectedFrom) {
        selectFrom(null);
        return;
      }
      if (ts.ownerId === human.id) {
        // Re-pick attacker.
        if (ts.armies < 2) {
          notifications.show({ message: 'Need ≥ 2 armies to launch an attack.' });
          return;
        }
        selectFrom(t);
        return;
      }
      if (!adjacent(selectedFrom, t)) {
        notifications.show({ message: 'Not adjacent to your selected territory.', color: 'neonPink' });
        return;
      }
      setAttackTarget(t);
      return;
    }

    // Fortify phase: pick source then destination (both friendly + adjacent).
    if (game.turn.phase === 'fortify') {
      if (!selectedFrom) {
        if (ts.ownerId !== human.id || ts.armies < 2) {
          notifications.show({ message: 'Pick a source with ≥ 2 armies.' });
          return;
        }
        selectFrom(t);
        return;
      }
      if (t === selectedFrom) {
        selectFrom(null);
        return;
      }
      if (ts.ownerId !== human.id) {
        notifications.show({ message: 'Destination must be your territory.', color: 'neonPink' });
        return;
      }
      if (!adjacent(selectedFrom, t)) {
        notifications.show({ message: 'Destination must be adjacent (classic rule).', color: 'neonPink' });
        return;
      }
      setFortifyTarget(t);
      return;
    }
  };

  const onSurrender = async () => {
    setSurrenderOpen(false);
    try {
      await surrender();
    } catch (e) {
      notifications.show({ message: (e as Error).message, color: 'red' });
    }
  };

  const onRestart = async () => {
    try {
      await startNew(game.settings.difficulty, game.settings.playerCount);
    } catch (e) {
      notifications.show({ message: (e as Error).message, color: 'red' });
    }
  };

  return (
    <Flex gap="md" h="100%" px="md" pb="md">
      {/* Left HUD */}
      <Stack gap="md" w={260} miw={260}>
        <TurnPanel onSurrender={() => setSurrenderOpen(true)} onRestart={() => void onRestart()} />
      </Stack>

      {/* Center: the globe + floating elements */}
      <Box style={{ flex: 1, position: 'relative', minWidth: 0 }}>
        <Globe onTerritoryClick={(t) => void onTerritoryClick(t)} />
        <CardFan />
      </Box>

      {/* Right HUD */}
      <Stack gap="md" w={300} miw={300}>
        <PlayersPanel />
        <EventLog />
      </Stack>

      {/* Modals */}
      {attackTarget && selectedFrom && (
        <AttackModal
          from={selectedFrom}
          to={attackTarget}
          onClose={() => setAttackTarget(null)}
        />
      )}
      {fortifyTarget && selectedFrom && (
        <FortifyModal
          from={selectedFrom}
          to={fortifyTarget}
          onClose={() => setFortifyTarget(null)}
        />
      )}
      {game.turn.postConquestPending && <PostConquestModal />}
      <SurrenderConfirm
        opened={surrenderOpen}
        onConfirm={() => void onSurrender()}
        onClose={() => setSurrenderOpen(false)}
      />
      {(game.status === 'won' || game.status === 'lost' || game.status === 'surrendered') && (
        <GameOverBanner game={game} onAck={() => { /* RiskApp will route to StartScreen */ }} />
      )}
    </Flex>
  );
}
