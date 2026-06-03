export interface DayForecast {
  date: string;
  weekday: string;
  highC: number;
  lowC: number;
  condition: string;
  icon: string;
  precipPct: number;
}

export interface Forecast {
  location: string;
  days: DayForecast[];
}
