import { useEffect, useRef } from 'react';
import { Stack, Group, Paper, Text, Title, Badge, Button, ScrollArea, Divider, Tooltip } from '@mantine/core';
import {
  IconChevronRight, IconFlag, IconRotateClockwise, IconCards, IconSwords, IconHammer,
  IconMessageCircle,
} from '@tabler/icons-react';
import { useRiskStore, currentPlayer, humanPlayer, ownedCount, totalArmies } from '../../store/riskStore';
import { useUIStore } from '../../store/uiStore';
import { generalById } from './cast';
import { territoryById, CONTINENTS } from './board';
import type { GameState, RiskEvent } from '../../types/risk';

interface Props {
  onSurrender: () => void;
  onRestart: () => void;
}

// TurnPanel: phase, current-player, armies-left, end-phase button.
export function TurnPanel({ onSurrender, onRestart }: Props) {
  const game = useRiskStore((s) => s.game);
  const endPhase = useRiskStore((s) => s.endPhase);
  const skipFortify = useRiskStore((s) => s.skipFortify);
  const openChatWithPrefill = useUIStore((s) => s.openChatWithPrefill);

  if (!game) return null;
  const cur = currentPlayer(game);
  const isHumanTurn = cur?.kind === 'human';
  const gen = generalById(cur?.generalId);
  const phaseInfo = phaseLabel(game);

  return (
    <Paper withBorder radius="md" p="md" h="100%">
      <Stack gap="md" h="100%">
        <div>
          <Group justify="space-between" align="center">
            <Text size="xs" c="dimmed" tt="uppercase" fw={700}>Turn {game.turn.turnNumber}</Text>
            <Group gap={4}>
              <Tooltip label="Ask Marvin about the board" position="bottom">
                <Button variant="subtle" size="xs" color="synthPurple"
                  onClick={() => openChatWithPrefill(buildMarvinPrompt(game))}>
                  <IconMessageCircle size={14} />
                </Button>
              </Tooltip>
              <Tooltip label="Restart game" position="bottom">
                <Button variant="subtle" size="xs" color="hotYellow" onClick={onRestart}>
                  <IconRotateClockwise size={14} />
                </Button>
              </Tooltip>
              <Tooltip label="Surrender" position="bottom">
                <Button variant="subtle" size="xs" color="neonPink" onClick={onSurrender}>
                  <IconFlag size={14} />
                </Button>
              </Tooltip>
            </Group>
          </Group>
          <Group gap="xs" mt={4}>
            <ColorChip color={cur?.color ?? 'gray'} />
            <Title order={4} c={cur?.kind === 'human' ? undefined : `${cur?.color}.5`}>
              {cur?.name ?? '...'}
            </Title>
          </Group>
          {gen && (
            <Text size="xs" c="dimmed" mt={2}>{gen.emoji} {gen.title}</Text>
          )}
        </div>

        <div>
          <Group gap="xs">
            {phaseInfo.icon}
            <Text fw={700}>{phaseInfo.label}</Text>
          </Group>
          <Text size="sm" c="dimmed">{phaseInfo.hint}</Text>
          {game.turn.phase === 'place' && game.status === 'playing' && (
            <Badge mt={6} color="synthPurple" size="lg">
              {game.turn.armiesToPlace} armies to place
            </Badge>
          )}
        </div>

        {isHumanTurn && game.status === 'playing' && (
          <Group gap="xs">
            <Button
              fullWidth
              rightSection={<IconChevronRight size={16} />}
              onClick={() => void endPhase()}
              disabled={
                (game.turn.phase === 'place' && game.turn.armiesToPlace > 0) ||
                game.turn.postConquestPending !== null
              }
            >
              End {game.turn.phase}
            </Button>
            {game.turn.phase === 'fortify' && (
              <Button variant="subtle" onClick={() => void skipFortify()}>Skip</Button>
            )}
          </Group>
        )}

        {!isHumanTurn && (
          <Paper p="xs" radius="sm" bg="var(--mantine-color-synthPurple-light)">
            <Text size="sm" c="dimmed">
              ⏳ {cur?.name ?? 'AI'} is thinking…
            </Text>
          </Paper>
        )}

        <Divider />

        <ContinentLegend game={game} />
      </Stack>
    </Paper>
  );
}

// buildMarvinPrompt assembles a short, ready-to-send question for Marvin that
// embeds the current Risk board snapshot. It's pre-filled in the chat so the
// user can edit before sending. Marvin can ride on this for commentary.
function buildMarvinPrompt(game: GameState): string {
  const human = humanPlayer(game);
  const lines: string[] = [
    `(Risk-game context — feel free to comment, no tools needed.)`,
    `Turn ${game.turn.turnNumber} · phase: ${game.turn.phase} · difficulty: ${game.settings.difficulty}.`,
  ];
  for (const p of game.players) {
    const terrs = ownedCount(game, p.id);
    const armies = totalArmies(game, p.id);
    lines.push(
      `- ${p.kind === 'human' ? 'You' : p.name}` +
      `${p.eliminated ? ' (eliminated)' : ''}: ` +
      `${terrs} territories, ${armies} armies, ${p.cards.length} cards.`,
    );
  }
  if (human) {
    lines.push(`What should I do next?`);
  } else {
    lines.push(`How's the board looking?`);
  }
  return lines.join('\n');
}

function phaseLabel(game: GameState) {
  if (game.status === 'setup') {
    return {
      icon: <IconHammer size={16} />,
      label: 'Initial placement',
      hint: 'Click any of your territories to drop an army. Take turns.',
    };
  }
  switch (game.turn.phase) {
    case 'place':
      return {
        icon: <IconCards size={16} />,
        label: 'Reinforce',
        hint: 'Click your territories to place fresh armies. Trade cards for bonuses.',
      };
    case 'attack':
      return {
        icon: <IconSwords size={16} />,
        label: 'Attack',
        hint: 'Pick one of your territories, then an adjacent enemy.',
      };
    case 'fortify':
      return {
        icon: <IconChevronRight size={16} />,
        label: 'Fortify',
        hint: 'One move between two adjacent friendly territories. Or skip.',
      };
    default:
      return { icon: null, label: '', hint: '' };
  }
}

function ContinentLegend({ game }: { game: GameState }) {
  const human = humanPlayer(game);
  if (!human) return null;
  return (
    <div>
      <Text size="xs" c="dimmed" tt="uppercase" fw={700} mb={4}>Continent bonuses</Text>
      <Stack gap={2}>
        {CONTINENTS.map((c) => {
          const owned = continentOwnerOrPartial(game, c.id, human.id);
          return (
            <Group key={c.id} justify="space-between">
              <Text size="xs">{c.name}</Text>
              <Group gap={4}>
                <Text size="xs" c="dimmed">{owned.owned}/{owned.total}</Text>
                <Badge size="xs" color={c.color}>+{c.bonus}</Badge>
              </Group>
            </Group>
          );
        })}
      </Stack>
    </div>
  );
}

function continentOwnerOrPartial(game: GameState, continentId: string, viewerId: string) {
  const tot = Object.entries(game.board).filter(([tid]) => {
    const td = territoryById(tid as never);
    return td?.continent === continentId;
  });
  const owned = tot.filter(([, ts]) => ts.ownerId === viewerId).length;
  return { owned, total: tot.length };
}

// ────── PlayersPanel ──────────────────────────────────────────────────────

export function PlayersPanel() {
  const game = useRiskStore((s) => s.game);
  if (!game) return null;
  return (
    <Paper withBorder radius="md" p="md">
      <Text size="xs" c="dimmed" tt="uppercase" fw={700} mb="xs">Players</Text>
      <Stack gap="xs">
        {game.players.map((p) => {
          const terrs = ownedCount(game, p.id);
          const armies = totalArmies(game, p.id);
          const isCurrent = p.id === game.turn.currentPlayerId;
          return (
            <Group key={p.id} justify="space-between" wrap="nowrap"
              style={{ opacity: p.eliminated ? 0.4 : 1 }}>
              <Group gap="xs" wrap="nowrap" style={{ minWidth: 0 }}>
                <ColorChip color={p.color} />
                <Text size="sm" fw={isCurrent ? 700 : 500} truncate="end">
                  {p.name}
                </Text>
              </Group>
              <Group gap={6} wrap="nowrap">
                <Tooltip label="Territories"><Badge variant="default" size="sm">{terrs}</Badge></Tooltip>
                <Tooltip label="Armies"><Badge variant="default" size="sm" color="electricBlue">{armies}</Badge></Tooltip>
                <Tooltip label="Cards"><Badge variant="default" size="sm" color="neonPink">{p.cards.length}</Badge></Tooltip>
              </Group>
            </Group>
          );
        })}
      </Stack>
    </Paper>
  );
}

// ────── EventLog ──────────────────────────────────────────────────────────

export function EventLog() {
  const game = useRiskStore((s) => s.game);
  const viewport = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    if (viewport.current) {
      viewport.current.scrollTop = viewport.current.scrollHeight;
    }
  }, [game?.events?.length]);

  if (!game) return null;
  return (
    <Paper withBorder radius="md" p="xs" style={{ flex: 1, minHeight: 0 }}>
      <Text size="xs" c="dimmed" tt="uppercase" fw={700} mb={4} px="xs">Event log</Text>
      <ScrollArea h="100%" viewportRef={viewport}>
        <Stack gap={2} px="xs" pb="xs">
          {game.events.map((ev) => (
            <EventLine key={ev.seq} ev={ev} game={game} />
          ))}
        </Stack>
      </ScrollArea>
    </Paper>
  );
}

function EventLine({ ev, game }: { ev: RiskEvent; game: GameState }) {
  const player = game.players.find((p) => p.id === ev.playerId);
  const tint = player?.color ?? 'gray';
  return (
    <Group gap={6} wrap="nowrap" align="flex-start">
      <Text size="xs" c={`${tint}.5`} fw={700} style={{ minWidth: 70 }}>
        {player?.name ?? ev.playerId}
      </Text>
      <Text size="xs" c="dimmed">{formatEvent(ev)}</Text>
    </Group>
  );
}

function formatEvent(ev: RiskEvent): string {
  const p = ev.payload ?? {};
  switch (ev.kind) {
    case 'turn_start':
      return `Turn ${p.turnNumber} — ${p.armiesToPlace} new armies`;
    case 'place':
      return `placed ${p.count} on ${niceTerr(p.territory)}`;
    case 'phase':
      return `→ ${p.to} phase`;
    case 'attack': {
      const ad = (p.attackerDice as number[]) ?? [];
      const dd = (p.defenderDice as number[]) ?? [];
      return `attacks ${niceTerr(p.to)} from ${niceTerr(p.from)} · A[${ad.join(',')}] vs D[${dd.join(',')}] · -${p.attackerLost}A / -${p.defenderLost}D`;
    }
    case 'conquer':
      return `conquered ${niceTerr(p.to)}!`;
    case 'post_conquest':
      return `moved ${p.moved} into ${niceTerr(p.to)}`;
    case 'fortify':
      return `fortified ${niceTerr(p.to)} with ${p.count} from ${niceTerr(p.from)}`;
    case 'draw_card':
      return `drew a card (${p.cardType})`;
    case 'trade_cards':
      return `traded set #${p.setNumber} → +${p.bonus} armies${p.territoryBonus ? ' (+2 territory bonus)' : ''}`;
    case 'eliminate':
      return `eliminated ${p.victim}; inherited ${p.cardsTransferred} cards`;
    case 'win':
      return `wins the world! 🏆`;
    case 'surrender':
      return `surrendered`;
    default:
      return ev.kind;
  }
}

function niceTerr(id: unknown): string {
  const td = territoryById(id as never);
  return td?.name ?? String(id ?? '');
}

// ────── ColorChip ─────────────────────────────────────────────────────────

export function ColorChip({ color, size = 12 }: { color: string; size?: number }) {
  return (
    <div style={{
      width: size, height: size, borderRadius: '50%',
      background: `var(--mantine-color-${color}-6, var(--mantine-color-gray-5))`,
      boxShadow: `0 0 6px var(--mantine-color-${color}-6, transparent)`,
      flexShrink: 0,
    }} />
  );
}
