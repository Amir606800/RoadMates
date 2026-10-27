import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useApp } from '../context/AppContext.jsx'
import { useAuth } from '../context/AuthContext.jsx'
import Avatar from '../components/Avatar.jsx'
import Icon from '../components/Icon.jsx'
import VerificationStatus from '../components/VerificationStatus.jsx'
import AdSlot from '../components/AdSlot.jsx'
import { appVersion, appCommit } from '../appVersion.js'

export default function ProfilePage() {
  const { user, role, setRole, isVerified, applicationStatus, applicationError, reloadApplication } = useApp()
  const { logout } = useAuth()
  const [message, setMessage] = useState('')
  const [loggingOut, setLoggingOut] = useState(false)
  const fullName = `${user.name} ${user.surname}`
  async function signOut() {
    if (loggingOut) return
    setLoggingOut(true); setMessage('')
    try { await logout() }
    catch { setMessage('We could not sign you out. Check your connection and try again.') }
    finally { setLoggingOut(false) }
  }
  return <>
    <section className="page-intro"><span className="eyebrow">YOUR CORNER OF THE ROAD</span><h1>Hey, {user.name}.</h1><p>Good journeys start with good people.</p></section>
    <section className="card profile-card"><Avatar name={fullName} large /><h2>{fullName}</h2><p className="text-sm muted">Member since {user.joined}</p><div className="profile-contact"><span><Icon name="phone" size={16} />{user.phone || 'No contact phone set'}</span><span>{user.email}</span></div><div className="account-roles">{user.roles.map((item) => <span className="badge" key={item}>{item}</span>)}</div></section>
    <div className="section-heading"><h2>How are you travelling?</h2></div>
    <div className="role-switch" role="group" aria-label="Travel role"><button className={role === 'client' ? 'selected' : ''} aria-pressed={role === 'client'} onClick={() => setRole('client')}><Icon name="users" size={18} />Client</button>{isVerified ? <button className={role === 'driver' ? 'selected' : ''} aria-pressed={role === 'driver'} onClick={() => setRole('driver')}><Icon name="car" size={18} />Driver</button> : <Link to="/verification"><Icon name="lock" size={16} />Become a driver</Link>}</div>
    <p className="role-hint">{role === 'driver' ? 'Publish drives and welcome your next road mates.' : 'Find a seat, meet your driver, and enjoy the journey.'}</p>
    <div className="stack">
      {applicationStatus === 'loading' ? <p className="text-sm muted" role="status">Checking driver verification…</p> : applicationStatus === 'error' ? <div className="card stack"><p className="field-error" role="alert">{applicationError}</p><button className="button button-secondary" onClick={() => reloadApplication().catch(() => {})}>Check again</button></div> : <VerificationStatus verification={user.verification} />}
      <Link className="settings-row card" to={isVerified ? '/driver/drives' : '/verification'} onClick={() => { if (isVerified) setRole('driver') }}><span className="settings-icon"><Icon name="shield" /></span><div><strong>{isVerified ? 'Manage your drives' : 'Driver verification'}</strong><span>{isVerified ? 'Create your next shared journey' : 'A few details, a world of journeys'}</span></div><Icon name="chevron" size={16} /></Link>
      {user.roles.includes('admin') && <Link className="settings-row card" to="/admin"><span className="settings-icon"><Icon name="shield" /></span><div><strong>Administration</strong><span>Administrator access</span></div><Icon name="chevron" size={16} /></Link>}
    </div>
    <div className="section-heading"><h2>Settings</h2></div>
    <div className="card settings-list"><button className="settings-row" onClick={() => setMessage('English is selected. More languages will be available in a future update.')}><span className="settings-icon"><Icon name="globe" /></span><div><strong>Language</strong><span>English</span></div><Icon name="chevron" size={16} /></button><button className="settings-row" disabled={loggingOut} onClick={signOut}><span className="settings-icon"><Icon name="logout" /></span><div><strong>{loggingOut ? 'Signing out…' : 'Log out'}</strong><span>End this session</span></div><Icon name="chevron" size={16} /></button></div>
    {message && <p className="settings-message text-sm muted" role="status">{message}</p>}
    <AdSlot placement="profile-bottom" />
    <p className="app-version">RoadMates v{appVersion} ({appCommit.slice(0, 7)})</p>
  </>
}
