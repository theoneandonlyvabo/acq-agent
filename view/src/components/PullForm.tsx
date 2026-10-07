// Form penarikan baru: grid node, dropdown path + tambah baru, hasil otomatis.
// File hasil = basename path sumber, tampil read-only.
import { useEffect, useState, type FormEvent } from 'react'
import NodeGrid, { type NodeState } from './NodeGrid'
import HexModal from './HexModal'
import { loadList, removeEntry, saveEntry } from '../storage'
import { probeEndpoints } from '../api'

const NEW_SENTINEL = '__baru__'
const DEFAULT_FROM = '127.0.0.1:50051'

interface Props {
  busy: boolean
  serverError: string | null
  onStart: (from: string, src: string, out: string) => void
}

// basename mengambil nama file dari path ala Unix maupun Windows.
function basename(path: string): string {
  const parts = path.split(/[\\/]/).filter(Boolean)
  return parts.length > 0 ? parts[parts.length - 1] : ''
}

export default function PullForm({ busy, serverError, onStart }: Props) {
  const [endpoints, setEndpoints] = useState<string[]>(() => loadList('endpoints'))
  const [paths, setPaths] = useState<string[]>(() => loadList('paths'))
  const [selected, setSelected] = useState<string>(() => loadList('endpoints')[0] ?? '')
  const [addingNew, setAddingNew] = useState<boolean>(() => loadList('endpoints').length === 0)
  const [custom, setCustom] = useState(DEFAULT_FROM)
  const [selectedPath, setSelectedPath] = useState<string>(() => loadList('paths')[0] ?? '')
  const [addingPath, setAddingPath] = useState<boolean>(() => loadList('paths').length === 0)
  const [customPath, setCustomPath] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [states, setStates] = useState<Record<string, NodeState>>({})
  const [nonce, setNonce] = useState(0)
  const [previewOpen, setPreviewOpen] = useState(false)

  useEffect(() => {
    if (endpoints.length === 0) return
    setStates(Object.fromEntries(endpoints.map((e) => [e, 'checking' as NodeState])))
    let cancelled = false
    probeEndpoints(endpoints).then((result) => {
      if (!cancelled) setStates(result)
    })
    return () => {
      cancelled = true
    }
  }, [endpoints, nonce])

  const effectiveSrc = addingPath ? customPath : selectedPath
  const outName = basename(effectiveSrc)
  const effectiveFrom = (addingNew ? custom : selected).trim()
  const canPreview = effectiveFrom !== '' && effectiveSrc.trim() !== '' && !busy

  function pickPath(value: string) {
    if (value === NEW_SENTINEL) {
      setAddingPath(true)
      return
    }
    setAddingPath(false)
    setSelectedPath(value)
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    const cleanFrom = (addingNew ? custom : selected).trim()
    const cleanSrc = effectiveSrc.trim()
    const cleanOut = basename(cleanSrc)
    if (!cleanFrom || !cleanSrc) {
      setError('Alamat sumber dan path sumber wajib diisi.')
      return
    }
    if (!cleanOut) {
      setError('Path sumber tidak valid.')
      return
    }
    setError(null)
    setEndpoints(saveEntry('endpoints', cleanFrom))
    setPaths(saveEntry('paths', cleanSrc))
    setSelected(cleanFrom)
    setAddingNew(false)
    setSelectedPath(cleanSrc)
    setAddingPath(false)
    onStart(cleanFrom, cleanSrc, cleanOut)
  }

  const message = error ?? serverError

  return (
    <section className="card" aria-labelledby="pull-heading">
      <h2 id="pull-heading">Penarikan baru</h2>
      <p className="card-desc">
        Pilih node sumber dari grid, pilih path di sumber. File hasil otomatis memakai nama file sumber.
      </p>
      <form onSubmit={submit}>
        <div className="form-grid">
          <div className="span-all">
            <NodeGrid
              endpoints={endpoints}
              states={states}
              selected={selected}
              addingNew={addingNew}
              custom={custom}
              placeholder={DEFAULT_FROM}
              onSelect={(addr) => {
                setSelected(addr)
                setAddingNew(false)
              }}
              onDelete={(addr) => {
                const next = removeEntry('endpoints', addr)
                setEndpoints(next)
                if (selected === addr) {
                  setSelected(next[0] ?? '')
                  setAddingNew(next.length === 0)
                }
              }}
              onAddNew={() => setAddingNew(true)}
              onCustomChange={setCustom}
              onRefresh={() => setNonce((n) => n + 1)}
            />
          </div>
          <div>
            <label className="field" htmlFor="src">
              <span>Path di sumber</span>
              {addingPath || paths.length === 0 ? (
                <input
                  id="src"
                  value={customPath}
                  onChange={(e) => setCustomPath(e.target.value)}
                  placeholder="testdata/dev.dd (seed make dev)"
                  autoComplete="off"
                />
              ) : (
                <select id="src" value={selectedPath} onChange={(e) => pickPath(e.target.value)}>
                  {paths.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                  <option value={NEW_SENTINEL}>+ Path baru…</option>
                </select>
              )}
            </label>
            {addingPath ? (
              paths.length > 0 ? (
                <button type="button" className="link-btn" onClick={() => setAddingPath(false)}>
                  ← Pilih dari tersimpan
                </button>
              ) : (
                <p className="field-hint">Path pertama — otomatis tersimpan setelah dipakai.</p>
              )
            ) : (
              <p className="field-hint">Path tersimpan otomatis setelah dipakai.</p>
            )}
          </div>
          <div className="field">
            <span>File hasil (otomatis)</span>
            <p className="out-value mono">{outName === '' ? '—' : outName}</p>
          </div>
        </div>
        {message && (
          <p className="form-error" role="alert">
            {message}
          </p>
        )}
        <div className="form-actions">
          <button type="button" className="btn btn-secondary" disabled={!canPreview} onClick={() => setPreviewOpen(true)}>
            Pratinjau isi
          </button>
          <button type="submit" className="btn btn-primary pull-submit" disabled={busy}>
            {busy ? 'Menarik…' : 'Mulai Penarikan'}
          </button>
        </div>
      </form>
      {previewOpen && (
        <HexModal from={effectiveFrom} path={effectiveSrc.trim()} onClose={() => setPreviewOpen(false)} />
      )}
    </section>
  )
}
