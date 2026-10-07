// Layar utama: status node, form penarikan, pekerjaan, hash, audit log.
import { useCallback, useEffect, useState } from 'react'
import { fetchAudit, fetchJob, fetchStatus, startPull, type Job, type Status } from './api'
import Header from './components/Header'
import PullForm from './components/PullForm'
import JobPanel from './components/JobPanel'
import HashPanel from './components/HashPanel'
import AuditLog from './components/AuditLog'

export default function App() {
  const [status, setStatus] = useState<Status | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [job, setJob] = useState<Job | null>(null)
  const [starting, setStarting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [lines, setLines] = useState<string[]>([])

  const loadStatus = useCallback(async () => {
    try {
      setStatus(await fetchStatus())
      setStatusError(null)
    } catch {
      setStatus(null)
      setStatusError('Bridge tidak terjangkau di 127.0.0.1:8080. Jalankan: agent serve --http 127.0.0.1:8080')
    }
  }, [])

  const loadAudit = useCallback(async () => {
    try {
      setLines((await fetchAudit(100)).lines)
    } catch {
      // Bridge mati: banner status yang bicara, jangan dobel berisik.
    }
  }, [])

  useEffect(() => {
    loadStatus()
    loadAudit()
  }, [loadStatus, loadAudit])

  const jobId = job?.id ?? null
  const jobState = job?.state ?? null
  useEffect(() => {
    if (jobId === null || jobState !== 'running') return
    const timer = setInterval(async () => {
      try {
        const next = await fetchJob(jobId)
        setJob(next)
        if (next.state !== 'running') loadAudit()
      } catch {
        // Diam sampai interval berikut; banner status menandai bridge mati.
      }
    }, 1000)
    return () => clearInterval(timer)
  }, [jobId, jobState, loadAudit])

  async function handleStart(from: string, src: string, out: string) {
    setStarting(true)
    setFormError(null)
    try {
      const id = await startPull(from, src, out)
      setJob(await fetchJob(id))
      loadAudit()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : 'Gagal memulai penarikan')
    } finally {
      setStarting(false)
    }
  }

  return (
    <div className="page">
      <Header status={status} />
      {statusError && (
        <p className="banner banner-fail" role="alert">
          {statusError}
        </p>
      )}
      <main>
        <PullForm busy={starting || jobState === 'running'} serverError={formError} onStart={handleStart} />
        <JobPanel job={job} />
        <HashPanel job={job} />
        <AuditLog lines={lines} onRefresh={loadAudit} />
      </main>
    </div>
  )
}
