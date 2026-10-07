# acq-agent

Prototype internal remote disk acquisition (peer-to-peer): satu node menarik
file image dari node lain lewat gRPC streaming, hash SHA-256 dicek di kedua
sisi, kejadian tercatat di audit log.

Status: POC, bukan produk jadi. Aturan main dan keputusan desain ada di
AGENTS.md.

## Perintah

- `make build` — kompilasi
- `make test` — unit test
- `make vet` — pemeriksaan statis
- `make lint` — lint (butuh golangci-lint)
- `make sec` — pindai keamanan (butuh gosec)

## Disk palsu untuk uji lokal

```sh
./scripts/ghostdisk.sh testdata/disk01.dd 16
```
