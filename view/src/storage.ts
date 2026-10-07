// Riwayat lokal (localStorage): alamat sumber dan path yang pernah dipakai.
// Tanpa backend; dibaca saat form dibuka, ditulis tiap penarikan dimulai.
const MAX_ENTRIES = 10;

function key(name: string): string {
  return `acq-agent:${name}`
}

// loadList membaca daftar tersimpan; rusak/kosong berarti daftar kosong.
export function loadList(name: string): string[] {
  try {
    const raw = localStorage.getItem(key(name))
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((v): v is string => typeof v === 'string' && v !== '');
  } catch {
    return []
  }
}

// saveEntry menaruh nilai paling depan, unik, maksimal MAX_ENTRIES.
export function saveEntry(name: string, value: string): string[] {
  const next = [value, ...loadList(name).filter((v) => v !== value)].slice(0, MAX_ENTRIES)
  try {
    localStorage.setItem(key(name), JSON.stringify(next))
  } catch {
    // Penyimpanan penuh/diblokir: riwayat sekadar tidak tersimpan.
  }
  return next
}
