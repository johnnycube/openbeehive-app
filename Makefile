.PHONY: proto run-server dev-app tidy build clean

# Generate Go and TypeScript stubs from the .proto files.
proto:
	buf generate

# --- Dev: run the backend and the frontend in separate shells ---

# Backend (loads .env if present). Run the app separately with `make dev-app`.
run-server:
	cd server && set -a && [ -f ../.env ] && . ../.env; set +a; go run ./cmd/server

# Frontend dev server (Vite on :5173).
dev-app:
	cd app && npm install && npm run dev

tidy:
	cd server && go mod tidy

# --- Production: single binary with the SPA embedded ---

# Build the static SPA, embed it into the server package, and compile the
# single binary at server/bin/openbeehive.
build:
	cd app && npm install && npm run build
	rm -rf server/internal/web/dist
	mkdir -p server/internal/web/dist
	cp -r app/build/. server/internal/web/dist/
	cd server && go build -o bin/openbeehive ./cmd/server
	@echo "built server/bin/openbeehive (SPA embedded)"

clean:
	rm -rf app/build server/bin
	rm -rf server/internal/web/dist
	mkdir -p server/internal/web/dist
	touch server/internal/web/dist/.gitkeep
