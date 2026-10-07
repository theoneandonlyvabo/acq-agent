// Grid node sumber tersimpan: baris 40px seragam, maks 4 tampil + scroll.
// Alamat ellipsis satu baris; status berupa dot; tambah-baru sebaris.
export type NodeState = 'checking' | 'online' | 'offline'

interface Props {
  endpoints: string[]
  states: Record<string, NodeState>
  selected: string
  addingNew: boolean
  custom: string
  placeholder: string
  onSelect: (addr: string) => void
  onDelete: (addr: string) => void
  onAddNew: () => void
  onCustomChange: (value: string) => void
  onRefresh: () => void
}

function dotClass(state: NodeState): string {
  if (state === 'online') return 'dot dot-ok'
  if (state === 'offline') return 'dot dot-fail'
  return 'dot dot-wait'
}

function dotText(state: NodeState): string {
  if (state === 'online') return 'Online'
  if (state === 'offline') return 'Offline'
  return 'Memeriksa…'
}

export default function NodeGrid({
  endpoints,
  states,
  selected,
  addingNew,
  custom,
  placeholder,
  onSelect,
  onDelete,
  onAddNew,
  onCustomChange,
  onRefresh,
}: Props) {
  return (
    <div>
      <div className="grid-head">
        <span id="node-label">Alamat sumber</span>
        <button type="button" className="link-btn" onClick={onRefresh}>
          Periksa ulang
        </button>
      </div>
      <div className="node-grid" role="listbox" aria-labelledby="node-label">
        {endpoints.map((ep) => {
          const active = !addingNew && ep === selected
          const state = states[ep] ?? 'checking'
          return (
            <div key={ep} className={active ? 'node-row selected' : 'node-row'}>
              <button
                type="button"
                role="option"
                aria-selected={active}
                title={ep}
                className="node-pick"
                onClick={() => onSelect(ep)}
              >
                <span className="radio" aria-hidden="true" />
                <span className="mono node-addr">{ep}</span>
                <span className={dotClass(state)} title={dotText(state)} />
              </button>
              <button
                type="button"
                className="node-del"
                aria-label={`Hapus ${ep}`}
                title="Hapus alamat ini"
                onClick={() => onDelete(ep)}
              >
                ×
              </button>
            </div>
          )
        })}
        {addingNew ? (
          <div className="node-row">
            <span className="radio" aria-hidden="true" />
            <input
              autoFocus
              value={custom}
              onChange={(e) => onCustomChange(e.target.value)}
              placeholder={placeholder}
              autoComplete="off"
              aria-label="Alamat sumber baru"
            />
          </div>
        ) : (
          <button type="button" className="node-row node-add" onClick={onAddNew}>
            + Alamat baru…
          </button>
        )}
      </div>
    </div>
  )
}
