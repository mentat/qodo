import { describe, it, expect, beforeEach } from 'bun:test';
import { useTTSStore } from '../src/store/ttsStore';

describe('ttsStore (Marvin voice preference)', () => {
  beforeEach(() => {
    localStorage.clear();
    useTTSStore.setState({ muted: false });
  });

  it('defaults to unmuted — speech on by default', () => {
    expect(useTTSStore.getState().muted).toBe(false);
  });

  it('toggles mute and persists the choice to localStorage', () => {
    useTTSStore.getState().toggleMuted();
    expect(useTTSStore.getState().muted).toBe(true);

    const raw = localStorage.getItem('marvin-tts');
    expect(raw).toBeTruthy();
    expect(JSON.parse(raw as string).state.muted).toBe(true);

    useTTSStore.getState().setMuted(false);
    expect(JSON.parse(localStorage.getItem('marvin-tts') as string).state.muted).toBe(false);
  });
});
