#!/bin/sh
# Membuat file image palsu untuk pengujian acq-agent.
# Pemakaian: ghostdisk.sh <file-keluar> [ukuran-MiB, default 16]
set -eu
OUT="${1:?pakai: ghostdisk.sh <file-keluar> [ukuran-MiB]}"
MIB="${2:-16}"
head -c "$((MIB * 1024 * 1024))" /dev/urandom > "$OUT"
ls -l "$OUT"
