import { useMemo, useState } from 'react';
import { Paper, Group, Text, Button, Stack, Badge, ActionIcon, Tooltip } from '@mantine/core';
import { IconX } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { useRiskStore, humanPlayer } from '../../store/riskStore';
import { territoryById } from './board';
import type { Card } from '../../types/risk';

const TYPE_LABEL: Record<Card['type'], string> = {
  inf: 'Infantry',
  cav: 'Cavalry',
  art: 'Artillery',
  wild: 'Wild',
};

const TYPE_COLOR: Record<Card['type'], string> = {
  inf: 'electricBlue',
  cav: 'neonGreen',
  art: 'hotYellow',
  wild: 'synthPurple',
};

const TYPE_EMOJI: Record<Card['type'], string> = {
  inf: '🪖',
  cav: '🐎',
  art: '💣',
  wild: '★',
};

// CardFan: a persistent fan in the bottom-right corner showing the human's
// hand. Clicking expands into a tray where the player can pick 3 cards and
// trade them for the next escalating bonus. Only visible during the Place
// phase of the human's turn.
export function CardFan() {
  const game = useRiskStore((s) => s.game);
  const trade = useRiskStore((s) => s.trade);
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<string[]>([]);

  const human = humanPlayer(game);
  const cards = useMemo(() => human?.cards ?? [], [human?.cards]);
  const isHumanTurn = game && human && game.turn.currentPlayerId === human.id;
  const canTrade = isHumanTurn && game?.turn.phase === 'place' && game?.status === 'playing';

  const togglePick = (id: string) => {
    setPicked((cur) => {
      if (cur.includes(id)) return cur.filter((x) => x !== id);
      if (cur.length >= 3) return cur;
      return [...cur, id];
    });
  };

  const valid = useMemo(() => {
    if (picked.length !== 3) return false;
    return validSet(cards.filter((c) => picked.includes(c.id)));
  }, [picked, cards]);

  const submit = async () => {
    try {
      await trade(picked);
      setPicked([]);
      setOpen(false);
      notifications.show({ title: 'Trade complete', message: 'Bonus armies added.', color: 'neonGreen' });
    } catch (e) {
      notifications.show({ title: 'Trade failed', message: (e as Error).message, color: 'red' });
    }
  };

  if (!game || !human) return null;

  return (
    <Paper
      withBorder
      radius="md"
      shadow="lg"
      p="sm"
      style={{
        position: 'absolute',
        bottom: 12,
        right: 12,
        zIndex: 5,
        minWidth: open ? 320 : 'auto',
        backdropFilter: 'blur(4px)',
        background: 'color-mix(in srgb, var(--mantine-color-body) 92%, transparent)',
      }}
    >
      <Group justify="space-between" align="center">
        <Group gap={4}>
          <Text size="xs" c="dimmed" fw={700} tt="uppercase">Your hand</Text>
          <Badge size="sm" color={cards.length >= 5 ? 'neonPink' : 'synthPurple'}>
            {cards.length} / 5
          </Badge>
        </Group>
        <Group gap={4}>
          {!open && cards.length > 0 && (
            <Button size="xs" variant="subtle" onClick={() => setOpen(true)}>
              Show
            </Button>
          )}
          {open && (
            <ActionIcon size="sm" variant="subtle" onClick={() => { setOpen(false); setPicked([]); }}>
              <IconX size={14} />
            </ActionIcon>
          )}
        </Group>
      </Group>

      {!open && cards.length > 0 && (
        <Group gap={2} mt={4}>
          {cards.map((c) => (
            <Tooltip key={c.id} label={cardLabel(c)} position="top">
              <div style={{
                width: 28, height: 38,
                borderRadius: 4,
                background: `var(--mantine-color-${TYPE_COLOR[c.type]}-1)`,
                border: `2px solid var(--mantine-color-${TYPE_COLOR[c.type]}-5)`,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 18,
              }}>
                {TYPE_EMOJI[c.type]}
              </div>
            </Tooltip>
          ))}
        </Group>
      )}

      {open && (
        <Stack gap="xs" mt="xs">
          <Text size="xs" c="dimmed">
            Pick 3 cards. Valid sets: three of a kind, one of each, or any 2 + a wild.
          </Text>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
            {cards.map((c) => {
              const isPicked = picked.includes(c.id);
              return (
                <button
                  key={c.id}
                  onClick={() => togglePick(c.id)}
                  style={{
                    width: 64, height: 86,
                    borderRadius: 6,
                    background: `var(--mantine-color-${TYPE_COLOR[c.type]}-light)`,
                    border: isPicked
                      ? '3px solid var(--mantine-color-synthPurple-6)'
                      : `2px solid var(--mantine-color-${TYPE_COLOR[c.type]}-5)`,
                    cursor: 'pointer',
                    padding: 4,
                    color: 'var(--mantine-color-text)',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                  type="button"
                >
                  <div style={{ fontSize: 22 }}>{TYPE_EMOJI[c.type]}</div>
                  <div style={{ fontSize: 10, fontWeight: 700 }}>{TYPE_LABEL[c.type]}</div>
                  <div style={{ fontSize: 9, opacity: 0.7, textAlign: 'center', lineHeight: 1.1 }}>
                    {c.territoryId ? territoryById(c.territoryId)?.name ?? '' : '—'}
                  </div>
                </button>
              );
            })}
          </div>
          <Group justify="space-between">
            <Text size="xs" c={valid ? 'neonGreen.6' : 'dimmed'}>
              {picked.length === 3
                ? (valid ? 'Valid set ✓' : 'Not a valid set')
                : `${picked.length}/3 selected`}
            </Text>
            <Button
              size="xs"
              disabled={!canTrade || !valid}
              onClick={() => void submit()}
              color="synthPurple"
            >
              Trade for bonus
            </Button>
          </Group>
          {!canTrade && (
            <Text size="xs" c="dimmed">Trade during your reinforce phase.</Text>
          )}
        </Stack>
      )}

      {!open && cards.length === 0 && (
        <Text size="xs" c="dimmed" mt={4}>No cards yet — conquer a territory to earn one.</Text>
      )}
    </Paper>
  );
}

function cardLabel(c: Card): string {
  if (c.type === 'wild') return 'Wild';
  return `${TYPE_LABEL[c.type]} · ${c.territoryId ? territoryById(c.territoryId)?.name ?? c.territoryId : ''}`;
}

function validSet(cards: Card[]): boolean {
  if (cards.length !== 3) return false;
  let wilds = 0;
  const counts: Record<string, number> = {};
  for (const c of cards) {
    if (c.type === 'wild') wilds++;
    else counts[c.type] = (counts[c.type] ?? 0) + 1;
  }
  if (wilds >= 1) return true;
  const keys = Object.keys(counts);
  if (keys.length === 1) return true;
  if (keys.length === 3) return true;
  return false;
}
