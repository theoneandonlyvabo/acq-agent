// Panel verifikasi SHA-256 sisi penarik vs sisi sumber.
import type { Job } from '../api'

function badgeFor(job: Job | null): { text: string; className: string } {
  if (!job) return { text: 'Menunggu', className: 'badge' }
  if (job.state === 'running') return { text: 'Berjalan', className: 'badge badge-info' }
  if (job.state === 'failed') return { text: 'Gagal', className: 'badge badge-fail' }
  if (job.sha256 !== '' && job.sha256 === job.wantSHA256) {
    return { text: 'COCOK', className: 'badge badge-ok' }
  }
  return { text: 'TIDAK COCOK', className: 'badge badge-fail' }
}

export default function HashPanel({ job }: { job: Job | null }) {
  const badge = badgeFor(job)
  const ready = job?.state === 'done'

  return (
    <section className="card" aria-labelledby="hash-heading">
      <div className="card-head">
        <h2 id="hash-heading">Verifikasi hash</h2>
        <span className={badge.className}>{badge.text}</span>
      </div>
      {!ready ? (
        <p className="empty">Hash sumber tiba di akhir stream; tampil setelah penarikan selesai.</p>
      ) : (
        <dl className="kv">
          <div>
            <dt>SHA-256 penarik</dt>
            <dd className="mono hash">{job.sha256}</dd>
          </div>
          <div>
            <dt>SHA-256 sumber</dt>
            <dd className="mono hash">{job.wantSHA256}</dd>
          </div>
        </dl>
      )}
    </section>
  )
}
