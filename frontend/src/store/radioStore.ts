import { create } from 'zustand';
import type { Track } from '../types/radio';
import { fetchTracks } from '../api/radio';
import * as engine from '../audio/audioEngine';

interface RadioState {
  tracks: Track[];
  current: Track | null;
  playing: boolean;
  loaded: boolean;
  load: () => Promise<void>;
  play: (track: Track) => Promise<void>;
  toggle: () => Promise<void>;
  pause: () => void;
  setPlaying: (playing: boolean) => void;
}

export const useRadioStore = create<RadioState>((set, get) => ({
  tracks: [],
  current: null,
  playing: false,
  loaded: false,

  load: async () => {
    if (get().loaded) return;
    const tracks = await fetchTracks();
    set({ tracks, loaded: true });
  },

  play: async (track) => {
    try {
      await engine.play(track);
      set({ current: track, playing: true });
    } catch (e) {
      console.error('radio play failed', e);
      set({ playing: false });
    }
  },

  toggle: async () => {
    const { current, tracks, playing } = get();
    if (playing) {
      engine.pause();
      set({ playing: false });
      return;
    }
    const t = current ?? tracks[0];
    if (!t) return;
    await get().play(t);
  },

  pause: () => {
    engine.pause();
    set({ playing: false });
  },

  setPlaying: (playing) => set({ playing }),
}));
