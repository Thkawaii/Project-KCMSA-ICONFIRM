import { useAppNavigate, useAppView } from '../lib/nav.jsx'
import { logout } from '../api/auth.js'
import { ArrowRightStartOnRectangleIcon } from './icons.jsx'
import WHAlertBell from './WHAlertBell.jsx'

// ป้ายชื่อ role ที่อ่านง่ายบน topbar (เดิมหน้าคลังส่งมาว่า "Warehouse" เหมือนกันหมด
// จึงแยก WH Manager / WH User ให้ชัดจาก role จริงใน localStorage)
const ROLE_LABELS = {
  WH_MANAGER: 'WH Manager',
  WH: 'WH User',
}

export default function AppShell({ navItems, roleLabel, children }) {
  const navigate = useAppNavigate()
  const currentView = useAppView()

  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase()

  // กรองเมนูตามสิทธิ์: เมนูที่ระบุ roles ไว้จะโชว์เฉพาะ role ที่ตรง (ไม่ระบุ = โชว์ทุก role)
  const visibleNav = (navItems || []).filter((item) => !item.roles || item.roles.includes(role))

  const shownLabel = ROLE_LABELS[role] || roleLabel || 'User'
  const displayName = shownLabel
  const initial = (shownLabel || 'U').trim().charAt(0).toUpperCase() || 'U'

  // กระดิ่งเตือนอายุใบอนุญาต (นำเข้า+ส่งออก รวมเป็นอันเดียว) แสดงเฉพาะ WH Manager
  const showLicenseBell = role === 'WH_MANAGER'

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
          {showLicenseBell && <WHAlertBell />}
          <div className="shell-user" title={shownLabel}>
            <span className="shell-avatar">{initial}</span>
            <span className="shell-user-info">
              <span className="shell-user-name">{displayName}</span>
              <span className="shell-user-role">{shownLabel}</span>
            </span>
          </div>
          <button className="shell-logout-btn" onClick={handleLogout}>
            <ArrowRightStartOnRectangleIcon className="size-4" />
            <span>Log out</span>
          </button>
        </div>
      </header>

      {visibleNav.length > 1 && (
        <nav className="shell-subnav" aria-label="เมนูภายในระบบ">
          {visibleNav.map((item) => (
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