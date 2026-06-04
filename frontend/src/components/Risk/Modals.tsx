import { useEffect, useState } from 'react';
import {
  Modal, Stack, Group, Text, Button, Slider, Badge, Paper, Title,
} from '@mantine/core';
import { useRiskStore, conquestProbability } from '../../store/riskStore';
import { territoryById } from './board';
import type { TerritoryID } from './board';
import type { AttackResult, GameState } from '../../types/risk';

// ────── AttackModal ──────────────────────────────────────────────────────

interface AttackModalProps {
  from: TerritoryID;
  to: TerritoryID;
  onClose: () => void;
}

export function AttackModal({ from, to, onClose }: AttackModalProps) {
  const game = useRiskStore((s) => s.game);
  const attack = useRiskStore((s) => s.attack);
  const lastRounds = useRiskStore((s) => s.lastRounds);
  const [rolling, setRolling] = useState(false);
  const [rolledThisOpen, setRolledThisOpen] = useState<AttackResult[]>([]);

  if (!game) return null;
  const src = game.board[from];
  const dst = game.board[to];
  const showOdds = game.settings.difficulty === 'easy';
  const p = conquestProbability(src.armies, dst.armies);

  const runAttack = async (mode: 'single' | 'blitz') => {
    try {
      setRolling(true);
      const rounds = await attack(from, to, mode);
      setRolledThisOpen(rounds);
    } finally {
      setRolling(false);
    }
  };

  const fromName = territoryById(from)?.name ?? from;
  const toName = territoryById(to)?.name ?? to;

  return (
    <Modal opened onClose={onClose} title={<Title order={4}>Attack</Title>} centered radius="md">
      <Stack gap="md">
        <Group justify="space-between" align="center">
          <ArmyChip name={fromName} armies={src.armies} accent="electricBlue" />
          <Text size="xl" fw={900}>→</Text>
          <ArmyChip name={toName} armies={dst.armies} accent="neonPink" />
        </Group>

        {showOdds && (
          <Group justify="center">
            <Badge size="lg" color={p >= 0.5 ? 'neonGreen' : 'neonPink'}>
              P(conquest) ≈ {(p * 100).toFixed(0)}%
            </Badge>
          </Group>
        )}

        <DiceRoll rounds={rolledThisOpen.length ? rolledThisOpen : (lastRounds.length ? lastRounds : [])} />

        <Group justify="space-between">
          <Button variant="subtle" onClick={onClose} disabled={rolling}>
            Withdraw
          </Button>
          <Group>
            <Button variant="outline" onClick={() => void runAttack('single')}
              disabled={rolling || src.armies < 2}>
              Roll once
            </Button>
            <Button color="neonPink" onClick={() => void runAttack('blitz')}
              loading={rolling} disabled={src.armies < 2}>
              Blitz
            </Button>
          </Group>
        </Group>
        {game.turn.postConquestPending && (
          <Text size="sm" c="neonGreen.6" ta="center" fw={700}>
            Conquered! Move armies in the next dialog.
          </Text>
        )}
      </Stack>
    </Modal>
  );
}

function ArmyChip({ name, armies, accent }: { name: string; armies: number; accent: string }) {
  return (
    <Paper withBorder radius="md" p="sm" style={{ borderColor: `var(--mantine-color-${accent}-5)` }}>
      <Text size="xs" c="dimmed">{name}</Text>
      <Text size="xl" fw={900} c={`${accent}.6`}>{armies}</Text>
    </Paper>
  );
}

// ────── DiceRoll: animated dice ───────────────────────────────────────────

function DiceRoll({ rounds }: { rounds: AttackResult[] }) {
  // Show only the last round's dice; cumulative casualty count beneath.
  const last = rounds[rounds.length - 1];
  if (!last) {
    return (
      <Stack gap="xs" align="center">
        <Text size="sm" c="dimmed">Ready to roll.</Text>
      </Stack>
    );
  }
  const cumA = rounds.reduce((s, r) => s + r.attackerLost, 0);
  const cumD = rounds.reduce((s, r) => s + r.defenderLost, 0);
  return (
    <Stack gap={4} align="center">
      <Group gap={4}>
        {last.attackerDice.map((v, i) => <Die key={`a${i}`} value={v} accent="electricBlue" />)}
        <Text mx="md" fw={900}>vs</Text>
        {last.defenderDice.map((v, i) => <Die key={`d${i}`} value={v} accent="neonPink" />)}
      </Group>
      <Text size="sm" c="dimmed">
        Round {rounds.length} · Attacker -{cumA} · Defender -{cumD}
      </Text>
    </Stack>
  );
}

const DIE_FACES = ['', '⚀', '⚁', '⚂', '⚃', '⚄', '⚅'];

function Die({ value, accent }: { value: number; accent: string }) {
  return (
    <div
      style={{
        width: 44, height: 44,
        borderRadius: 8,
        background: `var(--mantine-color-${accent}-light)`,
        border: `2px solid var(--mantine-color-${accent}-5)`,
        boxShadow: `0 0 12px var(--mantine-color-${accent}-3)`,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 32,
        animation: 'riskDieRoll 350ms ease-out',
      }}
    >
      {DIE_FACES[value] ?? value}
    </div>
  );
}

// ────── PostConquestModal ─────────────────────────────────────────────────

export function PostConquestModal() {
  const game = useRiskStore((s) => s.game);
  const resolve = useRiskStore((s) => s.resolvePostConquest);
  const pc = game?.turn.postConquestPending;
  const [n, setN] = useState<number>(0);
  const postConquestKey = pc ? `${pc.from}:${pc.to}:${pc.minArmies}:${pc.maxArmies}` : '';
  const postConquestArmies = pc ? Math.min(pc.maxArmies, Math.max(pc.minArmies, pc.minArmies)) : 0;

  /* eslint-disable react-hooks/set-state-in-effect -- A new conquest resets the required move amount. */
  useEffect(() => {
    if (postConquestKey) setN(postConquestArmies);
  }, [postConquestKey, postConquestArmies]);
  /* eslint-enable react-hooks/set-state-in-effect */

  if (!game || !pc) return null;
  const fromName = territoryById(pc.from)?.name ?? pc.from;
  const toName = territoryById(pc.to)?.name ?? pc.to;

  return (
    <Modal opened onClose={() => undefined} closeOnEscape={false} closeOnClickOutside={false}
      withCloseButton={false} centered title={<Title order={4}>Move into {toName}</Title>}>
      <Stack gap="md">
        <Text size="sm">
          You conquered <b>{toName}</b>! How many armies do you move from <b>{fromName}</b>?
          (At least {pc.minArmies}, at most {pc.maxArmies}.)
        </Text>
        <Slider
          min={pc.minArmies}
          max={pc.maxArmies}
          value={n}
          onChange={setN}
          marks={[
            { value: pc.minArmies, label: String(pc.minArmies) },
            { value: pc.maxArmies, label: String(pc.maxArmies) },
          ]}
          step={1}
          color="synthPurple"
        />
        <Group justify="space-between">
          <Text size="xs" c="dimmed">
            Source will keep {(game.board[pc.from]?.armies ?? 0) - n} armies.
          </Text>
          <Button color="synthPurple" onClick={() => void resolve(n)}>
            Move {n} armies
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

// ────── FortifyModal ──────────────────────────────────────────────────────

interface FortifyModalProps {
  from: TerritoryID;
  to: TerritoryID;
  onClose: () => void;
}

export function FortifyModal({ from, to, onClose }: FortifyModalProps) {
  const game = useRiskStore((s) => s.game);
  const fortify = useRiskStore((s) => s.fortify);
  const src = game?.board[from];
  const max = (src?.armies ?? 1) - 1;
  const initialFortifyArmies = Math.max(1, max);
  const [n, setN] = useState(initialFortifyArmies);

  /* eslint-disable react-hooks/set-state-in-effect -- Opening a different fortify route resets the slider amount. */
  useEffect(() => { setN(initialFortifyArmies); }, [initialFortifyArmies]);
  /* eslint-enable react-hooks/set-state-in-effect */

  if (!game || !src) return null;
  const fromName = territoryById(from)?.name ?? from;
  const toName = territoryById(to)?.name ?? to;

  return (
    <Modal opened onClose={onClose} centered title={<Title order={4}>Fortify</Title>} radius="md">
      <Stack gap="md">
        <Text size="sm">
          Move armies from <b>{fromName}</b> ({src.armies}) to <b>{toName}</b> (
          {game.board[to]?.armies ?? 0}). Source must keep at least 1.
        </Text>
        <Slider min={1} max={max} value={n} onChange={setN} step={1} color="synthPurple" />
        <Group justify="space-between">
          <Button variant="subtle" onClick={onClose}>Cancel</Button>
          <Button color="synthPurple" onClick={() => void fortify(from, to, n).then(onClose)}>
            Move {n}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

// ────── SurrenderConfirm ──────────────────────────────────────────────────

export function SurrenderConfirm({ opened, onConfirm, onClose }: {
  opened: boolean; onConfirm: () => void; onClose: () => void;
}) {
  return (
    <Modal opened={opened} onClose={onClose} centered title={<Title order={4}>Surrender?</Title>}>
      <Stack gap="md">
        <Text size="sm">
          This will end the current game and record a surrender. You can start a new one immediately.
        </Text>
        <Group justify="flex-end">
          <Button variant="subtle" onClick={onClose}>Keep playing</Button>
          <Button color="neonPink" onClick={onConfirm}>Surrender</Button>
        </Group>
      </Stack>
    </Modal>
  );
}

export function GameOverBanner({ game, onAck }: { game: GameState; onAck: () => void }) {
  const isWin = game.status === 'won';
  const color = isWin ? 'neonGreen' : 'neonPink';
  return (
    <Modal opened onClose={onAck} centered withCloseButton={false} size="lg">
      <Stack gap="md" align="center" p="lg">
        <Title order={1} c={`${color}.6`}>{isWin ? 'Victory' : 'Defeat'}</Title>
        <Text size="md" ta="center">
          {isWin
            ? 'The world is yours. Synthwave forever.'
            : 'One of the generals wears the crown. For now.'}
        </Text>
        <Button size="lg" color={color} onClick={onAck}>Back to lobby</Button>
      </Stack>
    </Modal>
  );
}
