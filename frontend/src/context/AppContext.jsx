import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { useAuth } from './AuthContext.jsx'
import { dateAfter } from '../utils/format.js'
import { getMyApplication } from '../api/driverApplications.js'

const AppContext = createContext(null)

export function AppProvider({ children }) {
  const { user: account, refresh } = useAuth()
  const [application, setApplication] = useState(null)
  const [applicationStatus, setApplicationStatus] = useState('loading')
  const [applicationError, setApplicationError] = useState('')
  const mounted = useRef(false)
  const applicationRequest = useRef(0)
  const accountRef = useRef(account)
  accountRef.current = account
  const [role, updateRole] = useState('client')
  const [search, setSearch] = useState({ from: null, to: null, date: dateAfter(1), seats: '1' })
  const [writtenReviews, setWrittenReviews] = useState([])
  const isVerified = account.roles.includes('driver')
  const car = { brand: application?.car_make || '', model: application?.car_model || '',
    year: application ? String(application.car_year) : '', color: application?.car_color || '',
    plate: application?.car_plate || '', seats: application ? String(application.seats) : '' }
  const user = { id: account.id, name: account.first_name, surname: account.last_name,
    email: account.email, phone: account.phone, roles: account.roles,
    joined: new Date(account.created_at).toLocaleDateString('en', { month: 'long', year: 'numeric' }),
    verification: { status: isVerified ? 'approved' : application?.status || 'none', reason: application?.reject_reason || '' }, car }

  const reloadApplication = useCallback(async (options = {}) => {
    const request = ++applicationRequest.current
    setApplicationError('')
    setApplicationStatus((current) => current === 'error' ? 'loading' : current)
    try {
      const latest = await getMyApplication(options)
      if (!mounted.current || options.signal?.aborted || request !== applicationRequest.current) return latest
      if (latest?.status === 'approved' && !accountRef.current.roles.includes('driver')) await refresh()
      if (mounted.current && !options.signal?.aborted && request === applicationRequest.current) {
        setApplication(latest)
        setApplicationStatus('ready')
      }
      return latest
    } catch (error) {
      if (error.name === 'AbortError') return
      if (mounted.current && request === applicationRequest.current) {
        setApplicationError('We could not check your driver application. Please try again.')
        setApplicationStatus('error')
      }
      throw error
    }
  }, [refresh])

  useEffect(() => {
    mounted.current = true
    const controller = new AbortController()
    reloadApplication({ signal: controller.signal }).catch(() => {})
    return () => { mounted.current = false; controller.abort() }
  }, [reloadApplication])

  useEffect(() => {
    const refreshStatus = () => reloadApplication().catch(() => {})
    const timer = application?.status === 'pending' ? setInterval(refreshStatus, 10000) : null
    window.addEventListener('focus', refreshStatus)
    return () => { clearInterval(timer); window.removeEventListener('focus', refreshStatus) }
  }, [application?.status, reloadApplication])

  function acceptApplication(next) {
    applicationRequest.current++
    setApplication(next)
    setApplicationStatus('ready')
    setApplicationError('')
  }

  function setRole(nextRole) {
    if (nextRole === 'client' || (nextRole === 'driver' && isVerified)) updateRole(nextRole)
  }

  return <AppContext.Provider value={{ user, role: role === 'driver' && !isVerified ? 'client' : role, setRole, isVerified, application, applicationStatus, applicationError, reloadApplication, acceptApplication, search, setSearch, writtenReviews, setWrittenReviews }}>{children}</AppContext.Provider>
}

export function useApp() {
  const value = useContext(AppContext)
  if (!value) throw new Error('useApp must be used inside AppProvider')
  return value
}
