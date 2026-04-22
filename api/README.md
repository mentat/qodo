# Qodo TODO API

Go REST API for the Qodo TODO app, using Firestore and Firebase Auth.

## Endpoints

| Method | Path                 | Auth | Description             |
|--------|----------------------|------|-------------------------|
| GET    | `/health`            | No   | Health check            |
| GET    | `/api/todos`         | Yes  | List user's todos       |
| POST   | `/api/todos`         | Yes  | Create a todo           |
| GET    | `/api/todos/{id}`    | Yes  | Get a single todo       |
| PUT    | `/api/todos/{id}`    | Yes  | Full update a todo      |
| PATCH  | `/api/todos/{id}`    | Yes  | Partial update a todo   |
| DELETE | `/api/todos/{id}`    | Yes  | Delete a todo           |
| POST   | `/api/todos/reorder` | Yes  | Batch reorder positions |

## Running Locally

```bash
GOOGLE_APPLICATION_CREDENTIALS=../service-account.json go run .
```

The API listens on port `8080` by default (override with `PORT` env var).

## Environment Variables

| Variable                       | Default     | Description                |
|--------------------------------|-------------|----------------------------|
| `PORT`                         | `8080`      | HTTP listen port           |
| `GOOGLE_CLOUD_PROJECT`         | `qodo-demo` | GCP project ID             |
| `GOOGLE_APPLICATION_CREDENTIALS` | -         | Path to service account key |
