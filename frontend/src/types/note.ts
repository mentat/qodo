export interface Note {
  id: string;
  userId: string;
  title: string;
  body: string;
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface NoteCreate {
  title: string;
  body: string;
  tags?: string[];
}
