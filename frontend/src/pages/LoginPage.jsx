import { useState } from 'react'
import { useAppNavigate } from '../lib/nav.jsx'
import { login } from '../api/auth.js'
import { homeRouteForRole } from '../lib/roleRoutes.js'
import { toastSuccess } from '../lib/toast.js'
import {
  EyeIcon,
  EyeSlashIcon,
  LockClosedIcon,
  ShieldCheckIcon,
} from '../components/icons.jsx'

// ใช้ homeRouteForRole ร่วมกับ main.jsx (แหล่งความจริงเดียว ดู lib/roleRoutes.js)
const resolveRoute = homeRouteForRole

export default function LoginPage() {
  const navigate = useAppNavigate()
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
      toastSuccess('Signed in successfully')
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
          <ShieldCheckIcon className="shield-icon size-[26px]" />
          <h1 className="brand-title">I-CONFIRMATION</h1>
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
              name="username"
              type="text"
              autoComplete="username"
              placeholder=""
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
                {showPassword ? <EyeSlashIcon className="size-[18px]" /> : <EyeIcon className="size-[18px]" />}
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
              <LockClosedIcon className="size-4" />
            )}
            {loading ? 'Logging in…' : 'Log in'}
          </button>
        </form>
      </div>
    </div>
  )
}