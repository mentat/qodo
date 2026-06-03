import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// Marvin's voice preference. Speech is ON by default; the user can mute, and
// the choice is persisted to localStorage so it survives reloads.
interface TTSState {
  muted: boolean;
  toggleMuted: () => void;
  setMuted: (muted: boolean) => void;
}

export const useTTSStore = create<TTSState>()(
  persist(
    (set) => ({
      muted: false,
      toggleMuted: () => set((s) => ({ muted: !s.muted })),
      setMuted: (muted) => set({ muted }),
    }),
    { name: 'marvin-tts' },
  ),
);
