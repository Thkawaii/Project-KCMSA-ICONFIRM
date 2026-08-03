import { useAppNavigate, useAppView } from '../lib/nav.jsx'
import { logout } from '../api/auth.js'
import { ArrowRightStartOnRectangleIcon } from './icons.jsx'
import LicenseAlertBell from './LicenseAlertBell.jsx'

export default function AppShell({ navItems, roleLabel, children }) {
  const navigate = useAppNavigate()
  const currentView = useAppView()

  const displayName = `${roleLabel} User`
  const initial = (roleLabel || 'U').trim().charAt(0).toUpperCase() || 'U'

  // กระดิ่งเตือนอายุใบอนุญาตแสดงเฉพาะ role WH (คนเดียวที่เข้าถึง /import-license ได้)
  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase()
  const showLicenseBell = role === 'WH'

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="shell">
      <header className="shell-topbar">
        <div className="brand-row">
          <span className="brand-badge">KOBELCO</span>
          <h1 className="brand-title-sm">I-CONFIRMATION</h1>
        </div>

        <div className="shell-topbar-right">
          {showLicenseBell && <LicenseAlertBell />}
          <div className="shell-user" title={roleLabel}>
            <span className="shell-avatar">{initial}</span>
            <span className="shell-user-info">
              <span className="shell-user-name">{displayName}</span>
              <span className="shell-user-role">{roleLabel}</span>
            </span>
          </div>
          <button className="shell-logout-btn" onClick={handleLogout}>
            <ArrowRightStartOnRectangleIcon className="size-4" />
            <span>Log out</span>
          </button>
        </div>
      </header>

      {navItems && navItems.length > 1 && (
        <nav className="shell-subnav" aria-label="เมนูภายในระบบ">
          {navItems.map((item) => (
            <button
              key={item.to}
              type="button"
              onClick={() => navigate(item.to)}
              className={
                'shell-subnav-item' + (currentView === item.to ? ' shell-subnav-item-active' : '')
              }
            >
              <span className="shell-subnav-icon">{item.icon}</span>
              {item.label}
            </button>
          ))}
        </nav>
      )}

      <main className="shell-main">{children}</main>
    </div>
  )
}