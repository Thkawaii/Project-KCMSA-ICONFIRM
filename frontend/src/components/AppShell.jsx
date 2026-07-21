import { NavLink, useNavigate } from 'react-router-dom'
import { logout } from '../api/auth.js'

export default function AppShell({ navItems, roleLabel, children }) {
  const navigate = useNavigate()

  const name = localStorage.getItem('iconfirm_name') || 'User'
  const initial = name.trim().charAt(0).toUpperCase() || 'U'

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="shell">
      <header className="shell-topbar">
        <div className="brand-row">
          <span className="brand-badge">KOBELCO</span>
          <h1 className="brand-title-sm">I-CONFIRM</h1>
        </div>

        <div className="shell-topbar-right">
          <div className="shell-user" title={roleLabel}>
            <span className="shell-avatar">{initial}</span>
            <span className="shell-user-info">
              <span className="shell-user-name">{name}</span>
              <span className="shell-user-role">{roleLabel}</span>
            </span>
          </div>
          <button className="shell-logout-btn" onClick={handleLogout}>
            <SignOutIcon />
            <span>Log out</span>
          </button>
        </div>
      </header>

      {navItems && navItems.length > 1 && (
        <nav className="shell-subnav" aria-label="เมนูภายในระบบ">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end
              className={({ isActive }) =>
                'shell-subnav-item' + (isActive ? ' shell-subnav-item-active' : '')
              }
            >
              <span className="shell-subnav-icon">{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>
      )}

      <main className="shell-main">{children}</main>
    </div>
  )
}

function SignOutIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4M16 17l5-5-5-5M21 12H9"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}