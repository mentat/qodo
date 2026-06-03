import { create } from 'zustand';

// AppId enumerates the suite's top-level apps shown in the left nav rail.
export type AppId = 'todos' | 'mail' | 'calendar' | 'contacts' | 'notes' | 'radio' | 'weather';

interface UIState {
  activeApp: AppId;
  chatOpen: boolean;
  setActiveApp: (a: AppId) => void;
  openChat: () => void;
  closeChat: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  activeApp: 'todos',
  chatOpen: false,
  setActiveApp: (activeApp) => set({ activeApp }),
  openChat: () => set({ chatOpen: true }),
  closeChat: () => set({ chatOpen: false }),
}));
