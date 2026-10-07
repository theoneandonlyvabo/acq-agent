// Form penarikan baru: grid node sumber, path (saran), file hasil otomatis.
// File hasil = basename path sumber, tampil read-only.
import { useEffect, useState, type FormEvent } from 'react'
import SuggestInput from './SuggestInput'
import NodeGrid, { type NodeState } from './NodeGrid'
import { loadList, saveEntry } from '../storage'
import { probeEndpoints } from '../api'

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
  const [src, setSrc] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [states, setStates] = useState<Record<string, NodeState>>({})
  const [nonce, setNonce] = useState(0)

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

  const outName = basename(src)

  function submit(e: FormEvent) {
    e.preventDefault()
    const cleanFrom = (addingNew ? custom : selected).trim()
    const cleanSrc = src.trim()
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
    onStart(cleanFrom, cleanSrc, cleanOut)
  }

  const message = error ?? serverError

  return (
    <section className="card" aria-labelledby="pull-heading">
      <h2 id="pull-heading">Penarikan baru</h2>
      <p className="card-desc">
        Pilih node sumber dari grid, isi path di sumber. File hasil otomatis memakai nama file sumber.
      </p>
      <form onSubmit={submit}>
        <div className="form-grid">
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
            onAddNew={() => setAddingNew(true)}
            onCustomChange={setCustom}
            onRefresh={() => setNonce((n) => n + 1)}
          />
          <SuggestInput
            id="src"
            label="Path di sumber"
            value={src}
            placeholder="testdata/disk01.dd"
            suggestions={paths}
            onChange={setSrc}
          />
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
        <button type="submit" className="btn btn-primary pull-submit" disabled={busy}>
          {busy ? 'Menarik…' : 'Mulai Penarikan'}
        </button>
      </form>
    </section>
  )
}
