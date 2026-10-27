.PHONY: up down logs rebuild migrate-up migrate-down test release

up:
	node scripts/compose.mjs up --build -d

down:
	node scripts/compose.mjs down

logs:
	node scripts/compose.mjs logs --follow --tail=100

rebuild:
	node scripts/compose.mjs build --no-cache
	node scripts/compose.mjs up -d

migrate-up:
	node scripts/compose.mjs run --rm migrate up

# Revert exactly one migration; this can fail if existing rows reference it.
migrate-down:
	node scripts/compose.mjs run --rm migrate down 1

test:
	node scripts/test.mjs

release:
	node scripts/release.mjs "$(VERSION)"
