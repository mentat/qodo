export interface Contact {
  id: string;
  userId: string;
  name: string;
  email: string;
  phone: string;
  company: string;
  notes: string;
  characterId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ContactCreate {
  name: string;
  email?: string;
  phone?: string;
  company?: string;
  notes?: string;
}
