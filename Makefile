.PHONY: install format format-check test vet build check scan loc ci release-check

install:
	npm ci

format:
	gofmt -w src
	npx prettier --write .

format-check:
	node scripts/gofmt-check.mjs
	npx prettier --check .

test:
	go test ./...
	npm test

vet:
	go vet ./...

build:
	npm run build

check:
	npm run check

scan:
	npm run scan:public

loc:
	npm run loc

ci:
	npm run ci

release-check:
	npm run release:check
