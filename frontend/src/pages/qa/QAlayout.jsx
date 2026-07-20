import { useEffect, useMemo, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { getQaQueue, confirmQaResult, getAuditLog } from '../../api/qa.js'
import AppShell from '../../components/AppShell.jsx'

const navItems = [
  { to: 'dashboard', label: 'Dashboard', icon: '▦' },
  { to: 'validation-results', label: 'Validation Results', icon: '✓' },
]

// ใช้ map part_no -> ชื่อ component เพื่อโชว์ใน Recent Validation Activity ของ Dashboard
// TODO: ย้ายไปดึงจาก /master-data แทน hardcode ถ้า part_no ชุดจริงมีเพิ่มบ่อย
export const componentNameByPartNo = {
  'CV001': 'Control Valve',
  'YN02P00133F2G1': 'IT Controller',
}

export default function QALayout() {
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

  return (
    <AppShell navItems={navItems} roleLabel="QA">
      <Outlet context={outletContext} />
    </AppShell>
  )
}