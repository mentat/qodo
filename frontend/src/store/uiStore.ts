import { create } from 'zustand';

// App navigation now lives in the URL (react-router); this store only holds
// the global Marvin chat drawer state.
interface UIState {
  chatOpen: boolean;
  openChat: () => void;
  closeChat: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  chatOpen: false,
  openChat: () => set({ chatOpen: true }),
  closeChat: () => set({ chatOpen: false }),
}));
