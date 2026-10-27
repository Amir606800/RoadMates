export default function Avatar({ name, tone = 'sage', large = false }) {
  const initials = name.split(' ').filter(Boolean).slice(0, 2).map((part) => part[0]).join('')
  return <span className={`avatar ${tone} ${large ? 'large' : ''}`} aria-hidden="true">{initials}</span>
}
