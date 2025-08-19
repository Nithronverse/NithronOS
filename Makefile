SHELL := /usr/bin/env bash

.PHONY: web-dev api-dev agent-dev fmt lint package clean

web-dev:
	cd web && npm run dev

api-dev:
	@if command -v air >/dev/null 2>&1; then \
		cd backend/nosd && air; \
	elif command -v reflex >/dev/null 2>&1; then \
		cd backend/nosd && reflex -r '\\.go$$' -- sh -c 'go run ./...'; \
	else \
		cd backend/nosd && go run ./...; \
	fi

agent-dev:
	@if command -v air >/dev/null 2>&1; then \
		cd agent/nos-agent && air; \
	elif command -v reflex >/dev/null 2>&1; then \
		cd agent/nos-agent && reflex -r '\\.go$$' -- sh -c 'go run ./...'; \
	else \
		cd agent/nos-agent && go run ./...; \
	fi

fmt:
	cd backend/nosd && go fmt ./...
	cd agent/nos-agent && go fmt ./...
	cd web && npx prettier -w .

lint:
	cd backend/nosd && golangci-lint run || true
	cd agent/nos-agent && golangci-lint run || true
	cd web && npm run lint || true

run:
	cd backend/nosd && go run ./...

package:
	bash packaging/build-all.sh

clean:
	rm -rf packaging/output backend/nosd/build agent/nos-agent/build web/dist


