import Icon from './Icon.jsx'

export default function EmptyState({ title, children, icon = 'search', action }) {
  return <div className="card empty-state"><Icon name={icon} size={30} /><h2>{title}</h2><p>{children}</p>{action}</div>
}
