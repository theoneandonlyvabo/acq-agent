// Viewer audit log: N baris terakhir, diparse jadi baris terstruktur.
interface Entry {
  ts: string
  level: string
  event: string
  msg: string
}

function parseLine(line: string): Entry | null {
  try {
    const v = JSON.parse(line) as Partial<Entry>
    if (typeof v.ts === 'string' && typeof v.event === 'string') {
      return { ts: v.ts, level: v.level ?? 'info', event: v.event, msg: v.msg ?? '' }
    }
    return null
  } catch {
    return null
  }
}

function shortTime(ts: string): string {
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return ts.slice(11, 19)
  return d.toLocaleTimeString('id-ID', { hour12: false })
}

interface Props {
  lines: string[]
  onRefresh: () => void
}

export default function AuditLog({ lines, onRefresh }: Props) {
  return (
    <section className="card" aria-labelledby="audit-heading">
      <div className="card-head">
        <h2 id="audit-heading">Audit log</h2>
        <button type="button" className="btn btn-secondary" onClick={onRefresh}>
          Muat ulang
        </button>
      </div>
      {lines.length === 0 ? (
        <p className="empty">Belum ada kejadian tercatat.</p>
      ) : (
        <ol className="audit-list">
          {lines.map((line, i) => {
            const e = parseLine(line)
            if (!e) {
              return (
                <li key={i} className="audit-row">
                  <code className="mono">{line}</code>
                </li>
              )
            }
            return (
              <li key={i} className="audit-row">
                <span className="mono audit-time">{shortTime(e.ts)}</span>
                <span className={`dot ${e.level === 'error' ? 'dot-fail' : 'dot-ok'}`} aria-hidden="true" />
                <code className="mono audit-event">{e.event}</code>
                <span className="audit-msg">{e.msg}</span>
              </li>
            )
          })}
        </ol>
      )}
    </section>
  )
}
