# AGENTS.md — acq-agent

Dokumen serah-terima untuk coding agent. Baca seluruhnya sebelum mengerjakan apa pun.

## 1. Aturan nomor satu

**Jangan mengambil keputusan sendiri dan jangan berasumsi.**
Jika menemukan hal yang tidak tercantum di bagian "Sudah diputuskan", atau tercantum di bagian "Belum diputuskan", **berhenti dan tanya pengguna dulu**. Jangan menandainya sebagai TODO lalu lanjut.

## 2. Tentang proyek

- Prototype internal untuk tool **remote disk acquisition**: menyalin isi disk dari satu node ke node lain lewat jaringan.
- Dibuat sebagai proof of concept untuk mentor/senior pengguna. Tujuannya membuktikan idenya jalan, **bukan** produk jadi. Jangan over-engineer.
- Nama repo: `acq-agent`. Tidak ada deadline.
- Pengguna mengembangkan di **macOS**.

## 3. Sudah diputuskan

**Stack dan bentuk**
- Bahasa: **Go**.
- Target OS agent: **Linux dan Windows**.
- Bentuk: **node yang bisa saling menarik data (peer-to-peer)**, bukan model agent/collector yang terpisah.
- Protokol: **gRPC dengan streaming**.
- Library pihak ketiga: **bebas**, selama tujuannya jalan.

**Keamanan komunikasi**
- Autentikasi: **mTLS**.
- Aturan izin (siapa boleh menarik siapa): **gabungan peran di sertifikat dan allowlist**.
- Sertifikat untuk testing/dev lokal dibuat dengan **program Go kecil** (hanya untuk latihan di laptop pengguna, bukan untuk produksi).

**Data dan log**
- Yang diacquire: **file image (disk palsu) saja**.
- Hash: **SHA-256**, dicek di **kedua sisi**.
- Audit log: **file log, satu baris JSON per kejadian**.

**Cara kerja**
- Operator memicu proses penarikan data dari sisi node yang berperan menarik.
- Kode berbahasa **Inggris**. Komentar dan dokumentasi berbahasa **Indonesia**.
- Commit message memakai **Conventional Commits** (`feat:`, `fix:`, dst).
- Perlu **Makefile** untuk menyingkat perintah (misalnya `make test`, `make build`).

## 4. Aturan keamanan (wajib)

Ini pengaman untuk mesin pengguna dan bagian dari proof of concept.

1. Disk/file dibuka **read-only**. Jangan pernah membuka dengan mode tulis.
2. **Tidak ada secret** (password, private key, token) yang ditulis di kode atau di-commit ke repo. File sertifikat/key hasil generate harus masuk `.gitignore`.
3. Agent **tidak pernah menyentuh disk fisik asli**. Hanya file image palsu. Jangan membuka `/dev/*`, `\\.\PhysicalDrive*`, atau loop device.

## 5. Cara kerja

- Kerjakan **satu langkah kecil**, lalu **berhenti dan tunggu pengguna memeriksa**. Jangan lanjut ke langkah berikutnya sebelum disetujui.
- Dalam satu langkah, boleh membuat/mengedit file seperlunya.
- Setelah tiap langkah, lapor singkat: apa yang dikerjakan, file yang berubah, hasil perintah yang dijalankan, dan pertanyaan yang muncul.
- Jangan menjalankan `git commit` sendiri. Pengguna yang commit. Boleh menyarankan commit message sesuai Conventional Commits.

## 6. Perintah yang boleh dijalankan sendiri

- `go build`
- `go test`
- `go vet`
- `golangci-lint`
- `gosec`
- `go get` (menambah dependency)

Perintah lain di luar daftar ini: **tanya dulu**. Termasuk `sudo`, `docker`, `git commit`, dan apa pun yang menyentuh device atau jaringan di luar kebutuhan testing lokal.

## 7. Testing

- Unit test (`go test`).
- Lint dengan `golangci-lint`.
- Scan keamanan dengan `gosec`.
- End-to-end Docker Compose **tidak dipilih** untuk prototype ini.
- Kode baca disk khusus Windows tidak bisa dites dari macOS. Catat keterbatasan ini, jangan berpura-pura sudah teruji.

## 8. Konvensi penamaan

Ikuti konvensi standar Go (Effective Go dan Go Code Review Comments): nama package huruf kecil satu kata, `PascalCase` untuk yang di-export, `camelCase` untuk yang tidak, akronim kapital (`sessionID`, `SHA256`), dan format dengan `gofmt`.

## 9. Definisi selesai

Prototype dianggap selesai jika:
1. Satu node menarik file image dari node lain (satu arah dulu),
2. hash SHA-256 di kedua sisi cocok, dan
3. kejadiannya tercatat di audit log.

## 10. Di luar scope sekarang

- Resume jika koneksi putus.
- Penanganan bad sector.
- Hal lain yang belum dibahas (kompresi, format E01, snapshot untuk live system, dll): jangan dikerjakan, tanya dulu jika terasa perlu.

## 11. Belum diputuskan (berhenti dan tanya)

- Format output image (raw `.dd` atau lainnya).
- Versi Go minimum.
- Struktur folder final dan bentuk binary (struktur lama untuk model agent/collector sudah tidak berlaku).
- Detail model peer-to-peer: siapa yang menghubungi siapa, bagaimana operator memicu penarikan, dan desain service/pesan gRPC.
- Nama peran di sertifikat, di bagian mana peran ditulis, dan format penyimpanan allowlist.
- Cara konfigurasi (flag, env var, atau file) dan lokasi file log.
- Module path Go dan hosting repo.
- Apakah agent boleh menjalankan `make` (Makefile hanya membungkus perintah yang sudah diizinkan).
- CI.

## 12. Referensi

- Ada repo publik berbahasa C# yang sudah teruji. Agent **boleh membacanya sebagai referensi logika**.
- URL dan lisensi repo itu belum dicatat. Jangan menyalin kode langsung. Jika ingin mengadopsi bagian tertentu, tanya pengguna dulu.