import { useEffect, useMemo, useRef, useState } from 'react'
import { getPartChecks, scanPartCheck } from '../api/partcheck.js'
import { scanStep, scanSelect, scanLoading, scanSuccessToast, scanErrorAlert } from '../lib/scanPopup.js'
import AppShell from '../components/AppShell.jsx'

// รูปบาร์โค้ดอ้างอิงของแต่ละพาร์ท (Vite จะ bundle ให้อัตโนมัติ)
import bcItcPn from '../assets/barcodes/IT_Controller_PN_.gif'
import bcItcSn from '../assets/barcodes/IT_Controller_SN_.gif'
import bcSwingSn from '../assets/barcodes/Swing_Motor__SN_.gif'
import bcPumpSn from '../assets/barcodes/Pump_Assy_HYD__SN_.gif'
import bcMotorSn from '../assets/barcodes/Motor_Propel__SN_.gif'
import bcValveSn from '../assets/barcodes/Control_Valve__SN_.gif'

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

// การ์ดบาร์โค้ดที่โชว์บนหน้า Part Confirmation (ตามรูป label จริง)
// กด Scan บนการ์ดไหน = เริ่ม flow สแกนของ partType นั้น
const BARCODE_CARDS = [
  { partType: 'ITC', title: 'IT Controller P/N', img: bcItcPn, kind: 'P/N' },
  { partType: 'ITC', title: 'IT Controller S/N', img: bcItcSn, kind: 'S/N' },
  { partType: 'SM', title: 'Swing Motor S/N', img: bcSwingSn, kind: 'S/N' },
  { partType: 'PH', title: 'Pump Assy HYD S/N', img: bcPumpSn, kind: 'S/N' },
  { partType: 'MP', title: 'Motor Propel S/N', img: bcMotorSn, kind: 'S/N' },
  { partType: 'CV', title: 'Control Valve S/N', img: bcValveSn, kind: 'S/N' },
]

export default function WHPartConfirmationPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  // เก็บผลสแกนล่าสุดลง State (ไว้โชว์บนหน้า)
  const [lastScan, setLastScan] = useState(null)

  const [dateTab, setDateTab] = useState('all')
  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  const [detailRow, setDetailRow] = useState(null)

  // busyRef = true ระหว่างที่ flow สแกนกำลังทำงาน (กันตัวดักสแกนเนอร์ยิงซ้อน)
  const busyRef = useRef(false)
  // เก็บฟังก์ชันจัดการเมื่อสแกนเนอร์ยิง (อัปเดตทุก render กัน closure ค้าง)
  const fireRef = useRef(() => {})

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
    setPage(1)
  }, [dateTab, search, pageSize])

  // ===== SCAN FLOW (SweetAlert) =====
  // สแกนเนอร์ยิงบาร์โค้ดพาร์ท -> popup สแกน TAG เครื่อง -> [P/N] -> S/N -> บันทึกลง API
  async function runScanFlow(partTypeCode) {
    if (!partTypeCode || busyRef.current) return
    const part = PART_TYPES.find((t) => t.code === partTypeCode)
    if (!part) return
    const partLabel = part.label
    const needsPN = Boolean(part.needsPN)

    busyRef.current = true
    try {
      // 1) สแกน TAG เครื่อง (ต้องขึ้นต้นด้วย MC-)
      const machineTag = await scanStep({
        title: `สแกน TAG เครื่อง — ${partLabel}`,
        html: `<div class="scan-popup-hint">${part.icon} <b>${partLabel}</b> · ยิงบาร์โค้ด TAG เครื่อง (ขึ้นต้นด้วย <b>MC-</b>)</div>`,
        placeholder: 'MC-...',
        validate: (v) =>
          /^MC-/i.test(v) ? null : 'รูปแบบ TAG เครื่องไม่ถูกต้อง ต้องขึ้นต้นด้วย MC-',
      })
      if (!machineTag) return // ยกเลิก

      // 2) สแกน P/N (เฉพาะพาร์ทที่ต้องมี P/N เช่น IT Controller)
      let pn = ''
      if (needsPN) {
        pn = await scanStep({
          title: `สแกน P/N — ${partLabel}`,
          html: `<div class="scan-popup-hint">TAG เครื่อง: <b>${machineTag}</b><br/>ยิงบาร์โค้ด <b>P/N</b> ของ ${partLabel}</div>`,
        })
        if (!pn) return
      }

      // 3) สแกน S/N (ขั้นสุดท้าย -> บันทึก)
      const sn = await scanStep({
        title: `สแกน S/N — ${partLabel}`,
        html: `<div class="scan-popup-hint">TAG เครื่อง: <b>${machineTag}</b>${
          needsPN ? `<br/>P/N: <b>${pn}</b>` : ''
        }<br/>ยิงบาร์โค้ด <b>S/N</b> ของ ${partLabel}</div>`,
        confirmText: 'บันทึก',
      })
      if (!sn) return

      // 4) ส่งขึ้น API
      scanLoading('กำลังบันทึก...')
      try {
        const created = await scanPartCheck({
          machineTag,
          partType: partTypeCode,
          pn: needsPN ? pn : '',
          sn,
        })
        // เก็บผลลง State ด้วย
        setLastScan({
          machineTag: created.Tag || machineTag,
          partType: created.PartType || partTypeCode,
          pn: needsPN ? pn : '',
          sn,
          at: created.CheckedDatetime || new Date().toISOString(),
        })
        await scanSuccessToast(`บันทึกแล้ว: ${tagLabel(created.PartType)} — ${created.Tag}`)
        await loadRows()
      } catch (err) {
        await scanErrorAlert(err.message || 'บันทึกไม่สำเร็จ')
      }
    } finally {
      busyRef.current = false
    }
  }

  // ระบุชนิดพาร์ทจากข้อความบาร์โค้ดที่ยิงมา (คืน code หรือ '' ถ้าไม่รู้จัก)
  function detectPartType(raw) {
    const s = (raw || '').toUpperCase()
    if (s.includes('IT CONTROLLER') || s.includes('ITC')) return 'ITC'
    if (s.includes('SWING')) return 'SM'
    if (s.includes('PROPEL')) return 'MP'
    if (s.includes('PUMP') || s.includes('HYD')) return 'PH'
    if (s.includes('CONTROL VALVE') || s.includes('VALVE')) return 'CV'
    return ''
  }

  // เมื่อสแกนเนอร์ยิง 1 ครั้ง: รู้พาร์ท -> เปิด flow, ไม่รู้ -> ให้เลือกพาร์ทเอง
  async function handleScannerFire(code) {
    if (busyRef.current) return
    let partType = detectPartType(code)
    if (!partType) {
      busyRef.current = true
      try {
        partType = await scanSelect({
          title: 'เลือกชนิดพาร์ทที่จะยืนยัน',
          html: `<div class="scan-popup-hint">บาร์โค้ดที่ยิง: <b>${code}</b><br/>ระบบระบุชนิดพาร์ทไม่ได้ กรุณาเลือกเอง</div>`,
          options: PART_TYPES.map((t) => ({ value: t.code, label: `${t.icon} ${t.label}` })),
        })
      } finally {
        busyRef.current = false
      }
      if (!partType) return
    }
    runScanFlow(partType)
  }

  // อัปเดต handler ล่าสุดไว้ใน ref (ให้ listener เรียกใช้ค่าปัจจุบันเสมอ)
  fireRef.current = handleScannerFire

  // ตัวดักสัญญาณเครื่องสแกนเนอร์ระดับหน้าเว็บ:
  // สแกนเนอร์ = คีย์บอร์ดที่พิมพ์เร็วมากแล้วปิดท้ายด้วย Enter -> เด้ง popup ให้เอง
  useEffect(() => {
    let buffer = ''
    let lastTime = 0
    function onKeydown(e) {
      // ข้ามถ้ากำลังพิมพ์ในช่องกรอก หรือ popup กำลังเปิดอยู่
      const tag = (e.target?.tagName || '').toLowerCase()
      if (tag === 'input' || tag === 'textarea' || tag === 'select') return
      if (busyRef.current) return

      const now = Date.now()
      if (now - lastTime > 80) buffer = '' // เว้นเกิน 80ms = คนพิมพ์ ไม่ใช่สแกนเนอร์
      lastTime = now

      if (e.key === 'Enter') {
        const code = buffer.trim()
        buffer = ''
        if (code.length >= 2) fireRef.current(code)
        return
      }
      if (e.key && e.key.length === 1) buffer += e.key
    }
    window.addEventListener('keydown', onKeydown)
    return () => window.removeEventListener('keydown', onKeydown)
  }, [])

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

  return (
    <AppShell navItems={navItems} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Part Confirmation</h2>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="pc-barcode-grid">
        {BARCODE_CARDS.map((card) => (
          <div className="pc-barcode-card" key={card.title}>
            <div className="pc-barcode-title">{card.title}</div>
            <div className="pc-barcode-box">
              <img className="pc-barcode-img" src={card.img} alt={card.title} />
            </div>
          </div>
        ))}
      </div>

      {lastScan && (
        <p className="upload-card-msg-ok" style={{ fontWeight: 700, marginTop: 4 }}>
          ล่าสุด: {tagLabel(lastScan.partType)} — TAG {lastScan.machineTag}
          {lastScan.pn ? ` · P/N ${lastScan.pn}` : ''} · S/N {lastScan.sn}
        </p>
      )}

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
    </AppShell>
  )
}