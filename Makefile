export GOEXPERIMENT = jsonv2
.PHONY: target

build:
	go build -o klar ./cmd/klar

gen:
	@if ! go generate ./...; then \
		bun run ./scripts/replaceASTNodeImpls.ts; \
		go generate ./...; \
	fi
