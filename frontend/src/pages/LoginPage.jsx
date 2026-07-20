import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { login } from '../api/auth.js'

// role_name ที่ backend ส่งกลับมาคือ "QA" / "WH" / "TSF" (ดู seed.go)
function resolveRoute(role) {
  const normalized = (role || '').toUpperCase()

  if (normalized === 'WH') return '/warehouse'
  if (normalized === 'QA') return '/qa'
  if (normalized === 'TSF') return '/tsf'
  if (normalized === 'UPLOAD') return '/upload'

  return '/dashboard'
}

export default function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')

    if (!email && !password) {
      setError('กรุณากรอกชื่อผู้ใช้และรหัสผ่าน')
      return
    }

    if (!email) {
      setError('กรุณากรอกชื่อผู้ใช้')
      return
    }

    if (!password) {
      setError('กรุณากรอกรหัสผ่าน')
      return
    }

    setLoading(true)

    try {
      const data = await login(email.trim(), password)
      navigate(resolveRoute(data.role))
    } catch (err) {
      setError(err.message || 'เข้าสู่ระบบไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-header">
        <div className="brand-row">
          <span className="brand-badge">KOBELCO</span>
          <ShieldCheckIcon />
          <h1 className="brand-title">I-CONFIRM</h1>
        </div>
        <p className="brand-subtitle">Traceability &amp; Validation System</p>
      </div>

      <div className="auth-card">
        <h2 className="card-title">Welcome back</h2>
        <p className="card-subtitle">Log in to your account</p>

        <form onSubmit={handleSubmit} noValidate>
          <div className="field">
            <label htmlFor="email">Username</label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          <div className="field">
            <div className="field-row">
              <label htmlFor="password">Password</label>
            </div>
            <div className="password-input">
              <input
                id="password"
                name="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <button
                type="button"
                className="toggle-visibility"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOffIcon /> : <EyeIcon />}
              </button>
            </div>
          </div>

          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}

          <button type="submit" className="submit-btn" disabled={loading}>
            {loading ? (
              <span className="spinner" aria-hidden="true" />
            ) : (
              <LockIcon />
            )}
            {loading ? 'Logging in…' : 'Log in'}
          </button>
        </form>
      </div>
    </div>
  )
}

function ShieldCheckIcon() {
  return (
    <svg
      className="shield-icon"
      width="26"
      height="26"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M12 2.5l7.5 3v5.6c0 4.86-3.2 8.98-7.5 10.4-4.3-1.42-7.5-5.54-7.5-10.4V5.5l7.5-3z"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <path
        d="M8.7 12.3l2.2 2.2 4.4-4.6"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function LockIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="5" y="11" width="14" height="9" rx="2" stroke="currentColor" strokeWidth="1.8" />
      <path d="M8 11V7.5a4 4 0 118 0V11" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  )
}

function EyeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M3 3l18 18M10.6 10.7a3 3 0 004.2 4.2M6.6 6.7C4.2 8.2 2 12 2 12s3.6 7 10 7c1.8 0 3.4-.4 4.7-1.1M17.4 17.3C19.6 15.9 22 12 22 12s-1.5-3-4.4-5"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}