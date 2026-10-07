# Perintah singkat. Hanya membungkus perintah yang diizinkan AGENTS.md.

.PHONY: test vet build lint sec

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...

lint:
	golangci-lint run ./...

sec:
	gosec ./...
