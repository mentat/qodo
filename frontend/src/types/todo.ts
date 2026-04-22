export interface Todo {
  id: string;
  title: string;
  description: string;
  completed: boolean;
  priority: 'low' | 'medium' | 'high';
  category: string;
  dueDate: string | null;
  position: number;
  userId: string;
  createdAt: string;
  updatedAt: string;
}

export type TodoCreate = Pick<Todo, 'title'> &
  Partial<Pick<Todo, 'description' | 'priority' | 'category' | 'dueDate'>>;

export type TodoUpdate = Partial<Omit<Todo, 'id' | 'userId' | 'createdAt' | 'updatedAt'>>;
