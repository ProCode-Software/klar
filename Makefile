export GOEXPERIMENT = jsonv2

build:
	go build -o klar ./cmd/klar

gen:
	go generate ./...