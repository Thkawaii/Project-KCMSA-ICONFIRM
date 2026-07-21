import { useEffect, useMemo, useRef, useState } from 'react'
import { getPartChecks, scanPartCheck } from '../api/partCheck.js'
import AppShell from '../components/AppShell.jsx'

const navItems = [
  { to: '/warehouse', label: 'จ่ายของ (FIFO & S/O)', icon: '📦' },
  { to: '/warehouse/confirm', label: 'Part Confirmation', icon: '✅' },
]

const TAG_TYPES = [
  { code: 'MC', label: 'Machine', icon: '🚜' },
  { code: 'ITC', label: 'IT Controller', icon: '🛰️' },
  { code: 'CV', label: 'Control Valve', icon: '🔧' },
  { code: 'SM', label: 'Swing Motor', icon: '⚙️' },
  { code: 'MP', label: 'Motor Propel', icon: '🚜' },
  { code: 'PH', label: 'Pump Assy HYD', icon: '💧' },
]

function tagLabel(code) {
  return TAG_TYPES.find((t) => t.code === code)?.label || code
}

export default function WHPartConfirmationPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [idleValue, setIdleValue] = useState('')
  const [idleError, setIdleError] = useState('')
  const [idleMsg, setIdleMsg] = useState('')
  const [scanning, setScanning] = useState(false)
  const idleInputRef = useRef(null)

  const [dateTab, setDateTab] = useState('all')
  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  const [detailRow, setDetailRow] = useState(null)

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const data = await getPartChecks()
      setRows(data || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRows()
  }, [])

  useEffect(() => {
    idleInputRef.current?.focus()
  }, [])

  useEffect(() => {
    setPage(1)
  }, [dateTab, search, pageSize])

  async function handleScanSubmit(e) {
    e.preventDefault()
    const tag = idleValue.trim()
    if (!tag) return

    setScanning(true)
    setIdleError('')
    setIdleMsg('')
    try {
      const created = await scanPartCheck(tag)
      setIdleMsg(`บันทึกแล้ว: ${tagLabel(created.TagType)} — ${created.RefNo}`)
      setIdleValue('')
      await loadRows()
      setTimeout(() => setIdleMsg(''), 2500)
    } catch (err) {
      setIdleError(err.message || 'สแกนไม่สำเร็จ')
      setIdleValue('')
    } finally {
      setScanning(false)
      idleInputRef.current?.focus()
    }
  }

  const typeCounts = useMemo(() => {
    const counts = {}
    for (const t of TAG_TYPES) counts[t.code] = rows.filter((r) => r.TagType === t.code).length
    return counts
  }, [rows])

  const filtered = useMemo(() => {
    const now = new Date()
    let list = rows

    if (dateTab !== 'all') {
      list = list.filter((r) => {
        const diffDays = (now - new Date(r.CheckedDatetime)) / (1000 * 60 * 60 * 24)
        if (dateTab === 'day') return diffDays <= 1
        if (dateTab === 'week') return diffDays <= 7
        if (dateTab === 'month') return diffDays <= 31
        return true
      })
    }

    const term = search.trim().toLowerCase()
    if (term) {
      list = list.filter(
        (r) =>
          (r.Tag || '').toLowerCase().includes(term) ||
          (r.RefNo || '').toLowerCase().includes(term) ||
          (r.CheckedBy || '').toLowerCase().includes(term)
      )
    }

    return list
  }, [rows, dateTab, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  return (
    <AppShell navItems={navItems} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Part Confirmation</h2>
          <p className="wh-subtitle">ยิงบาร์โค้ด TAG เพื่อบันทึกการตรวจสอบทันที — ไม่ต้องกรอกอะไรเพิ่ม</p>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="scan-hero" style={{ paddingBottom: 24 }}>
        <div className="scan-hero-graphic" style={{ padding: '26px 40px' }}>
          <BigBarcode />
          <div className="scan-hero-label">SCAN HERE</div>
        </div>
        <p className="scan-hero-hint">
          รูปแบบ TAG: <code>MC-</code>Machine · <code>ITC-</code>IT Controller · <code>CV-</code>Control Valve ·{' '}
          <code>SM-</code>Swing Motor · <code>MP-</code>Motor Propel · <code>PH-</code>Pump Assy HYD
        </p>
        <form onSubmit={handleScanSubmit} className="scan-hero-form">
          <input
            ref={idleInputRef}
            className="scan-hero-input"
            type="text"
            placeholder="รอรับสัญญาณจากเครื่องสแกน..."
            value={idleValue}
            onChange={(e) => setIdleValue(e.target.value)}
            disabled={scanning}
            autoFocus
          />
        </form>
        {scanning && <p className="wh-subtitle">กำลังบันทึก...</p>}
        {idleMsg && <p className="upload-card-msg-ok" style={{ fontWeight: 700 }}>{idleMsg}</p>}
        {idleError && (
          <p className="form-error" role="alert">
            {idleError}
          </p>
        )}
      </div>

      <div className="tsf-stats-row">
        {TAG_TYPES.map((t) => (
          <div className="tsf-stat-card" key={t.code}>
            <span className="tsf-stat-icon">{t.icon}</span>
            <div>
              <div className="tsf-stat-value">{typeCounts[t.code] || 0}</div>
              <div className="tsf-stat-label">{t.label}</div>
            </div>
          </div>
        ))}
      </div>

      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            รายการตรวจสอบ ({filtered.length})
          </h2>
        </div>
        <div className="vr-tabs">
          {[
            { key: 'all', label: 'ทั้งหมด' },
            { key: 'day', label: 'รายวัน' },
            { key: 'week', label: 'รายสัปดาห์' },
            { key: 'month', label: 'รายเดือน' },
          ].map((tab) => (
            <button
              key={tab.key}
              className={'vr-tab' + (dateTab === tab.key ? ' vr-tab-active' : '')}
              onClick={() => setDateTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="tsf-history-toolbar">
        <label className="tsf-history-pagesize">
          <select value={pageSize} onChange={(e) => setPageSize(Number(e.target.value))}>
            <option value={10}>10</option>
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
          entries per page
        </label>
        <input
          className="wh-search"
          type="text"
          placeholder="ค้นหา Tag / รหัส / ผู้ตรวจสอบ"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>TAG</th>
              <th>Part</th>
              <th>Checked By</th>
              <th>วันที่</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              paged.map((r) => (
                <tr key={r.ID}>
                  <td>
                    <strong>{r.TagType}</strong>
                  </td>
                  <td>{tagLabel(r.TagType)}</td>
                  <td>{r.CheckedBy}</td>
                  <td>{new Date(r.CheckedDatetime).toLocaleString('th-TH')}</td>
                  <td>
                    <button className="tsf-action-btn" onClick={() => setDetailRow(r)}>
                      รายละเอียด
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">
                  ยังไม่มีรายการตรวจสอบ
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {!loading && filtered.length > 0 && (
        <div className="tsf-pagination">
          <span className="wh-subtitle" style={{ fontSize: 13 }}>
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of{' '}
            {filtered.length} entries
          </span>
          <div className="tsf-pagination-buttons">
            <button className="wh-modal-cancel" onClick={() => goToPage(1)} disabled={page === 1}>
              «
            </button>
            <button className="wh-modal-cancel" onClick={() => goToPage(page - 1)} disabled={page === 1}>
              ‹
            </button>
            <span className="tsf-pagination-current">
              {page} / {totalPages}
            </span>
            <button className="wh-modal-cancel" onClick={() => goToPage(page + 1)} disabled={page === totalPages}>
              ›
            </button>
            <button className="wh-modal-cancel" onClick={() => goToPage(totalPages)} disabled={page === totalPages}>
              »
            </button>
          </div>
        </div>
      )}

      {detailRow && (
        <div className="wh-modal-overlay" onClick={() => setDetailRow(null)}>
          <div className="wh-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">รายละเอียดการตรวจสอบ</h3>
            <p className="wh-modal-line">
              TAG: <strong>{detailRow.Tag}</strong>
            </p>
            <p className="wh-modal-line">ประเภท: {tagLabel(detailRow.TagType)}</p>
            <p className="wh-modal-line">รหัสอ้างอิง: {detailRow.RefNo}</p>
            <p className="wh-modal-line">ตรวจสอบโดย: {detailRow.CheckedBy}</p>
            <p className="wh-modal-line">
              เวลา: {new Date(detailRow.CheckedDatetime).toLocaleString('th-TH')}
            </p>
            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={() => setDetailRow(null)}>
                ปิด
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  )
}

function BigBarcode() {
  const bars = [3, 1, 2, 1, 3, 2, 1, 1, 2, 3, 1, 2, 1, 3, 1, 1, 2, 1, 3, 2, 1, 3, 1, 2, 1, 1, 3, 2, 1, 2]
  let x = 0
  return (
    <svg viewBox="0 0 320 90" className="scan-hero-barcode">
      {bars.map((w, i) => {
        const bx = x
        x += w * 4 + 3
        return <rect key={i} x={bx} y={0} width={w * 3} height={70} fill="currentColor" />
      })}
    </svg>
  )
}