import { BrowserRouter, Link, Navigate, Route, Routes } from 'react-router-dom'
import Layout from './components/Layout.jsx'
import DrivesPage from './pages/DrivesPage.jsx'
import DriveDetailsPage from './pages/DriveDetailsPage.jsx'
import ProfilePage from './pages/ProfilePage.jsx'
import VerificationPage from './pages/VerificationPage.jsx'
import DriverDrivesPage from './pages/DriverDrivesPage.jsx'
import DriverGuard from './components/DriverGuard.jsx'
import ReviewsPage from './pages/ReviewsPage.jsx'
import MapPage from './pages/MapPage.jsx'
import AuthPage from './pages/AuthPage.jsx'
import AdminPage from './pages/AdminPage.jsx'
import AdminApplicationPage from './pages/AdminApplicationPage.jsx'
import RequireAuth from './components/RequireAuth.jsx'
import { AppProvider } from './context/AppContext.jsx'
import { useAuth } from './context/AuthContext.jsx'

function AuthenticatedLayout() {
  const { user } = useAuth()
  return <AppProvider key={user.id}><Layout /></AppProvider>
}

function NotFound() {
  return <section className="page-intro"><span className="eyebrow">A SMALL DETOUR</span><h1>That road ends here.</h1><p>We couldn’t find this page. Your next journey is still waiting.</p><Link className="back-link" to="/drives">Back to drives</Link></section>
}

export default function App() {
  return <BrowserRouter><Routes>
    <Route path="/login" element={<AuthPage key="login" />} />
    <Route path="/register" element={<AuthPage key="register" register />} />
    <Route element={<RequireAuth roles={['client']}><AuthenticatedLayout /></RequireAuth>}>
    <Route index element={<Navigate to="/drives" replace />} />
    <Route path="drives" element={<DrivesPage />} />
    <Route path="drives/:driveId" element={<DriveDetailsPage />} />
    <Route path="profile" element={<ProfilePage />} />
    <Route path="verification" element={<VerificationPage />} />
    <Route path="driver/drives" element={<DriverGuard><DriverDrivesPage /></DriverGuard>} />
    <Route path="reviews" element={<ReviewsPage />} />
    <Route path="map" element={<MapPage />} />
    <Route path="admin" element={<RequireAuth roles={['admin']}><AdminPage /></RequireAuth>} />
    <Route path="admin/driver-applications/:applicationId" element={<RequireAuth roles={['admin']}><AdminApplicationPage /></RequireAuth>} />
    <Route path="*" element={<NotFound />} />
  </Route></Routes></BrowserRouter>
}
