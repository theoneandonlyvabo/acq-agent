# Perintah singkat. Hanya membungkus perintah yang diizinkan AGENTS.md
# ditambah pnpm untuk view (disetujui terpisah).
# Tool Go (buf, golangci-lint, gosec) dipakai dari GOPATH/bin bila ada.
# Path absolut (lookup PATH make-3.81 tidak bisa diandalkan untuk dir ini).

GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

.PHONY: all build vet test lint sec proto seed view-install view-build view-lint demo dev dev-down clean help

all: proto build vet test lint sec view-build view-lint

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

lint:
	$(GOBIN)/golangci-lint run ./...

sec:
	$(GOBIN)/gosec ./cmd/... ./internal/... ./tools/...

proto:
	$(GOBIN)/buf lint
	$(GOBIN)/buf generate

seed:
	./scripts/ghostdisk.sh testdata/disk-a.dd 4 "ACQAGENT-TEST-DISK-A"
	./scripts/ghostdisk.sh testdata/disk-b.dd 8 "ACQAGENT-TEST-DISK-B"
	./scripts/ghostdisk.sh testdata/disk-c.dd 1 "ACQAGENT-TEST-DISK-C"

view-install:
	cd view && pnpm install

view-build: view-install
	cd view && pnpm run build

view-lint: view-install
	cd view && pnpm run lint

demo: build
	@if [ -f /tmp/acq-agent-demo.pid ]; then pid=`cat /tmp/acq-agent-demo.pid`; if ps -p $$pid 2>/dev/null | grep -q acq-agent-demo; then kill $$pid && echo "mati server demo basi: $$pid"; fi; rm -f /tmp/acq-agent-demo.pid; fi
	./scripts/ghostdisk.sh testdata/demo.dd 8
	go build -o /tmp/acq-agent-demo ./cmd/agent
	(/tmp/acq-agent-demo serve --addr 127.0.0.1:50051 --audit-log /tmp/acq-agent-demo-serve.log & echo $$! > /tmp/acq-agent-demo.pid)
	sleep 1
	/tmp/acq-agent-demo pull --from 127.0.0.1:50051 --audit-log /tmp/acq-agent-demo-pull.log --src testdata/demo.dd --out /tmp/acq-agent-demo-out.dd
	cmp testdata/demo.dd /tmp/acq-agent-demo-out.dd && echo "demo OK: file identik"
	grep -q '"event":"pull.completed"' /tmp/acq-agent-demo-serve.log && grep -q '"event":"pull.completed"' /tmp/acq-agent-demo-pull.log && echo "demo OK: audit tercatat dua sisi"
	-kill `cat /tmp/acq-agent-demo.pid`
	rm -f testdata/demo.dd /tmp/acq-agent-demo /tmp/acq-agent-demo-serve.log /tmp/acq-agent-demo-pull.log /tmp/acq-agent-demo-out.dd /tmp/acq-agent-demo.pid

dev: build view-install
	@test -f testdata/dev.dd || ./scripts/ghostdisk.sh testdata/dev.dd 16
	go build -o /tmp/acq-agent-dev ./cmd/agent
	trap 'kill `cat /tmp/acq-agent-dev.pid` 2>/dev/null; rm -f /tmp/acq-agent-dev.pid /tmp/acq-agent-dev /tmp/acq-agent-dev-serve.log' EXIT INT TERM; (/tmp/acq-agent-dev serve --addr 127.0.0.1:50051 --audit-log /tmp/acq-agent-dev-serve.log --http 127.0.0.1:8080 & echo $$! > /tmp/acq-agent-dev.pid); sleep 2; curl -s --max-time 5 localhost:8080/api/status > /dev/null && echo "API: http://127.0.0.1:8080/api/status"; echo "UI:  http://localhost:5173 (Ctrl+C berhenti)"; cd view && pnpm dev --port 5173

dev-down:
	-@if [ -f /tmp/acq-agent-dev.pid ]; then pid=`cat /tmp/acq-agent-dev.pid`; if ps -p $$pid 2>/dev/null | grep -q acq-agent-dev; then kill $$pid && echo "mati serve: $$pid"; fi; rm -f /tmp/acq-agent-dev.pid; fi
	-@for p in `ps aux | grep -F "vite --port 5173" | grep -v grep | awk '{print $$2}'`; do kill $$p 2>/dev/null && echo "mati vite: $$p"; done
	rm -f /tmp/acq-agent-dev /tmp/acq-agent-dev-serve.log

clean:
	rm -f testdata/*.dd audit.log
	rm -rf view/dist

help:
	@echo "make all          bangun + semua cek (Go dan view)"
	@echo "make demo         demo transfer lokal ujung-ke-ujung"
	@echo "make dev            seed + serve + bridge + dev UI (satu command, Ctrl+C berhenti)"
	@echo "make dev-down       matikan sisa dev + bersih"
	@echo "make build/vet/test/lint/sec   cek Go"
	@echo "make proto        lint + generate kode gRPC"
	@echo "make seed         tiga disk uji berlabel di testdata/"
	@echo "make view-install/view-build/view-lint   perintah view"
	@echo "make clean        hapus artefak generate lokal"
