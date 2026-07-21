import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppShell from '../../components/AppShell.jsx'
import BarcodeScannerModal from '../../components/Barcodescannermodal.jsx'
import { getMachineSpecs } from '../../api/machineSpec.js'
import { computeStatus, getApprovals, STATUS_LABEL } from '../../api/qaMachineStatus.js'

const navItems = [{ to: '/qa', label: 'ตรวจสอบ QA', icon: '✓' }]

// รวมรายการที่อัปโหลดมาแล้ว (อาจมีหลายแถวต่อ 1 เครื่อง เพราะอัปโหลดทีละหมวด)
// ให้เหลือ 1 แถวต่อเครื่อง โดยใช้แถวที่อัปโหลดล่าสุด
function dedupeByMachine(rows) {
  const byMachine = new Map()
  for (const row of rows) {
    const no = row.MachineNo
    if (!no) continue
    const existing = byMachine.get(no)
    if (!existing) {
      byMachine.set(no, row)
      continue
    }
    const existingDate = existing.UploadDate ? new Date(existing.UploadDate).getTime() : 0
    const rowDate = row.UploadDate ? new Date(row.UploadDate).getTime() : 0
    if (rowDate >= existingDate) byMachine.set(no, row)
  }
  return Array.from(byMachine.values())
}

export default function QAMachineList() {
  const navigate = useNavigate()

  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [showScanner, setShowScanner] = useState(false)

  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  // เก็บ approvals ทั้งหมดไว้ใน state เดียว จะได้ re-render ตารางเมื่อกลับมาจากหน้า detail
  const [approvalsVersion, setApprovalsVersion] = useState(0)

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const data = await getMachineSpecs()
      setRows(dedupeByMachine(data || []))
    } catch (err) {
      setLoadError(err.message || 'โหลดรายการไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRows()
  }, [])

  // รีเฟรชสถานะทุกครั้งที่กลับมาที่หน้านี้ (เช่น ตรวจเสร็จจากหน้า detail แล้วกด "กลับ")
  useEffect(() => {
    function handleFocus() {
      setApprovalsVersion((v) => v + 1)
    }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [])

  const machinesWithStatus = useMemo(() => {
    // eslint-disable-next-line no-unused-expressions
    approvalsVersion
    return rows.map((row) => {
      const approvals = getApprovals(row.MachineNo)
      return { ...row, __status: computeStatus(row, approvals) }
    })
  }, [rows, approvalsVersion])

  const stats = useMemo(() => {
    const total = machinesWithStatus.length
    const pending = machinesWithStatus.filter((m) => m.__status === 'PENDING').length
    const ok = machinesWithStatus.filter((m) => m.__status === 'OK').length
    const fix = machinesWithStatus.filter((m) => m.__status === 'FIX').length
    return { total, pending, ok, fix }
  }, [machinesWithStatus])

  const filtered = useMemo(() => {
    let list = machinesWithStatus

    if (startDate) {
      const start = new Date(startDate).getTime()
      list = list.filter((m) => !m.UploadDate || new Date(m.UploadDate).getTime() >= start)
    }
    if (endDate) {
      const end = new Date(endDate).getTime() + 24 * 60 * 60 * 1000 - 1
      list = list.filter((m) => !m.UploadDate || new Date(m.UploadDate).getTime() <= end)
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase()
      list = list.filter(
        (m) =>
          m.MachineNo?.toLowerCase().includes(q) ||
          m.Spec1?.toLowerCase().includes(q) ||
          m.BaseSpec?.toLowerCase().includes(q)
      )
    }

    return list
  }, [machinesWithStatus, startDate, endDate, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const pageRows = filtered.slice((page - 1) * pageSize, page * pageSize)

  useEffect(() => {
    setPage(1)
  }, [startDate, endDate, search, pageSize])

  function handleScanDetected(decodedText) {
    setShowScanner(false)
    const machineNo = decodedText.trim()
    if (machineNo) navigate(`/qa/machine/${encodeURIComponent(machineNo)}`)
  }

  return (
    <AppShell navItems={navItems} roleLabel="QA">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">QA</h2>
        </div>
      </div>

      <div className="dash-stats-row">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>แมชชีนทั้งหมด</span>
            <span className="dash-stat-icon dash-icon-blue">▦</span>
          </div>
          <div className="dash-stat-value">{stats.total}</div>
          <div className="qa-stat-sub">ทั้งหมด</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>รอตรวจสอบ</span>
            <span className="dash-stat-icon dash-icon-yellow">⏳</span>
          </div>
          <div className="dash-stat-value">{stats.pending}</div>
          <div className="qa-stat-sub">ทั้งหมด</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>สำเร็จ</span>
            <span className="dash-stat-icon dash-icon-green">✓</span>
          </div>
          <div className="dash-stat-value">{stats.ok}</div>
          <div className="qa-stat-sub">ทั้งหมด</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>รอแก้ไข</span>
            <span className="dash-stat-icon dash-icon-red">!</span>
          </div>
          <div className="dash-stat-value">{stats.fix}</div>
          <div className="qa-stat-sub">ทั้งหมด</div>
        </div>
      </div>

      <div className="qa-scan-row">
        <button className="qa-scan-btn" onClick={() => setShowScanner(true)}>
          📷 เริ่มสแกน
        </button>
      </div>

      {showScanner && (
        <BarcodeScannerModal
          title="สแกนแมชชีน"
          onDetected={handleScanDetected}
          onClose={() => setShowScanner(false)}
        />
      )}

      <div className="qa-filter-card">
        <h3 className="qa-filter-title">เลือกช่วงเวลาที่ต้องการให้แสดง</h3>
        <div className="qa-filter-row">
          <div className="qa-filter-field">
            <label>วันเริ่มต้น</label>
            <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
          </div>
          <div className="qa-filter-field">
            <label>วันสิ้นสุด</label>
            <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
          </div>
          <button
            className="qa-scan-btn qa-filter-btn"
            onClick={() => {
              setPage(1)
            }}
          >
            🔽 กรอง
          </button>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="wh-table-card">
        <div className="qa-table-toolbar">
          <div className="qa-table-toolbar-left">
            แสดง{' '}
            <select value={pageSize} onChange={(e) => setPageSize(Number(e.target.value))}>
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
            </select>{' '}
            รายการต่อหน้า
          </div>
          <div className="qa-table-toolbar-right">
            <input
              className="wh-search"
              type="text"
              placeholder="ค้นหา..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>

        <table className="wh-table">
          <thead>
            <tr>
              <th>Machine</th>
              <th>Spec</th>
              <th>Base Spec</th>
              <th>สถานะ</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={4} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              pageRows.map((row) => (
                <tr
                  key={row.MachineNo}
                  className="qa-clickable-row"
                  onClick={() => navigate(`/qa/machine/${encodeURIComponent(row.MachineNo)}`)}
                >
                  <td>
                    <strong>{row.MachineNo}</strong>
                  </td>
                  <td>{row.Spec1 || '—'}</td>
                  <td>{row.BaseSpec || '—'}</td>
                  <td>
                    <span className={`qa-status-badge qa-status-${row.__status.toLowerCase()}`}>
                      {row.__status === 'OK' ? 'OK' : STATUS_LABEL[row.__status]}
                    </span>
                  </td>
                </tr>
              ))}
            {!loading && pageRows.length === 0 && (
              <tr>
                <td colSpan={4} className="wh-empty-cell">
                  ไม่พบรายการ
                </td>
              </tr>
            )}
          </tbody>
        </table>

        {!loading && filtered.length > 0 && (
          <div className="qa-pagination">
            <button disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
              ก่อนหน้า
            </button>
            <span>
              หน้า {page} / {totalPages}
            </span>
            <button disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>
              ถัดไป
            </button>
          </div>
        )}
      </div>
    </AppShell>
  )
}