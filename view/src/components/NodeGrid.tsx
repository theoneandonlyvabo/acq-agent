// Grid node sumber tersimpan (maks 4 baris tampil) + status online + tambah baru.
// Pilih baris untuk mengisi alamat; tambah-baru membuka input sebaris.
export type NodeState = 'checking' | 'online' | 'offline'

interface Props {
  endpoints: string[]
  states: Record<string, NodeState>
  selected: string
  addingNew: boolean
  custom: string
  placeholder: string
  onSelect: (addr: string) => void
  onAddNew: () => void
  onCustomChange: (value: string) => void
  onRefresh: () => void
}

function badge(state: NodeState): { text: string; className: string } {
  if (state === 'online') return { text: 'Online', className: 'badge badge-sm badge-ok' }
  if (state === 'offline') return { text: 'Offline', className: 'badge badge-sm badge-fail' }
  return { text: 'Memeriksa…', className: 'badge badge-sm' }
}

export default function NodeGrid({
  endpoints,
  states,
  selected,
  addingNew,
  custom,
  placeholder,
  onSelect,
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
          const b = badge(states[ep] ?? 'checking')
          return (
            <button
              key={ep}
              type="button"
              role="option"
              aria-selected={active}
              className={active ? 'node-row selected' : 'node-row'}
              onClick={() => onSelect(ep)}
            >
              <span className="radio" aria-hidden="true" />
              <span className="mono node-addr">{ep}</span>
              <span className={b.className}>{b.text}</span>
            </button>
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
