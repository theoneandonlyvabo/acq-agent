// Modal pratinjau heksadesimal isi file di node sumber (16 byte/baris + ASCII).
// Paging 4 KiB; Escape/muklik latar menutup.
import { useEffect, useState } from 'react'
import { previewSource } from '../api'

const PAGE = 4096
const PER_ROW = 16

function toBytes(b64: string): Uint8Array {
  const bin = atob(b64)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
  return out
}

function hex(n: number, pad: number): string {
  return n.toString(16).padStart(pad, '0')
}

function ascii(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => (b >= 32 && b <= 126 ? String.fromCharCode(b) : '·'))
    .join('')
}

interface Props {
  from: string
  path: string
  onClose: () => void
}

export default function HexModal({ from, path, onClose }: Props) {
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [data, setData] = useState<Uint8Array>(new Uint8Array())
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    previewSource(from, path, offset, PAGE).then(
      (r) => {
        if (cancelled) return
        setTotal(r.total_bytes)
        setData(toBytes(r.data_base64))
        setLoading(false)
      },
      (e) => {
        if (cancelled) return
        setError(e instanceof Error ? e.message : 'Gagal memuat pratinjau')
        setLoading(false)
      },
    )
    return () => {
      cancelled = true
    }
  }, [from, path, offset])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const rows: { at: number; bytes: Uint8Array }[] = []
  for (let i = 0; i < data.length; i += PER_ROW) {
    rows.push({ at: offset + i, bytes: data.slice(i, i + PER_ROW) })
  }

  const end = Math.min(offset + data.length, total)
  const prevDisabled = loading || offset === 0
  const nextDisabled = loading || offset + PAGE >= total

  return (
    <div
      className="overlay"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="dialog" role="dialog" aria-modal="true" aria-label={`Pratinjau ${path}`}>
        <div className="dialog-head">
          <div>
            <h2>Pratinjau isi</h2>
            <p className="mono dialog-path">{path}</p>
          </div>
          <button type="button" className="btn btn-secondary" onClick={onClose}>
            Tutup
          </button>
        </div>
        <div className="dialog-body">
          {loading ? (
            <p className="empty">Memuat…</p>
          ) : error ? (
            <p className="form-error" role="alert">
              {error}
            </p>
          ) : rows.length === 0 ? (
            <p className="empty">Kosong pada offset ini.</p>
          ) : (
            rows.map((r) => {
              const hexPart = Array.from(r.bytes)
                .map((b) => hex(b, 2))
                .join(' ')
                .padEnd(PER_ROW * 3 - 1)
              return (
                <div key={r.at} className="hex-row mono">
                  <span className="hex-off">{hex(r.at, 8)}</span>
                  <span className="hex-bytes">{hexPart}</span>
                  <span className="hex-ascii">{ascii(r.bytes)}</span>
                </div>
              )
            })
          )}
        </div>
        <div className="dialog-foot">
          <button
            type="button"
            className="btn btn-secondary"
            disabled={prevDisabled}
            onClick={() => setOffset((o) => Math.max(0, o - PAGE))}
          >
            ← Sebelumnya
          </button>
          <span className="mono dialog-pos">
            {total === 0 ? '0 dari 0' : `${offset.toLocaleString('id-ID')}–${end.toLocaleString('id-ID')} dari ${total.toLocaleString('id-ID')}`}
          </span>
          <button
            type="button"
            className="btn btn-secondary"
            disabled={nextDisabled}
            onClick={() => setOffset((o) => o + PAGE)}
          >
            Berikutnya →
          </button>
        </div>
      </div>
    </div>
  )
}
