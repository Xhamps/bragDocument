.PHONY: dev build test lint fmt typecheck docker

dev: ; npm run dev
build: ; npm run build
test: ; npm test
lint: ; npm run lint && npm run fmt:check
fmt: ; npm run fmt
typecheck: ; npm run typecheck
docker: ; docker build -t bragdoc-frontend .
