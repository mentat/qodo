import { create } from 'zustand';
import * as api from '../api/weather';
import type { Forecast } from '../types/weather';

interface WeatherState {
  forecast: Forecast | null;
  location: string;
  loading: boolean;
  fetch: (location?: string) => Promise<void>;
  setLocation: (location: string) => void;
}

export const useWeatherStore = create<WeatherState>((set, get) => ({
  forecast: null,
  location: 'Neo Tokyo',
  loading: false,

  fetch: async (location) => {
    const loc = location ?? get().location;
    set({ loading: true, location: loc });
    try {
      set({ forecast: await api.fetchForecast(loc, 6) });
    } finally {
      set({ loading: false });
    }
  },

  setLocation: (location) => set({ location }),
}));
