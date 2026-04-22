.PHONY: setup dev dev-api dev-frontend test test-api test-frontend deploy deploy-api deploy-frontend build-api build-frontend serve

PROJECT_ID := qodo-demo
REGION := us-central1
AR_REPO := us-central1-docker.pkg.dev/$(PROJECT_ID)/qodo
API_IMAGE := $(AR_REPO)/qodo-api
ACCOUNT := jesse.l@qodo.ai

# ── Setup ──────────────────────────────────────────────────────────
setup:
	cd api && go mod download
	cd frontend && bun install

# ── Local Development ──────────────────────────────────────────────
dev:
	$(MAKE) dev-api & $(MAKE) dev-frontend & wait

dev-api:
	cd api && GOOGLE_APPLICATION_CREDENTIALS=../service-account.json PORT=4090 go run .

dev-frontend:
	cd frontend && bun run dev

# ── Testing ────────────────────────────────────────────────────────
test: test-api test-frontend

test-api:
	cd api && go test ./...

test-frontend:
	cd frontend && bun test

# ── Build ──────────────────────────────────────────────────────────
build-api:
	docker build -t $(API_IMAGE):latest ./api

build-frontend:
	cd frontend && bun run build

# ── Deploy ─────────────────────────────────────────────────────────
deploy: deploy-api deploy-frontend

deploy-api: build-api
	docker push $(API_IMAGE):latest
	gcloud run deploy qodo-api \
		--image $(API_IMAGE):latest \
		--region $(REGION) \
		--project $(PROJECT_ID) \
		--account $(ACCOUNT) \
		--platform managed \
		--allow-unauthenticated \
		--set-env-vars GOOGLE_CLOUD_PROJECT=$(PROJECT_ID)

deploy-frontend: build-frontend
	firebase deploy --only hosting --project $(PROJECT_ID) --account $(ACCOUNT)

# ── Supervisord ────────────────────────────────────────────────────
serve:
	supervisord -c supervisord.conf
