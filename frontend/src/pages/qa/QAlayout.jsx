import { useEffect, useMemo, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { getQaQueue, confirmQaResult, getAuditLog } from '../../api/qa.js'
import { logout } from '../../api/auth.js'

const navItems = [
  { to: 'dashboard', label: 'Dashboard', icon: '▦' },
  { to: 'production-orders', label: 'Production Orders', icon: '📋' },
  { to: 'scan-validate', label: 'Scan & Validate', icon: '⇄' },
  { to: 'validation-results', label: 'Validation Results', icon: '✓' },
  { to: 'master-data', label: 'Master Data', icon: '🗄' },
  { to: 'traceability', label: 'Traceability', icon: '🔗' },
  { to: 'quality-alerts', label: 'Quality Alerts', icon: '⚠' },
  { to: 'part-comparison', label: 'Part Comparison', icon: '⇅' },
]

// ใช้ map part_no -> ชื่อ component เพื่อโชว์ใน Recent Validation Activity ของ Dashboard
// TODO: ย้ายไปดึงจาก /master-data แทน hardcode ถ้า part_no ชุดจริงมีเพิ่มบ่อย
export const componentNameByPartNo = {
  'CV001': 'Control Valve',
  'YN02P00133F2G1': 'IT Controller',
}

export default function QALayout() {
  const navigate = useNavigate()
  const [queue, setQueue] = useState([])
  const [auditLog, setAuditLog] = useState([])
  const [remarkDraft, setRemarkDraft] = useState({})
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  async function loadData() {
    setLoading(true)
    setLoadError('')
    try {
      const [qaRows, logs] = await Promise.all([getQaQueue(), getAuditLog()])
      setQueue(qaRows || [])
      setAuditLog(
        (logs || []).map((log) => ({
          ...log,
          action_date: new Date(log.ActionDatetime),
        }))
      )
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูล QA ไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  function isMismatch(row) {
    return row.ExpectedPartNo !== row.ActualPartNo || row.ExpectedSpec !== row.ActualSpec
  }

  async function confirmResult(row, result) {
    const remark = remarkDraft[row.ID] ?? ''

    try {
      await confirmQaResult(row.ID, result, remark)
      await loadData()
    } catch (err) {
      setLoadError(err.message || 'ยืนยันผลไม่สำเร็จ')
    }
  }

  const outletContext = useMemo(
    () => ({ queue, auditLog, remarkDraft, setRemarkDraft, isMismatch, confirmResult, loading, loadError }),
    [queue, auditLog, remarkDraft, loading, loadError]
  )

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="qa-shell">
      <aside className="qa-sidebar">
        <div className="qa-sidebar-brand">
          <span className="brand-badge">KOBELCO</span>
          <div>
            <h1 className="qa-sidebar-title">I-CONFIRM</h1>
            <p className="qa-sidebar-subtitle">Traceability System</p>
          </div>
        </div>

        <nav className="qa-nav">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) => 'qa-nav-item' + (isActive ? ' qa-nav-item-active' : '')}
            >
              <span className="qa-nav-icon">{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>

        <button className="qa-logout-btn" onClick={handleLogout}>
          Logout
        </button>
      </aside>

      <div className="qa-content">
        <Outlet context={outletContext} />
      </div>
    </div>
  )
}