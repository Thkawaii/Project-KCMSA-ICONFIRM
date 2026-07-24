import { useEffect, useMemo, useRef, useState } from 'react'
import { getPartChecks, scanPartCheck } from '../api/partcheck.js'
import { getImportLicenseItems, getImportLicenseSummary } from '../api/importLicense.js'
import { scanStep, scanSelect, scanLoading, scanSuccessToast, scanErrorAlert } from '../lib/scanPopup.js'
import AppShell from '../components/AppShell.jsx'
import { WH_NAV_ITEMS } from './Importlicensepage.jsx'

// รูปบาร์โค้ดอ้างอิงของแต่ละพาร์ท (Vite จะ bundle ให้อัตโนมัติ)
import bcItcPn from '../assets/barcodes/IT_Controller_PN_.gif'
import bcItcSn from '../assets/barcodes/IT_Controller_SN_.gif'
import bcSwingSn from '../assets/barcodes/Swing_Motor__SN_.gif'
import bcPumpSn from '../assets/barcodes/Pump_Assy_HYD__SN_.gif'
import bcMotorSn from '../assets/barcodes/Motor_Propel__SN_.gif'
import bcValveSn from '../assets/barcodes/Control_Valve__SN_.gif'

const TAG_TYPES = [
  { code: 'MC', label: 'Machine', icon: '🚜', needsPN: false },
  { code: 'ITC', label: 'IT Controller', icon: '🛰️', needsPN: true },
  { code: 'CV', label: 'Control Valve', icon: '🔧', needsPN: false },
  { code: 'SM', label: 'Swing Motor', icon: '⚙️', needsPN: false },
  { code: 'MP', label: 'Motor Propel', icon: '🚜', needsPN: false },
  { code: 'PH', label: 'Pump Assy HYD', icon: '💧', needsPN: false },
]

// ชนิดพาร์ทที่เลือกได้ในฟอร์ม (ไม่รวม Machine เพราะ Machine คือ tag ที่ใช้ระบุตัวเครื่อง)
// IT Controller ต้องสแกนทั้ง P/N และหมายเลขเครื่อง ส่วนพาร์ทอื่นสแกนเฉพาะ S/N
const PART_TYPES = TAG_TYPES.filter((t) => t.code !== 'MC')

function tagLabel(code) {
  return TAG_TYPES.find((t) => t.code === code)?.label || code || '—'
}

// ป้ายผลการเทียบกับบัญชีใบอนุญาตนำเข้า (ค่าตรงกับค่าคงที่ฝั่ง backend)
const MATCH_LABELS = {
  MATCH: { text: '✓ ตรงกับใบอนุญาต', cls: 'il-badge-ok' },
  NOT_FOUND: { text: '✕ ไม่พบในใบอนุญาต', cls: 'il-badge-bad' },
  WRONG_INVOICE: { text: '⚠ คนละอินวอยซ์', cls: 'il-badge-warn' },
  WRONG_PRODNO: { text: '⚠ หมายเลขการผลิตไม่ตรง', cls: 'il-badge-warn' },
  DUPLICATE: { text: '⚠ ยืนยันซ้ำ', cls: 'il-badge-warn' },
  NOT_REQUIRED: { text: '— ไม่ต้องเทียบ', cls: 'il-badge-muted' },
}

function matchBadge(status) {
  const m = MATCH_LABELS[status] || MATCH_LABELS.NOT_REQUIRED
  return <span className={'il-badge ' + m.cls}>{m.text}</span>
}

// การ์ดบาร์โค้ดที่โชว์บนหน้า Part Confirmation (ตามรูป label จริง)
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

  // ── ตารางอ้างอิง: บัญชีใบอนุญาตนำเข้า ────────────────────────────────────
  const [licenseItems, setLicenseItems] = useState([])
  const [lots, setLots] = useState([])
  const [selectedLot, setSelectedLot] = useState('') // 'licenseNo|invoiceNo'
  const [licenseTab, setLicenseTab] = useState('all')
  const [highlightId, setHighlightId] = useState(null)

  // ผลสแกนล่าสุด (ไว้โชว์แถบสรุปบนหน้า)
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
  // อินวอยซ์ที่เลือกอยู่ ณ ตอนสแกน — flow เป็น async เลยต้องอ่านผ่าน ref
  const invoiceRef = useRef('')

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const [checks, items, summary] = await Promise.all([
        getPartChecks(),
        getImportLicenseItems(),
        getImportLicenseSummary(),
      ])
      setRows(checks || [])
      setLicenseItems(items || [])
      setLots(summary || [])
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

  const [licenseNo, invoiceNo] = selectedLot ? selectedLot.split('|') : ['', '']
  invoiceRef.current = invoiceNo

  // ── SCAN FLOW (SweetAlert) ───────────────────────────────────────────────
  // ITC: TAG เครื่อง -> P/N -> หมายเลขเครื่อง -> (หมายเลขการผลิต) -> เทียบ + บันทึก
  // พาร์ทอื่น: TAG เครื่อง -> S/N -> บันทึก (ไม่ต้องเทียบบัญชี)
  async function runScanFlow(partTypeCode) {
    if (!partTypeCode || busyRef.current) return
    const part = PART_TYPES.find((t) => t.code === partTypeCode)
    if (!part) return

    const partLabel = part.label
    const isITC = part.code === 'ITC'
    const needsPN = Boolean(part.needsPN)
    const lotInvoice = invoiceRef.current

    busyRef.current = true
    try {
      // 1) สแกน TAG เครื่อง (ต้องขึ้นต้นด้วย MC-)
      const machineTag = await scanStep({
        title: `สแกน TAG เครื่อง — ${partLabel}`,
        html: `<div class="scan-popup-hint">${part.icon} <b>${partLabel}</b> · ยิงบาร์โค้ด TAG เครื่อง (ขึ้นต้นด้วย <b>MC-</b>)${
          lotInvoice ? `<br/>ล็อตที่กำลังยืนยัน: <b>${lotInvoice}</b>` : ''
        }</div>`,
        placeholder: 'MC-...',
        validate: (v) => (/^MC-/i.test(v) ? null : 'รูปแบบ TAG เครื่องไม่ถูกต้อง ต้องขึ้นต้นด้วย MC-'),
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

      // 3) สแกนหมายเลขเครื่อง (ITC) หรือ S/N (พาร์ทอื่น)
      const sn = await scanStep({
        title: isITC ? 'สแกนหมายเลขเครื่อง (12 หลัก)' : `สแกน S/N — ${partLabel}`,
        html: `<div class="scan-popup-hint">TAG เครื่อง: <b>${machineTag}</b>${
          needsPN ? `<br/>P/N: <b>${pn}</b>` : ''
        }<br/>ยิงบาร์โค้ด <b>${isITC ? 'หมายเลขเครื่อง' : 'S/N'}</b> ของ ${partLabel}${
          isITC ? '<br/>ระบบจะเทียบกับบัญชีใบอนุญาตนำเข้าให้ทันที' : ''
        }</div>`,
        confirmText: isITC ? 'ต่อไป' : 'บันทึก',
      })
      if (!sn) return

      // 4) หมายเลขการผลิต (IMEI) — ไม่บังคับ กด "ข้ามขั้นนี้" ได้
      let productionNo = ''
      if (isITC) {
        productionNo =
          (await scanStep({
            title: 'สแกนหมายเลขการผลิต (15 หลัก)',
            html: `<div class="scan-popup-hint">หมายเลขเครื่อง: <b>${sn}</b><br/>ยิงบาร์โค้ด <b>หมายเลขการผลิต</b> เพื่อตรวจซ้ำอีกชั้น<br/>ถ้าไม่มีให้กด "ข้ามขั้นนี้"</div>`,
            confirmText: 'บันทึก',
            cancelText: 'ข้ามขั้นนี้',
          })) || ''
      }

      // 5) ส่งขึ้น API — backend เทียบกับบัญชีแล้วตอบผลกลับมาในทีเดียว
      scanLoading('กำลังตรวจสอบกับบัญชีใบอนุญาต...')
      try {
        const res = await scanPartCheck({
          machineTag,
          partType: partTypeCode,
          pn: needsPN ? pn : '',
          sn,
          productionNo,
          invoiceNo: lotInvoice,
        })

        const check = res.check || res

        setLastScan({
          machineTag: check.Tag || machineTag,
          partType: check.PartType || partTypeCode,
          pn: needsPN ? pn : '',
          sn,
          productionNo,
          matchStatus: check.MatchStatus,
          message: check.MatchMessage || res.message,
          at: check.CheckedDatetime || new Date().toISOString(),
        })

        // ไฮไลต์แถวในตารางที่เพิ่งจับคู่ได้ ให้เห็นด้วยตาว่าไปโดนแถวไหน
        if (res.item?.ID) {
          setHighlightId(res.item.ID)
          setTimeout(() => setHighlightId(null), 6000)
        }

        if (res.matched) {
          await scanSuccessToast(`ตรงกับบัญชี: ${sn}`)
        } else if (isITC) {
          await scanErrorAlert(check.MatchMessage || res.message || 'ไม่ตรงกับบัญชีใบอนุญาตนำเข้า')
        } else {
          await scanSuccessToast(`บันทึกแล้ว: ${tagLabel(check.PartType)} — ${check.Tag}`)
        }

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

  fireRef.current = handleScannerFire

  // ตัวดักสัญญาณเครื่องสแกนเนอร์ระดับหน้าเว็บ:
  // สแกนเนอร์ = คีย์บอร์ดที่พิมพ์เร็วมากแล้วปิดท้ายด้วย Enter -> เด้ง popup ให้เอง
  useEffect(() => {
    let buffer = ''
    let lastTime = 0
    function onKeydown(e) {
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

  // ── ตารางเทียบ: บัญชีใบอนุญาตของล็อตที่เลือก ─────────────────────────────
  const licenseRows = useMemo(() => {
    let list = licenseItems
    if (selectedLot) {
      list = list.filter((r) => r.LicenseNo === licenseNo && r.InvoiceNo === invoiceNo)
    }
    if (licenseTab === 'pending') list = list.filter((r) => r.ConfirmStatus !== 'CONFIRMED')
    if (licenseTab === 'confirmed') list = list.filter((r) => r.ConfirmStatus === 'CONFIRMED')
    return list
  }, [licenseItems, selectedLot, licenseNo, invoiceNo, licenseTab])

  const licenseCounts = useMemo(() => {
    const scope = selectedLot
      ? licenseItems.filter((r) => r.LicenseNo === licenseNo && r.InvoiceNo === invoiceNo)
      : licenseItems
    return {
      total: scope.length,
      confirmed: scope.filter((r) => r.ConfirmStatus === 'CONFIRMED').length,
      pending: scope.filter((r) => r.ConfirmStatus !== 'CONFIRMED').length,
    }
  }, [licenseItems, selectedLot, licenseNo, invoiceNo])

  // ── ประวัติการสแกน ───────────────────────────────────────────────────────
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

  const mismatchCount = useMemo(
    () => rows.filter((r) => r.PartType === 'ITC' && r.MatchStatus && r.MatchStatus !== 'MATCH').length,
    [rows]
  )

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  return (
    <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Part Confirmation</h2>
          <p className="wh-subtitle">
            สแกนแล้วระบบเทียบกับบัญชีใบอนุญาตนำเข้าให้ทันที — ตรง/ไม่ตรง ขึ้นในตารางด้านล่างเลย
          </p>
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

      {/* ── ผลสแกนล่าสุด ────────────────────────────────────────────────── */}
      {lastScan && (
        <div
          className={
            'il-result-bar' +
            (lastScan.matchStatus === 'MATCH'
              ? ' il-result-ok'
              : lastScan.matchStatus === 'NOT_REQUIRED'
              ? ''
              : ' il-result-bad')
          }
        >
          <div>
            <strong>{tagLabel(lastScan.partType)}</strong> · TAG {lastScan.machineTag} · หมายเลขเครื่อง{' '}
            <span className="il-mono">{lastScan.sn}</span>
            {lastScan.productionNo ? (
              <>
                {' '}
                · หมายเลขการผลิต <span className="il-mono">{lastScan.productionNo}</span>
              </>
            ) : null}
          </div>
          <div className="il-result-msg">
            {matchBadge(lastScan.matchStatus)} {lastScan.message}
          </div>
        </div>
      )}

      {/* ── เลือกล็อตที่จะยืนยัน ────────────────────────────────────────── */}
      {lots.length > 0 ? (
        <div className="il-lot-row">
          <button
            className={'il-lot-chip' + (selectedLot === '' ? ' il-lot-chip-active' : '')}
            onClick={() => setSelectedLot('')}
          >
            ทุกใบอนุญาต
          </button>
          {lots.map((s) => {
            const key = `${s.LicenseNo}|${s.InvoiceNo}`
            const done = s.Confirmed >= s.Total
            return (
              <button
                key={key}
                className={
                  'il-lot-chip' +
                  (selectedLot === key ? ' il-lot-chip-active' : '') +
                  (done ? ' il-lot-chip-done' : '')
                }
                onClick={() => setSelectedLot(key)}
              >
                <strong>Invoice {s.InvoiceNo}</strong>
                <span className="il-lot-chip-sub">
                  {s.LicenseNo} · {s.Confirmed}/{s.Total} ยืนยันแล้ว
                </span>
              </button>
            )
          })}
        </div>
      ) : (
        <p className="wh-subtitle">
          ยังไม่มีบัญชีใบอนุญาตนำเข้าในระบบ — ไปที่เมนู <strong>Import License</strong>{' '}
          เพื่ออัปโหลดไฟล์ Excel ก่อน แล้วค่อยกลับมาสแกน
        </p>
      )}

      {/* ── ตารางเทียบกับบัญชีใบอนุญาต ─────────────────────────────────── */}
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            เทียบกับบัญชีใบอนุญาตนำเข้า ({licenseCounts.confirmed}/{licenseCounts.total})
          </h2>
        </div>
        <div className="vr-tabs">
          {[
            { key: 'all', label: `ทั้งหมด (${licenseCounts.total})` },
            { key: 'pending', label: `รอสแกน (${licenseCounts.pending})` },
            { key: 'confirmed', label: `ยืนยันแล้ว (${licenseCounts.confirmed})` },
          ].map((tab) => (
            <button
              key={tab.key}
              className={'vr-tab' + (licenseTab === tab.key ? ' vr-tab-active' : '')}
              onClick={() => setLicenseTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              <th>แบบ/รุ่น</th>
              <th>ใบอนุญาตนำเข้า</th>
              <th>อินวอยซ์</th>
              <th>หมายเลขเครื่อง</th>
              <th>หมายเลขการผลิต</th>
              <th>หมายเหตุ</th>
              <th>ส่งออกไปประเทศ</th>
              <th>สถานะ</th>
              <th>TAG ที่สแกนคู่</th>
              <th>ยืนยันเมื่อ</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={11} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              licenseRows.map((r) => (
                <tr key={r.ID} className={highlightId === r.ID ? 'il-row-hit' : ''}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {r.ItemNo || '—'}
                  </td>
                  <td data-label="แบบ/รุ่น">{r.Model || '—'}</td>
                  <td data-label="ใบอนุญาตนำเข้า">{r.LicenseNo || '—'}</td>
                  <td data-label="อินวอยซ์">{r.InvoiceNo || '—'}</td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    <strong>{r.MachineNo}</strong>
                  </td>
                  <td className="il-mono" data-label="หมายเลขการผลิต">
                    {r.ProductionNo || '—'}
                  </td>
                  <td data-label="หมายเหตุ">{r.Remark || '—'}</td>
                  <td data-label="ส่งออกไปประเทศ">{r.ExportCountry || '—'}</td>
                  <td data-label="สถานะ">
                    {r.ConfirmStatus === 'CONFIRMED' ? (
                      <span className="il-badge il-badge-ok">✓ ตรงกัน</span>
                    ) : (
                      <span className="il-badge il-badge-pending">⏳ รอสแกน</span>
                    )}
                  </td>
                  <td data-label="TAG ที่สแกนคู่">{r.ConfirmedTag || '—'}</td>
                  <td data-label="ยืนยันเมื่อ">
                    {r.ConfirmedDatetime ? new Date(r.ConfirmedDatetime).toLocaleString('th-TH') : '—'}
                  </td>
                </tr>
              ))}
            {!loading && licenseRows.length === 0 && (
              <tr>
                <td colSpan={11} className="wh-empty-cell">
                  ไม่มีรายการในมุมมองนี้
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* ── ประวัติการสแกน ─────────────────────────────────────────────── */}
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            ประวัติการสแกน ({filtered.length})
          </h2>
          {mismatchCount > 0 && (
            <p className="wh-subtitle" style={{ color: '#b42318', fontWeight: 600 }}>
              มี {mismatchCount} รายการที่สแกนแล้วไม่ตรงกับบัญชีใบอนุญาต
            </p>
          )}
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
          placeholder="ค้นหา Tag / P/N / หมายเลขเครื่อง / ผู้ตรวจสอบ"
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
              <th>หมายเลขเครื่อง / S/N</th>
              <th>ผลเทียบใบอนุญาต</th>
              <th>Checked By</th>
              <th>วันที่</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
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
                  <td className="il-mono" data-label="หมายเลขเครื่อง / S/N">
                    {r.SN || '—'}
                  </td>
                  <td data-label="ผลเทียบใบอนุญาต">{matchBadge(r.MatchStatus)}</td>
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
                <td colSpan={8} className="wh-empty-cell">
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
            <button
              className="wh-modal-cancel"
              onClick={() => goToPage(page + 1)}
              disabled={page === totalPages}
            >
              ›
            </button>
            <button
              className="wh-modal-cancel"
              onClick={() => goToPage(totalPages)}
              disabled={page === totalPages}
            >
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
            <p className="wh-modal-line">หมายเลขเครื่อง / S/N: {detailRow.SN || '—'}</p>
            <p className="wh-modal-line">หมายเลขการผลิต: {detailRow.ProductionNo || '—'}</p>
            <p className="wh-modal-line">ใบอนุญาตนำเข้า: {detailRow.LicenseNo || '—'}</p>
            <p className="wh-modal-line">อินวอยซ์: {detailRow.InvoiceNo || '—'}</p>
            <p className="wh-modal-line">
              ผลเทียบ: {matchBadge(detailRow.MatchStatus)} {detailRow.MatchMessage || ''}
            </p>
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
