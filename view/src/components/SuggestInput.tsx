// Input teks dengan saran dropdown dari riwayat (elemen datalist bawaan).
// Ketik manual selalu bisa; daftar kosong berarti mulai dari nol.
interface Props {
  id: string
  label: string
  value: string
  placeholder: string
  suggestions: string[]
  onChange: (value: string) => void
}

export default function SuggestInput({ id, label, value, placeholder, suggestions, onChange }: Props) {
  return (
    <label className="field" htmlFor={id}>
      <span>{label}</span>
      <input
        id={id}
        value={value}
        list={`${id}-riwayat`}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete="off"
      />
      <datalist id={`${id}-riwayat`}>
        {suggestions.map((s) => (
          <option key={s} value={s} />
        ))}
      </datalist>
    </label>
  )
}
