import { useEffect, useMemo, useRef, useState } from 'react'
import { getPartChecks, scanPartCheck } from '../api/partcheck.js'
import AppShell from '../components/AppShell.jsx'

const navItems = [
  { to: '/warehouse', label: 'จ่ายของ (FIFO & S/O)', icon: '📦' },
  { to: '/warehouse/confirm', label: 'Part Confirmation', icon: '✅' },
]

const TAG_TYPES = [
  { code: 'MC', label: 'Machine', icon: '🚜', needsPN: false },
  { code: 'ITC', label: 'IT Controller', icon: '🛰️', needsPN: true },
  { code: 'CV', label: 'Control Valve', icon: '🔧', needsPN: false },
  { code: 'SM', label: 'Swing Motor', icon: '⚙️', needsPN: false },
  { code: 'MP', label: 'Motor Propel', icon: '🚜', needsPN: false },
  { code: 'PH', label: 'Pump Assy HYD', icon: '💧', needsPN: false },
]

// ชนิดพาร์ทที่เลือกได้ในฟอร์ม (ไม่รวม Machine เพราะ Machine คือ tag ที่ใช้ระบุตัวเครื่อง)
// IT Controller ต้องสแกนทั้ง P/N และ S/N ส่วนพาร์ทอื่นสแกนเฉพาะ S/N
const PART_TYPES = TAG_TYPES.filter((t) => t.code !== 'MC')

function tagLabel(code) {
  return TAG_TYPES.find((t) => t.code === code)?.label || code || '—'
}

export default function WHPartConfirmationPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  // ===== ขั้นที่ 1: เลือกชนิดพาร์ท + สแกน tag เครื่อง =====
  const [selectedPartType, setSelectedPartType] = useState('')
  const [machineValue, setMachineValue] = useState('')
  const [machineError, setMachineError] = useState('')
  const machineInputRef = useRef(null)

  // ===== ขั้นที่ 2 (popup): สแกน P/N แล้วต่อด้วย S/N =====
  const [popupOpen, setPopupOpen] = useState(false)
  const [popupStep, setPopupStep] = useState('pn') // 'pn' | 'sn'
  const [pendingMachineTag, setPendingMachineTag] = useState('')
  const [pnValue, setPnValue] = useState('')
  const [snValue, setSnValue] = useState('')
  const [popupError, setPopupError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [successMsg, setSuccessMsg] = useState('')
  const pnInputRef = useRef(null)
  const snInputRef = useRef(null)

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
    if (!popupOpen) machineInputRef.current?.focus()
  }, [popupOpen, selectedPartType])

  useEffect(() => {
    if (popupOpen && popupStep === 'pn') pnInputRef.current?.focus()
    if (popupOpen && popupStep === 'sn') snInputRef.current?.focus()
  }, [popupOpen, popupStep])

  useEffect(() => {
    setPage(1)
  }, [dateTab, search, pageSize])

  // รอบแรก: สแกน tag เครื่อง -> ถ้ารูปแบบถูกต้อง เด้ง popup ให้สแกนรอบสอง (P/N -> S/N)
  function handleMachineSubmit(e) {
    e.preventDefault()
    const tag = machineValue.trim()
    setMachineError('')
    if (!tag) return

    if (!/^MC-/i.test(tag)) {
      setMachineError('รูปแบบ tag เครื่องไม่ถูกต้อง ต้องขึ้นต้นด้วย MC-')
      setMachineValue('')
      return
    }

    setPendingMachineTag(tag)
    setMachineValue('')
    setPnValue('')
    setSnValue('')
    setPopupError('')
    setPopupStep(needsPN ? 'pn' : 'sn')
    setPopupOpen(true)
  }

  // รอบสอง ส่วนที่ 1: สแกน P/N -> เลื่อนไปช่อง S/N ต่อทันที
  function handlePnSubmit(e) {
    e.preventDefault()
    if (!pnValue.trim()) {
      setPopupError('กรุณาสแกน P/N ก่อน')
      return
    }
    setPopupError('')
    setPopupStep('sn')
  }

  // รอบสอง ส่วนที่ 2: สแกน S/N -> บันทึกการตรวจสอบทั้งหมด
  async function handleSnSubmit(e) {
    e.preventDefault()
    const sn = snValue.trim()
    if (!sn) {
      setPopupError('กรุณาสแกน S/N ก่อน')
      return
    }

    setSubmitting(true)
    setPopupError('')
    try {
      const created = await scanPartCheck({
        machineTag: pendingMachineTag,
        partType: selectedPartType,
        pn: needsPN ? pnValue.trim() : '',
        sn,
      })
      setSuccessMsg(`บันทึกแล้ว: ${tagLabel(created.PartType)} — ${created.Tag}`)
      setTimeout(() => setSuccessMsg(''), 2500)
      closePopup()
      await loadRows()
    } catch (err) {
      setPopupError(err.message || 'บันทึกไม่สำเร็จ')
    } finally {
      setSubmitting(false)
    }
  }

  function closePopup() {
    setPopupOpen(false)
    setPopupStep('pn')
    setPendingMachineTag('')
    setPnValue('')
    setSnValue('')
    setPopupError('')
  }

  const typeCounts = useMemo(() => {
    const counts = {}
    for (const t of PART_TYPES) counts[t.code] = rows.filter((r) => r.PartType === t.code).length
    counts.MC = new Set(rows.map((r) => r.Tag)).size
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
          (r.PN || '').toLowerCase().includes(term) ||
          (r.SN || '').toLowerCase().includes(term) ||
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

  const selectedPartLabel = tagLabel(selectedPartType)
  const selectedPart = PART_TYPES.find((t) => t.code === selectedPartType)
  const needsPN = Boolean(selectedPart?.needsPN)

  return (
    <AppShell navItems={navItems} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Part Confirmation</h2>
          <p className="wh-subtitle">
            เลือกชนิดพาร์ท แล้วยิงบาร์โค้ด TAG เครื่อง — ระบบจะเด้งให้สแกน P/N และ S/N ของพาร์ทต่อทันที
          </p>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="scan-hero" style={{ paddingBottom: 24 }}>
        <div className="pc-parttype-block">
          <p className="pc-parttype-heading">1. เลือกชนิดพาร์ทที่ต้องการยืนยัน</p>
          <div className="pc-parttype-grid">
            {PART_TYPES.map((t) => {
              const active = selectedPartType === t.code
              return (
                <button
                  type="button"
                  key={t.code}
                  className={'pc-parttype-card' + (active ? ' pc-parttype-card-selected' : '')}
                  onClick={() => setSelectedPartType(t.code)}
                  aria-pressed={active}
                >
                  {active && <span className="pc-parttype-check">✓</span>}
                  <span className="pc-parttype-icon">{t.icon}</span>
                  <span className="pc-parttype-label">{t.label}</span>
                  <span className={'pc-parttype-badge ' + (t.needsPN ? 'pc-badge-full' : 'pc-badge-sn')}>
                    {t.needsPN ? 'P/N & S/N' : 'S/N'}
                  </span>
                </button>
              )
            })}
          </div>
        </div>

        {selectedPartType && (
          <>
            <div className="pc-selected-chip">
              <span>
                {selectedPart?.icon} {selectedPartLabel} · สแกน {needsPN ? 'P/N & S/N' : 'S/N'}
              </span>
              <button type="button" onClick={() => setSelectedPartType('')}>
                เปลี่ยน
              </button>
            </div>
            <div className="scan-hero-graphic" style={{ padding: '26px 40px' }}>
              <BigBarcode />
              <div className="scan-hero-label">SCAN HERE</div>
            </div>
            <p className="scan-hero-hint">
              2. สแกน TAG เครื่อง รูปแบบ <code>MC-</code>ตามด้วยรหัสเครื่อง — ระบบจะเด้งให้สแกน{' '}
              {needsPN ? 'P/N และ S/N' : 'S/N'} ของ <strong>{selectedPartLabel}</strong> ต่อทันที
            </p>
            <form onSubmit={handleMachineSubmit} className="scan-hero-form">
              <input
                ref={machineInputRef}
                className="scan-hero-input"
                type="text"
                placeholder="รอรับสัญญาณจากเครื่องสแกน (MC-...)"
                value={machineValue}
                onChange={(e) => setMachineValue(e.target.value)}
                disabled={popupOpen}
                autoFocus
              />
            </form>
            {successMsg && (
              <p className="upload-card-msg-ok" style={{ fontWeight: 700 }}>
                {successMsg}
              </p>
            )}
            {machineError && (
              <p className="form-error" role="alert">
                {machineError}
              </p>
            )}
          </>
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
          placeholder="ค้นหา Tag / P/N / S/N / ผู้ตรวจสอบ"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Machine TAG</th>
              <th>Part</th>
              <th>P/N</th>
              <th>S/N</th>
              <th>Checked By</th>
              <th>วันที่</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={7} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              paged.map((r) => (
                <tr key={r.ID}>
                  <td className="wh-cell-head" data-label="Machine TAG">
                    <strong>{r.Tag}</strong>
                  </td>
                  <td data-label="Part">{tagLabel(r.PartType)}</td>
                  <td data-label="P/N">{r.PN || '—'}</td>
                  <td data-label="S/N">{r.SN || '—'}</td>
                  <td data-label="Checked By">{r.CheckedBy}</td>
                  <td data-label="วันที่">{new Date(r.CheckedDatetime).toLocaleString('th-TH')}</td>
                  <td className="wh-cell-action">
                    <button className="tsf-action-btn" onClick={() => setDetailRow(r)}>
                      รายละเอียด
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={7} className="wh-empty-cell">
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
              Machine TAG: <strong>{detailRow.Tag}</strong>
            </p>
            <p className="wh-modal-line">ชนิดพาร์ท: {tagLabel(detailRow.PartType)}</p>
            <p className="wh-modal-line">P/N: {detailRow.PN || '—'}</p>
            <p className="wh-modal-line">S/N: {detailRow.SN || '—'}</p>
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

      {/* ===== Popup รอบสอง: สแกน P/N แล้วต่อด้วย S/N ===== */}
      {popupOpen && (
        <div className="wh-modal-overlay">
          <div className="wh-modal pc-scan-popup" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">
              สแกน {needsPN ? 'P/N และ S/N' : 'S/N'} — {selectedPartLabel}
            </h3>

            <div className="pc-scan-summary">
              <p className="wh-modal-line">
                ชนิดพาร์ท: <strong>{selectedPart?.icon} {selectedPartLabel}</strong>
              </p>
              <p className="wh-modal-line">
                Machine TAG: <strong>{pendingMachineTag}</strong>
              </p>
            </div>

            <div className="pc-scan-steps">
              {needsPN && (
                <span
                  className={
                    'pc-scan-step' + (popupStep === 'pn' ? ' pc-scan-step-active' : ' pc-scan-step-done')
                  }
                >
                  1. P/N
                </span>
              )}
              <span className={'pc-scan-step' + (popupStep === 'sn' ? ' pc-scan-step-active' : '')}>
                {needsPN ? '2. S/N' : 'S/N'}
              </span>
            </div>

            {popupStep === 'pn' && (
              <form onSubmit={handlePnSubmit}>
                <label className="upload-panel-label" htmlFor="pc-pn-input">
                  สแกน P/N
                </label>
                <input
                  id="pc-pn-input"
                  ref={pnInputRef}
                  className="scan-hero-input"
                  type="text"
                  placeholder="รอรับสัญญาณจากเครื่องสแกน..."
                  value={pnValue}
                  onChange={(e) => setPnValue(e.target.value)}
                />
              </form>
            )}

            {popupStep === 'sn' && (
              <form onSubmit={handleSnSubmit}>
                <p className="wh-modal-line">
                  P/N: <strong>{pnValue}</strong>
                </p>
                <label className="upload-panel-label" htmlFor="pc-sn-input">
                  สแกน S/N
                </label>
                <input
                  id="pc-sn-input"
                  ref={snInputRef}
                  className="scan-hero-input"
                  type="text"
                  placeholder="รอรับสัญญาณจากเครื่องสแกน..."
                  value={snValue}
                  onChange={(e) => setSnValue(e.target.value)}
                  disabled={submitting}
                />
              </form>
            )}

            {popupError && (
              <p className="form-error" role="alert">
                {popupError}
              </p>
            )}

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={closePopup} disabled={submitting}>
                ยกเลิก
              </button>
              {popupStep === 'sn' && (
                <button className="wh-issue-btn" onClick={handleSnSubmit} disabled={submitting}>
                  {submitting ? 'กำลังบันทึก...' : 'ยืนยัน'}
                </button>
              )}
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