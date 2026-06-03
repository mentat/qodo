import { create } from 'zustand';

// App navigation now lives in the URL (react-router); this store holds the
// global Marvin chat drawer state, plus an optional one-shot prefill so apps
// like Risk can hand Marvin a contextual question when opening the chat.
interface UIState {
  chatOpen: boolean;
  chatPrefill: string | null;
  openChat: () => void;
  openChatWithPrefill: (text: string) => void;
  consumePrefill: () => string | null;
  closeChat: () => void;
}

export const useUIStore = create<UIState>((set, get) => ({
  chatOpen: false,
  chatPrefill: null,
  openChat: () => set({ chatOpen: true }),
  openChatWithPrefill: (text) => set({ chatOpen: true, chatPrefill: text }),
  consumePrefill: () => {
    const v = get().chatPrefill;
    if (v) set({ chatPrefill: null });
    return v;
  },
  closeChat: () => set({ chatOpen: false }),
}));
