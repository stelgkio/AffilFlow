# Run from repo root. Backend and web also have their own Makefiles under those directories.
.PHONY: fabric-up fabric-down compose-up compose-down

# Docker Compose: Postgres + API — see docker-compose.yml
compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

# Hyperledger Fabric via official fabric-samples test-network (Docker). See infra/fabric/README.md
fabric-up:
	bash scripts/fabric-test-network.sh up

fabric-down:
	bash scripts/fabric-test-network.sh down
