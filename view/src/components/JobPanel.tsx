// Panel pekerjaan: status, byte, dan galat bila ada.
import type { Job } from '../api'
import { friendlyError } from '../api'

const stateText: Record<Job['state'], string> = {
  running: 'Berjalan',
  done: 'Selesai',
  failed: 'Gagal',
}

const stateBadge: Record<Job['state'], string> = {
  running: 'badge badge-info',
  done: 'badge badge-ok',
  failed: 'badge badge-fail',
}

export default function JobPanel({ job }: { job: Job | null }) {
  if (!job) {
    return (
      <section className="card" aria-labelledby="job-heading">
        <h2 id="job-heading">Pekerjaan</h2>
        <p className="empty">Belum ada penarikan. Isi form di atas untuk mulai.</p>
      </section>
    )
  }

  return (
    <section className="card" aria-labelledby="job-heading">
      <div className="card-head">
        <h2 id="job-heading">Pekerjaan</h2>
        <span className={stateBadge[job.state]}>{stateText[job.state]}</span>
      </div>
      <dl className="kv">
        <div>
          <dt>ID</dt>
          <dd className="mono">{job.id}</dd>
        </div>
        <div>
          <dt>Sumber</dt>
          <dd className="mono">
            {job.from} · {job.src}
          </dd>
        </div>
        <div>
          <dt>Hasil</dt>
          <dd className="mono">{job.out}</dd>
        </div>
        <div>
          <dt>Byte</dt>
          <dd className="mono">{job.bytes.toLocaleString('id-ID')}</dd>
        </div>
        <div>
          <dt>Mulai</dt>
          <dd className="mono">{job.startedAt}</dd>
        </div>
        {job.state === 'failed' && job.error && (
          <div>
            <dt>Galat</dt>
            <dd>
              <span className="text-fail">{friendlyError(job.error)}</span>
              {friendlyError(job.error) !== job.error && (
                <span className="mono raw-error">{job.error}</span>
              )}
            </dd>
          </div>
        )}
      </dl>
      {job.state === 'running' && (
        <div className="meter" role="progressbar" aria-label="Penarikan berjalan">
          <span />
        </div>
      )}
    </section>
  )
}
