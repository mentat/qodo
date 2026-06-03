import { useEffect } from 'react';
import { Box, Center, Loader } from '@mantine/core';
import { useAuth } from '../../hooks/useAuth';
import { useRiskStore } from '../../store/riskStore';
import { StartScreen } from './StartScreen';
import { GameScreen } from './GameScreen';

// RiskApp is the root for the /risk route. It owns the Firestore subscription
// for the user's active game + lifetime stats — when there's no active game,
// it shows the start screen; otherwise it shows the game board.
export function RiskApp() {
  const { user } = useAuth();
  const subscribe = useRiskStore((s) => s.subscribe);
  const unsubscribe = useRiskStore((s) => s.unsubscribe);
  const game = useRiskStore((s) => s.game);
  const loading = useRiskStore((s) => s.loading);
  const refresh = useRiskStore((s) => s.refresh);

  useEffect(() => {
    if (!user) return;
    subscribe(user.uid);
    // Kick a one-shot REST fetch too, so the initial "no game" state shows
    // immediately if the user has never started one (onSnapshot doesn't fire
    // for a non-existent doc).
    void refresh();
    return () => {
      unsubscribe();
    };
  }, [user, subscribe, unsubscribe, refresh]);

  if (loading) {
    return (
      <Center mih="60vh">
        <Loader />
      </Center>
    );
  }

  return (
    <Box style={{ height: 'calc(100vh - 92px)' }}>
      {game && game.status !== 'won' && game.status !== 'lost' && game.status !== 'surrendered'
        ? <GameScreen />
        : <StartScreen finishedGame={game} />}
    </Box>
  );
}
