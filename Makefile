# Perintah singkat. Hanya membungkus perintah yang diizinkan AGENTS.md.

.PHONY: test vet build

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...
