export GOEXPERIMENT = jsonv2

build:
	go build -o klar ./cmd/klar

gen:
	bun run ./scripts/replaceASTNodeImpls.ts
	go generate ./...