import { useMemo, useState } from 'react';
import {
  Stack, Group, Paper, Title, Text, Button, SegmentedControl,
  SimpleGrid, Badge, Divider, Anchor,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useRiskStore } from '../../store/riskStore';
import { GENERALS, type General } from './cast';
import type { Difficulty, GameState, Stats } from '../../types/risk';

const DIFFICULTIES: { value: Difficulty; label: string; blurb: string }[] = [
  { value: 'easy', label: 'Easy', blurb: 'AI shows attack odds; trades cards eagerly; weaker borders.' },
  { value: 'normal', label: 'Normal', blurb: 'Continent-aware play. Holds borders. No hints.' },
  { value: 'hard', label: 'Hard', blurb: 'Saves cards for the inflection. Hard borders, thin interiors.' },
];

interface Props {
  finishedGame: GameState | null;
}

export function StartScreen({ finishedGame }: Props) {
  const [difficulty, setDifficulty] = useState<Difficulty>('normal');
  const [playerCount, setPlayerCount] = useState<number>(4);
  const startNew = useRiskStore((s) => s.startNew);
  const stats = useRiskStore((s) => s.stats);

  const opponents = useMemo(() => GENERALS.slice(0, playerCount - 1), [playerCount]);

  const onStart = async () => {
    try {
      await startNew(difficulty, playerCount);
      notifications.show({
        title: 'Game on',
        message: `${difficulty.toUpperCase()} · ${playerCount} players · the world awaits`,
        color: 'synthPurple',
      });
    } catch (e) {
      notifications.show({ title: 'Failed to start', message: (e as Error).message, color: 'red' });
    }
  };

  return (
    <Stack p="xl" gap="lg" maw={960} mx="auto">
      <Group justify="space-between" align="flex-end">
        <div>
          <Title order={1} fw={900}>RISK · World Domination</Title>
          <Text c="dimmed" mt={4}>
            42 territories. 6 continents. One winner. Roll the dice or lose them all.
          </Text>
        </div>
        {stats && <StatsCard stats={stats} />}
      </Group>

      {finishedGame && <FinishedGameBanner state={finishedGame} />}

      <Divider />

      <Paper withBorder radius="md" p="lg">
        <Stack gap="lg">
          <div>
            <Text fw={700} mb={6}>Difficulty</Text>
            <SegmentedControl
              fullWidth
              value={difficulty}
              onChange={(v) => setDifficulty(v as Difficulty)}
              data={DIFFICULTIES.map((d) => ({ value: d.value, label: d.label }))}
            />
            <Text size="sm" c="dimmed" mt={6}>
              {DIFFICULTIES.find((d) => d.value === difficulty)?.blurb}
            </Text>
          </div>
          <div>
            <Text fw={700} mb={6}>Players</Text>
            <SegmentedControl
              fullWidth
              value={String(playerCount)}
              onChange={(v) => setPlayerCount(Number(v))}
              data={[2, 3, 4, 5, 6].map((n) => ({ value: String(n), label: `${n} players` }))}
            />
            <Text size="sm" c="dimmed" mt={6}>
              {playerCount === 2
                ? 'Two-player variant: a neutral army defends but never attacks.'
                : `You + ${playerCount - 1} AI generals.`}
            </Text>
          </div>

          <div>
            <Text fw={700} mb={6}>You will face</Text>
            <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }} spacing="sm">
              {opponents.map((g) => <GeneralCard key={g.id} g={g} />)}
            </SimpleGrid>
          </div>

          <Button size="lg" color="synthPurple" onClick={onStart} radius="md">
            Begin world conquest
          </Button>
        </Stack>
      </Paper>

      <Text size="xs" c="dimmed" ta="center">
        Tip: open <Anchor href="/radio" c="neonPink">Radio</Anchor> for synthwave tunes while you play.
      </Text>
    </Stack>
  );
}

function GeneralCard({ g }: { g: General }) {
  return (
    <Paper withBorder radius="md" p="sm" style={{
      borderColor: `var(--mantine-color-${g.color}-6)`,
      background: 'transparent',
    }}>
      <Group gap="sm" wrap="nowrap" align="flex-start">
        <Text size="xl" lh={1}>{g.emoji}</Text>
        <div>
          <Text fw={700} size="sm">{g.name}</Text>
          <Text size="xs" c="dimmed">{g.title}</Text>
          <Text size="xs" mt={4}>{g.blurb}</Text>
        </div>
      </Group>
    </Paper>
  );
}

function StatsCard({ stats }: { stats: Stats }) {
  const totalWins = sumValues(stats.winsByDifficulty);
  const totalLosses = sumValues(stats.lossesByDifficulty);
  return (
    <Paper withBorder radius="md" p="sm" style={{ minWidth: 200 }}>
      <Text size="xs" c="dimmed" fw={700} tt="uppercase">Lifetime</Text>
      <Group gap="xs" mt={4}>
        <Badge color="neonGreen">{totalWins}W</Badge>
        <Badge color="neonPink">{totalLosses}L</Badge>
        {stats.currentWinStreak > 0 && <Badge color="hotYellow">{stats.currentWinStreak} streak</Badge>}
      </Group>
      <Text size="xs" c="dimmed" mt={4}>
        Games started: {stats.totalGamesStarted}
        {stats.longestGameTurns > 0 && ` · longest: ${stats.longestGameTurns} turns`}
      </Text>
    </Paper>
  );
}

function FinishedGameBanner({ state }: { state: GameState }) {
  const color = state.status === 'won' ? 'neonGreen' : state.status === 'surrendered' ? 'hotYellow' : 'neonPink';
  const label = state.status === 'won' ? 'Victory' :
                state.status === 'surrendered' ? 'Surrendered' :
                'Defeat';
  return (
    <Paper withBorder radius="md" p="md" style={{ borderColor: `var(--mantine-color-${color}-6)` }}>
      <Group justify="space-between">
        <div>
          <Badge color={color} size="lg">{label}</Badge>
          <Text mt={6} fw={600}>
            {state.status === 'won'
              ? 'You painted the whole map in your color.'
              : state.status === 'surrendered'
                ? 'You stepped away from the table.'
                : 'The board belongs to someone else now.'}
          </Text>
        </div>
        <Text size="sm" c="dimmed">
          {state.settings.difficulty.toUpperCase()} · {state.settings.playerCount} players · {state.turn.turnNumber} turns
        </Text>
      </Group>
    </Paper>
  );
}

function sumValues(r: Record<string, number>): number {
  return Object.values(r).reduce((a, b) => a + b, 0);
}
