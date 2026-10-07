// Klien HTTP untuk bridge lokal agent (dev only).
// Alamat bridge sama dengan flag --http saat serve.
export const BRIDGE_URL = 'http://127.0.0.1:8080'

export interface Status {
  nodeID: string
  tls: boolean
  addr: string
}

export interface Job {
  id: string
  from: string
  src: string
  out: string
  state: 'running' | 'done' | 'failed'
  bytes: number
  sha256: string
  wantSHA256: string
  error?: string
  startedAt: string
}

export interface AuditResponse {
  lines: string[]
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BRIDGE_URL}${path}`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return (await res.json()) as T
}

export const fetchStatus = () => get<Status>('/api/status')
export const fetchJob = (id: string) => get<Job>(`/api/jobs/${id}`)
export const fetchAudit = (limit = 100) => get<AuditResponse>(`/api/audit?limit=${limit}`)

export async function startPull(from: string, src: string, out: string): Promise<string> {
  const res = await fetch(`${BRIDGE_URL}/api/pull`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ from, src, out }),
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string }
    throw new Error(body.error ?? `HTTP ${res.status}`)
  }
  const data = (await res.json()) as { jobID: string }
  return data.jobID
}

// probeEndpoint mengecek keterjangkauan TCP sebuah alamat via bridge.
export async function probeEndpoint(addr: string): Promise<boolean> {
  try {
    const res = await fetch(`${BRIDGE_URL}/api/probe?addr=${encodeURIComponent(addr)}`)
    if (!res.ok) return false
    const data = (await res.json()) as { online: boolean }
    return data.online === true
  } catch {
    return false
  }
}

export async function probeEndpoints(addrs: string[]): Promise<Record<string, 'online' | 'offline'>> {
  const entries = await Promise.all(
    addrs.map(async (addr): Promise<[string, 'online' | 'offline']> => [
      addr,
      (await probeEndpoint(addr)) ? 'online' : 'offline',
    ]),
  )
  return Object.fromEntries(entries)
}

// friendlyError memetakan galat teknis umum ke bahasa Indonesia.
// Teks asli tetap ditampilkan kecil di bawahnya oleh pemanggil.
export function friendlyError(raw: string): string {
  if (raw.includes('no such file or directory')) {
    return 'File sumber tidak ada di node sumber. Buat dulu dengan scripts/ghostdisk.sh.'
  }
  if (raw.includes('connection refused')) {
    return 'Tidak bisa terhubung ke node sumber. Pastikan serve jalan di alamat itu.'
  }
  return raw
}

export interface PreviewResponse {
  offset: number
  total_bytes: number
  data_base64: string
}

// previewSource mengintip sebagian isi file di node sumber via bridge.
export async function previewSource(from: string, path: string, offset: number, limit: number): Promise<PreviewResponse> {
  const q = `from=${encodeURIComponent(from)}&path=${encodeURIComponent(path)}&offset=${offset}&limit=${limit}`
  const res = await fetch(`${BRIDGE_URL}/api/preview?${q}`)
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string }
    throw new Error(body.error ?? `HTTP ${res.status}`)
  }
  return (await res.json()) as PreviewResponse
}
