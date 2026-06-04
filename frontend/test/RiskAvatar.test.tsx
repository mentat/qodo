import { describe, expect, it, mock } from 'bun:test';
import { screen } from '@testing-library/react';
import { renderWithMantine } from './render';
import type { GameState } from '../src/types/risk';

mock.module('@lottiefiles/dotlottie-react', () => ({
  DotLottieReact: (props: Record<string, unknown>) => (
    <canvas
      data-testid="risk-avatar-lottie"
      data-src={String(props.src ?? '')}
      data-state-machine-id={String(props.stateMachineId ?? '')}
      data-animation-id={String(props.animationId ?? '')}
      aria-label={String(props['aria-label'] ?? '')}
    />
  ),
}));

const { RiskAvatarFrame } = await import('../src/components/Risk/RiskAvatar');
const { avatarEventForPlayer } = await import('../src/components/Risk/RiskAvatarEvents');
const { GENERALS } = await import('../src/components/Risk/cast');

describe('RiskAvatarFrame', () => {
  it('renders a dotLottie avatar with the risk state machine', () => {
    const general = GENERALS[0];

    renderWithMantine(
      <RiskAvatarFrame general={general} active thinking mood="nod" moodKey={7} />,
    );

    const avatar = screen.getByTestId('risk-avatar-lottie');
    expect(avatar).toHaveAttribute('data-src', general.avatarSrc);
    expect(avatar).toHaveAttribute('data-state-machine-id', 'risk-avatar');
    expect(avatar).toHaveAttribute('aria-label', general.avatarAlt);
  });

  it('uses the idle animation instead of the state machine when motion is reduced', () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = ((query: string) => ({
      matches: query.includes('prefers-reduced-motion'),
      media: query,
      onchange: null,
      addListener() {},
      removeListener() {},
      addEventListener() {},
      removeEventListener() {},
      dispatchEvent() {
        return false;
      },
    })) as typeof window.matchMedia;

    try {
      renderWithMantine(<RiskAvatarFrame general={GENERALS[1]} />);

      const avatar = screen.getByTestId('risk-avatar-lottie');
      expect(avatar).toHaveAttribute('data-state-machine-id', '');
      expect(avatar).toHaveAttribute('data-animation-id', 'idle');
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });
});

describe('avatarEventForPlayer', () => {
  it('maps recent combat events to avatar moods', () => {
    const game = {
      events: [
        {
          seq: 4,
          ts: '',
          playerId: 'ai-1',
          kind: 'attack',
          payload: { to: 'alaska', attackerLost: 2, defenderLost: 0 },
        },
      ],
      board: {
        alaska: { ownerId: 'human', armies: 3 },
      },
    } as unknown as GameState;

    expect(avatarEventForPlayer(game, 'ai-1')).toEqual({ mood: 'shake', key: 4 });
    expect(avatarEventForPlayer(game, 'human')).toEqual({ mood: 'startled', key: 4 });
  });
});
