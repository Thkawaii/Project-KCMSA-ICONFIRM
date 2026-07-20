import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { logout } from '../api/auth.js'

export default function AppShell({ navItems, roleLabel, children }) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)

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
        <button
          className="shell-hamburger"
          onClick={() => setOpen(true)}
          aria-label="Open menu"
        >
          <MenuIcon />
        </button>
      </header>

      <main className="shell-main">{children}</main>

      {open && (
        <div className="shell-overlay" onClick={() => setOpen(false)}>
          <div className="shell-drawer" onClick={(e) => e.stopPropagation()}>
            <div className="shell-drawer-topbar">
              <div className="brand-row">
                <span className="brand-badge">KOBELCO</span>
                <h1 className="brand-title-sm">I-CONFIRM</h1>
              </div>
              <button
                className="shell-close"
                onClick={() => setOpen(false)}
                aria-label="Close menu"
              >
                <CloseIcon />
              </button>
            </div>

            <nav className="shell-nav">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  onClick={() => setOpen(false)}
                  className={({ isActive }) =>
                    'shell-nav-item' + (isActive ? ' shell-nav-item-active' : '')
                  }
                >
                  <span className="shell-nav-icon">{item.icon}</span>
                  {item.label}
                </NavLink>
              ))}
            </nav>

            <div className="shell-profile">
              <div className="shell-avatar">{initial}</div>
              <div className="shell-profile-info">
                <div className="shell-profile-name">{name}</div>
                <div className="shell-profile-role">{roleLabel}</div>
              </div>
            </div>

            <button className="shell-signout" onClick={handleLogout}>
              <SignOutIcon /> Sign Out
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function MenuIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M3 6h18M3 12h18M3 18h18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}

function CloseIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
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