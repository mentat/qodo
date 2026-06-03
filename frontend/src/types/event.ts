// CalendarEvent uses JS Date for start/end so it drops straight into
// react-big-calendar. The store normalizes Firestore Timestamps / RFC3339
// strings into Dates on read.
export interface CalendarEvent {
  id: string;
  userId: string;
  title: string;
  description: string;
  location: string;
  start: Date;
  end: Date;
  allDay: boolean;
  color: string;
  characterId?: string;
  // kind distinguishes real events from due-dated todos merged onto the grid.
  kind: 'event' | 'todo';
}
