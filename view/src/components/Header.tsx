// Bar atas: judul + identitas node dari bridge.
import type { Status } from '../api'

interface Props {
  status: Status | null
}

export default function Header({ status }: Props) {
  return (
    <header className="topbar">
      <div>
        <h1>Monitor Sesi Akuisisi</h1>
        <p className="subtitle">Penarikan file image antar node (peer-to-peer)</p>
      </div>
      {status && (
        <div className="node-chips" role="status" aria-label="Status node">
          <div className="chip" title="Identitas node (CommonName sertifikat)">
            <span className="chip-label">Node</span>
            <span className="chip-value">{status.nodeID || 'anonim'}</span>
          </div>
          <div className="chip" title="Alamat gRPC node ini">
            <span className="chip-label">Alamat</span>
            <span className="chip-value mono">{status.addr}</span>
          </div>
          <span className={status.tls ? 'badge badge-ok' : 'badge badge-warn'}>
            {status.tls ? 'mTLS aktif' : 'Plaintext (dev)'}
          </span>
        </div>
      )}
    </header>
  )
}
