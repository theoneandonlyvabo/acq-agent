#!/bin/sh
# Membuat file image palsu untuk pengujian acq-agent.
# Pemakaian: ghostdisk.sh <file-keluar> [ukuran-MiB, default 16] [label]
# Label (opsional) ditulis sebagai teks di awal file agar mudah dikenali
# saat pratinjau heksadesimal; sisanya byte acak.
set -eu
OUT="${1:?pakai: ghostdisk.sh <file-keluar> [ukuran-MiB] [label]}"
MIB="${2:-16}"
LABEL="${3:-}"
BYTES="$((MIB * 1024 * 1024))"
if [ -n "$LABEL" ]; then
  printf '%s' "$LABEL" > "$OUT"
  SISA="$((BYTES - $(wc -c < "$OUT")))"
  if [ "$SISA" -gt 0 ]; then
    head -c "$SISA" /dev/urandom >> "$OUT"
  fi
else
  head -c "$BYTES" /dev/urandom > "$OUT"
fi
ls -l "$OUT"
