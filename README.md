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

## Coba transfer lokal (plaintext, dev saja)

```sh
go run ./cmd/agent serve --addr 127.0.0.1:50051 &
go run ./cmd/agent pull --from 127.0.0.1:50051 --src testdata/disk01.dd --out /tmp/hasil.dd
shasum -a 256 testdata/disk01.dd /tmp/hasil.dd
```

## Coba transfer mTLS + allowlist

```sh
go run ./tools/certgen --nodes node-a,node-b
echo '{"pairs":[["node-a","node-b"]]}' > /tmp/allow.json
go run ./cmd/agent serve --addr 127.0.0.1:50051 --tls-cert certs/node-a.pem --tls-key certs/node-a-key.pem --tls-ca certs/ca.pem --allowlist /tmp/allow.json &
go run ./cmd/agent pull --from 127.0.0.1:50051 --src testdata/disk01.dd --out /tmp/hasil.dd --tls-cert certs/node-b.pem --tls-key certs/node-b-key.pem --tls-ca certs/ca.pem
```

Isi `certs/` (CA, sertifikat, dan key) tidak di-commit ke repo.

## Definisi selesai (checklist POC)

- Satu node menarik file image dari node lain satu arah (`pull`)
- Hash SHA-256 kedua sisi cocok (diverifikasi otomatis; tidak cocok = gagal, file hasil dihapus)
- Kejadian tercatat di audit log (satu baris JSON canonical per kejadian, di kedua sisi)

## Keterbatasan

- Tanpa flag `--tls-*` koneksi plaintext (dev saja); POC aman selalu pakai mTLS + `--allowlist`.
- Guard path Windows (`\\.\`, `\\?\`) hanya teruji logikanya via unit test; tidak teruji di device Windows asli dari macOS.
- Di luar scope dan tidak dikerjakan: resume koneksi putus, bad sector, kompresi, format E01, snapshot live system.
- `certgen` hanya untuk latihan lokal, bukan PKI produksi.
