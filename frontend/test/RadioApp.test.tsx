import { afterEach, describe, expect, it } from 'bun:test';
import { act, screen } from '@testing-library/react';
import { renderWithMantine } from './render';
import { RadioApp } from '../src/components/Radio/RadioApp';
import { useRadioStore } from '../src/store/radioStore';
import type { Track } from '../src/types/radio';

const initialRadioState = useRadioStore.getState();

const tracks: Track[] = [
  {
    id: 'midnight-drive',
    title: 'Midnight Drive',
    artist: 'Neon Operator',
    url: '/api/radio/stream?id=midnight-drive',
  },
  {
    id: 'chrome-sunset',
    title: 'Chrome Sunset',
    artist: 'Tape Echo',
    url: '/api/radio/stream?id=chrome-sunset',
  },
];

function seedRadio(playing: boolean) {
  act(() => {
    useRadioStore.setState({
      tracks,
      current: tracks[0],
      playing,
      loaded: true,
    });
  });
}

afterEach(() => {
  act(() => {
    useRadioStore.setState(initialRadioState, true);
  });
});

describe('RadioApp boombox skin', () => {
  it('shows pause controls and animatable cassette state while playing', () => {
    seedRadio(true);

    renderWithMantine(<RadioApp />);

    expect(screen.getByRole('button', { name: /pause radio/i })).toBeInTheDocument();
    expect(screen.getByTestId('radio-cassette')).toHaveClass('is-playing');
  });

  it('shows play controls and a stopped cassette state while paused', () => {
    seedRadio(false);

    renderWithMantine(<RadioApp />);

    expect(screen.getByRole('button', { name: /play radio/i })).toBeInTheDocument();
    expect(screen.getByTestId('radio-cassette')).not.toHaveClass('is-playing');
  });
});
