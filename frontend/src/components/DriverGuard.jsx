import RequireAuth from './RequireAuth.jsx'

export default function DriverGuard({ children }) {
  return <RequireAuth roles={['driver']}>{children}</RequireAuth>
}
