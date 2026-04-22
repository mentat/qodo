# Qodo TODO Frontend

React frontend built with Mantine v8, Zustand, and Vite.

## Tech Stack

- **React 19** + TypeScript
- **Mantine v8** — UI components, forms, dates, notifications
- **Zustand** — State management
- **@hello-pangea/dnd** — Drag and drop reordering
- **Firebase Auth** — Email/password + Google sign-in
- **Vite** — Build tool

## Running Locally

```bash
bun install
bun run dev
```

The dev server runs on `http://localhost:5173` and proxies `/api` requests to `http://localhost:8080`.

## Building

```bash
bun run build
```

Output is in `dist/`, deployed to Firebase Hosting.

## Features

- Login / Sign up with email or Google
- Create, edit, delete todos
- Priority levels (low, medium, high)
- Categories
- Due dates with overdue highlighting
- Drag and drop reordering
- Search and filter
- Dark mode toggle
