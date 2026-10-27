import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'
import { AuthProvider } from './context/AuthContext.jsx'
import AppCompatibility from './components/AppCompatibility.jsx'
import './styles.css'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <AuthProvider><AppCompatibility><App /></AppCompatibility></AuthProvider>
  </StrictMode>,
)
