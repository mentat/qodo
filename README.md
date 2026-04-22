# Qodo TODO App

Full-stack TODO application with Firebase Auth, Firestore, and Cloud Run.

## Architecture

- **Frontend**: React + Mantine v8 + Zustand, deployed to Firebase Hosting (`qodo-demo.web.app`)
- **Backend**: Go API with chi router, deployed to Cloud Run (`qodo-api`)
- **Database**: Firestore
- **Auth**: Firebase Auth (Email/Password + Google)
- **CI/CD**: GitHub Actions — auto-deploys on merge to `main`

## Local Development

```bash
# Install all dependencies
make setup

# Run both API and frontend
make dev
```

- API runs on `http://localhost:8080`
- Frontend runs on `http://localhost:5173` (Vite proxies `/api` to the API)

### Prerequisites

- Go 1.24+
- Bun
- `service-account.json` in the project root (for local Firestore/Auth access)

## Testing

```bash
make test          # Run all tests
make test-api      # Go tests only
make test-frontend # Frontend tests only
```

## Deployment

```bash
make deploy            # Deploy everything
make deploy-api        # Deploy API to Cloud Run
make deploy-frontend   # Deploy frontend to Firebase Hosting
```

## Project Structure

```
├── api/           # Go API server
├── frontend/      # React + Mantine frontend
├── firebase.json  # Firebase Hosting + Firestore config
├── firestore.*    # Firestore rules and indexes
├── Makefile       # Dev, test, and deploy commands
└── .github/       # CI/CD workflows
```
