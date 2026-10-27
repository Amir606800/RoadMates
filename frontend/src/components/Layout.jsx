import { useEffect, useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import Icon from './Icon.jsx'
import AdSlot from './AdSlot.jsx'

const tabs = [['Drives', 'car', '/drives'], ['Map', 'map', '/map'], ['Reviews', 'star', '/reviews'], ['Profile', 'user', '/profile']]

export default function Layout() {
  const [notificationsOpen, setNotificationsOpen] = useState(false)
  const { pathname } = useLocation()
  const title = pathname.startsWith('/admin') ? 'Administration' : pathname.includes('verification') ? 'Verification' : pathname.includes('/drives/') ? 'Drive details' : pathname.startsWith('/driver') ? 'My drives' : tabs.find(([, , path]) => pathname.startsWith(path))?.[0] || 'RoadMates'
  useEffect(() => {
    window.scrollTo(0, 0)
    setNotificationsOpen(false)
    document.title = `${title} · RoadMates`
  }, [pathname, title])
  return <div className="app-shell">
    <header className="app-header">
      <Link className="brand" to="/drives"><span className="brand-mark"><Icon name="road" size={21} /></span>RoadMates</Link>
      <div className="header-actions"><span className="header-title">{title}</span><button className="icon-button notification-button" aria-label="Notifications" aria-expanded={notificationsOpen} onClick={() => setNotificationsOpen(!notificationsOpen)}><Icon name="bell" /><span className="notification-dot" /></button></div>
      {notificationsOpen && <section className="notifications card" aria-label="Notifications"><div className="row between"><h2>You're all caught up</h2><button className="icon-button" aria-label="Close notifications" onClick={() => setNotificationsOpen(false)}><Icon name="close" /></button></div><p>Your journey updates will appear here.</p></section>}
    </header>
    <main className="app-main" id="main-content"><Outlet /><AdSlot placement="layout-bottom" /></main>
    <nav className="bottom-nav" aria-label="Main navigation">{tabs.map(([label, icon, path]) => <NavLink to={path} className={({ isActive }) => `nav-item ${isActive || (label === 'Profile' && pathname === '/verification') || (label === 'Drives' && pathname === '/driver/drives') ? 'active' : ''}`} key={label}><span className="nav-icon"><Icon name={icon} size={21} /></span>{label}</NavLink>)}</nav>
  </div>
}
