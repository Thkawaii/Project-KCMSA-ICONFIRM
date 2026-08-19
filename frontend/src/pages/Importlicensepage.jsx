import { useEffect, useMemo, useState } from 'react'
import {
  getImportLicenseItems,
  getImportLicenseSummary,
  uploadImportLicense,
  previewImportLicense,
  deleteImportLicenseItem,
  clearImportLicense,
  renewImportLicense,
} from '../api/importLicense.js'
import {
  getExportLicense,
  getExportLicenseTrace,
  uploadExportLicense,
  previewExportLicense,
  deleteExportLicense,
  clearExportLicense,
} from '../api/exportLicense.js'
import { PreviewResult, ExtraColumnsCell } from '../components/FormatTools.jsx'
import AppShell from '../components/AppShell.jsx'
import FileDropZone from '../components/Filedropzone.jsx'
import SelectField from '../components/Selectfield.jsx'
import { confirmDelete, toastError, toastSuccess, promptRenewDays } from '../lib/toast.js'
import {
  computeLicenseExpiry,
  formatThaiDate,
  daysLeftLabel,
  STATUS_LABEL,
  EXPIRY_STATUS,
} from '../lib/licenseExpiry.js'
import { useDailyTick } from '../lib/useDailyTick.js'
import { useAppParams } from '../lib/nav.jsx'
import { buildStyledXlsxWorkbookBlob, downloadBlob } from '../lib/xlsx.js'
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClipboardDocumentCheckIcon,
  CubeIcon,
  DocumentTextIcon,
  ReceiptPercentIcon,
  Squares2X2Icon,
} from '../components/icons.jsx'

// เมนูฝั่งคลัง — กรองตาม role ที่ AppShell (WH_MANAGER เห็นครบ, WH เห็นแค่ Part Confirmation)
//   roles บนแต่ละเมนู = role ที่มีสิทธิ์เห็นเมนูนั้น (ไม่ใส่ = เห็นทุก role)
export const WH_NAV_ITEMS = [
  {
    to: '/warehouse',
    label: 'Import License',
    icon: <DocumentTextIcon className="size-4" />,
    roles: ['LOG'],
  },
  {
    to: '/warehouse/export-license',
    label: 'Export License',
    icon: <ReceiptPercentIcon className="size-4" />,
    roles: ['LOG'],
  },
  {
    to: '/warehouse/confirm',
    label: 'Part Confirmation',
    icon: <ClipboardDocumentCheckIcon className="size-4" />,
    roles: ['WH', 'LOG'],
  },
]

// หมายเหตุการออกแบบ:
// หน้านี้เป็น "ตารางอ้างอิง" ล้วนๆ ไม่มีสถานะรอยืนยัน/ยืนยันแล้ว เพราะบัญชี
// แนบท้ายใบอนุญาตผ่านการตรวจจาก กสทช. มาแล้วตั้งแต่ต้นทาง — ของที่อยู่ในนี้
// คือของที่ถูกต้องโดยนิยาม
// สถานะการสแกนยืนยันไปอยู่ที่หน้า Part Confirmation ซึ่งเป็นคนสแกนของจริง

// หน้านี้เหลือแค่บัญชีใบอนุญาตนำเข้า (ชีต Serial) อย่างเดียวแล้ว
//   Export License ย้ายไปเป็นเมนูหลักของตัวเอง (ดู pages/Exportlicensepage.jsx)

// จับคู่สถานะอายุใบอนุญาตกับคลาสป้าย (ใช้ชุดสีเดียวกับ .il-badge ที่มีอยู่แล้ว)
const EXPIRY_BADGE_CLASS = {
  [EXPIRY_STATUS.EXPIRED]: 'il-badge il-badge-bad',
  [EXPIRY_STATUS.EXPIRING]: 'il-badge il-badge-warn',
  [EXPIRY_STATUS.VALID]: 'il-badge il-badge-ok',
  [EXPIRY_STATUS.NO_DATE]: 'il-badge il-badge-muted',
}

// เซลล์ "หมดอายุ (6 เดือน)" — ป้ายสถานะ + วันหมดอายุ + วันคงเหลือ
function ExpiryCell({ issueDate }) {
  const exp = computeLicenseExpiry(issueDate)
  return (
    <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && (
        <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>
      )}
    </div>
  )
}

export default function ImportLicensePage() {
  const today = useDailyTick() // เปลี่ยนค่าเมื่อข้ามวัน → บังคับ recompute สถานะอายุ
  const params = useAppParams() // รับ focusLicense/focusInvoice จากกระดิ่งแจ้งเตือน
  const [items, setItems] = useState([])
  const [summary, setSummary] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [selectedLot, setSelectedLot] = useState('') // 'licenseNo|invoiceNo'
  const [search, setSearch] = useState('')
  const [modelFilter, setModelFilter] = useState('all')
  const [expiryFilter, setExpiryFilter] = useState('all') // สถานะอายุใบอนุญาต
  const [pageSize, setPageSize] = useState(25)
  const [page, setPage] = useState(1)

  const [detailRow, setDetailRow] = useState(null) // แถวที่กำลังเปิดดู modal รายละเอียด

  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadMsg, setUploadMsg] = useState(null)
  const [previewData, setPreviewData] = useState(null)
  const [previewing, setPreviewing] = useState(false)

  async function handlePreview() {
    if (!file) {
      setUploadMsg({ error: 'กรุณาเลือกไฟล์ก่อนตรวจสอบ' })
      return
    }
    setPreviewing(true)
    setPreviewData(null)
    try {
      const data = await previewImportLicense(file)
      setPreviewData(data)
    } catch (err) {
      setUploadMsg({ error: err.message || 'ตรวจสอบไฟล์ไม่สำเร็จ' })
    } finally {
      setPreviewing(false)
    }
  }

  async function loadAll() {
    setLoading(true)
    setLoadError('')
    try {
      const [rows, sum] = await Promise.all([getImportLicenseItems(), getImportLicenseSummary()])
      setItems(rows || [])
      setSummary(sum || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดบัญชีใบอนุญาตนำเข้าไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [])

  useEffect(() => {
    setPage(1)
  }, [selectedLot, search, modelFilter, expiryFilter, pageSize])

  // ── มาจากกระดิ่งแจ้งเตือน: auto-search ใบที่คลิกทันที ──────────────────────
  // เคลียร์ filter อื่น ๆ ก่อน แล้วตั้งคำค้น = เลขใบอนุญาต (ไม่มีไฮไลต์สีแล้ว)
  useEffect(() => {
    const lic = (params?.focusLicense || '').trim()
    if (!lic) return
    setModelFilter('all')
    setExpiryFilter('all')
    setSelectedLot('')
    setSearch(lic)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params?.focusLicense, params?.focusInvoice, params?.focusTs])

  // เมื่อ summary โหลดเสร็จ ค่อย "ปักหมุดล็อต" ให้ตรงใบ (ถ้ามีจริงในบัญชี)
  // ได้การ์ดหัวใบอนุญาต (currentLot) โชว์ใบนั้นเด่น ๆ = "แสดงใบนั้นเลย"
  useEffect(() => {
    const lic = (params?.focusLicense || '').trim()
    const inv = (params?.focusInvoice || '').trim()
    if (!lic || !inv) return
    const key = `${lic}|${inv}`
    if (summary.some((s) => `${s.LicenseNo}|${s.InvoiceNo}` === key)) {
      setSelectedLot(key)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [summary, params?.focusLicense, params?.focusInvoice, params?.focusTs])

  async function handleUpload() {
    if (!file) {
      setUploadMsg({ error: 'กรุณาเลือกไฟล์ Excel หรือ CSV ก่อน' })
      return
    }
    setUploading(true)
    setUploadMsg(null)
    try {
      const result = await uploadImportLicense(file)
      setUploadMsg({
        success: `นำเข้าสำเร็จ — เพิ่มใหม่ ${result.imported} เครื่อง, อัปเดต ${result.updated} เครื่อง, ข้าม ${result.skipped} แถว`,
        problems: result.problems || [],
      })
      setFile(null)
      setPreviewData(null)
      await loadAll()
    } catch (err) {
      setUploadMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  async function handleDeleteRow(row) {
    const ok = await confirmDelete({
      text: `ลบหมายเลขเครื่อง ${row.MachineNo} ออกจากบัญชี?`,
    })
    if (!ok) return
    try {
      await deleteImportLicenseItem(row.ID)
      await loadAll()
      toastSuccess(`ลบ ${row.MachineNo} แล้ว`)
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ'
      setLoadError(msg)
      toastError(msg)
    }
  }

  async function handleClearAllImport() {
    const ok = await confirmDelete({
      text: 'ลบใบอนุญาตนำเข้าทั้งหมดออกจากระบบ? กู้คืนไม่ได้',
      confirmText: 'ลบทั้งหมด',
    })
    if (!ok) return
    try {
      const res = await clearImportLicense('', '', true)
      setSelectedLot('')
      await loadAll()
      toastSuccess(`ลบแล้ว ${res.deleted ?? 0} เครื่อง`)
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleClearLicense(lot) {
    // lot = แถว summary หนึ่ง (คู่ เลขใบอนุญาต + อินวอยซ์)
    const licenseNo = lot?.LicenseNo ?? ''
    const invoiceNo = lot?.InvoiceNo ?? ''
    const label =
      licenseNo || (invoiceNo ? `Invoice ${invoiceNo}` : 'ล็อตนี้ (ไม่มีเลขใบอนุญาต)')
    const ok = await confirmDelete({
      text: `ลบ ${label} ออกจากระบบทั้งล็อต? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งใบ',
    })
    if (!ok) return
    try {
      await clearImportLicense(licenseNo, invoiceNo)
      setSelectedLot('')
      await loadAll()
      toastSuccess(`ลบ ${label} แล้ว`)
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ'
      setLoadError(msg)
      toastError(msg)
    }
  }

  // ── ต่ออายุใบอนุญาตทั้งล็อต ────────────────────────────────────────────────
  // เปิด popup ให้กรอกจำนวนวันที่ต่อ -> เลื่อนวันหมดอายุออกไป -> โหลดใหม่ (คำนวณ realtime)
  async function handleRenewLicense(lot) {
    const licenseNo = lot?.LicenseNo ?? ''
    const invoiceNo = lot?.InvoiceNo ?? ''
    const label =
      licenseNo || (invoiceNo ? `Invoice ${invoiceNo}` : 'ล็อตนี้ (ไม่มีเลขใบอนุญาต)')

    // วันหมดอายุปัจจุบันของล็อต (อ้างอิงจากเครื่องแรกในล็อตที่มีวันที่ออก)
    const lotRows = items.filter(
      (r) => (r.LicenseNo || '') === licenseNo && (r.InvoiceNo || '') === invoiceNo
    )
    const sample = lotRows.find((r) => r.IssueDate) || lotRows[0]
    const curExp = sample ? computeLicenseExpiry(sample.IssueDate) : null
    const curLine = curExp?.hasDate
      ? `<div class="scan-popup-hint">วันหมดอายุปัจจุบัน: <b>${formatThaiDate(curExp.expiryDate)}</b> (${daysLeftLabel(curExp.daysLeft)})</div>`
      : '<div class="scan-popup-hint">ยังไม่ได้ระบุวันที่ออกใบอนุญาต — ต่ออายุจะนับวันหมดอายุใหม่จากวันนี้</div>'

    const days = await promptRenewDays({
      title: `ต่ออายุ ${label}`,
      html: `${curLine}<div class="scan-popup-hint">ระบบจะเลื่อนวันหมดอายุออกไปตามจำนวนวันที่กรอก</div>`,
      defaultDays: 180,
    })
    if (!days) return

    try {
      const res = await renewImportLicense(licenseNo, invoiceNo, days)
      await loadAll() // โหลดใหม่ -> ตาราง/ป้ายสถานะ/กระดิ่ง คำนวณวันหมดอายุใหม่ทันที
      const newExp = res?.newExpiry ? formatThaiDate(new Date(res.newExpiry)) : ''
      toastSuccess(
        `ต่ออายุ ${label} อีก ${days} วันแล้ว${newExp ? ` — หมดอายุ ${newExp}` : ''}`
      )
    } catch (err) {
      const msg = err.message || 'ต่ออายุไม่สำเร็จ'
      setLoadError(msg)
      toastError(msg)
    }
  }

  const filtered = useMemo(() => {
    let rows = items

    if (selectedLot) {
      const [licenseNo, invoiceNo] = selectedLot.split('|')
      rows = rows.filter((r) => r.LicenseNo === licenseNo && r.InvoiceNo === invoiceNo)
    }

    // กรองตามแบบ/รุ่น
    if (modelFilter !== 'all') {
      rows = rows.filter((r) => (r.Model || '') === modelFilter)
    }

    // กรองตามสถานะวันหมดอายุ (ยังไม่ระบุวันที่ / ใกล้หมดอายุ / หมดอายุแล้ว / ปกติ)
    if (expiryFilter !== 'all') {
      rows = rows.filter((r) => computeLicenseExpiry(r.IssueDate).status === expiryFilter)
    }

    const term = search.trim().toLowerCase()
    if (term) {
      rows = rows.filter(
        (r) =>
          (r.MachineNo || '').toLowerCase().includes(term) ||
          (r.ProductionNo || '').toLowerCase().includes(term) ||
          (r.LicenseNo || '').toLowerCase().includes(term) ||
          (r.InvoiceNo || '').toLowerCase().includes(term) ||
          (r.DeclarationNo || '').toLowerCase().includes(term) ||
          (r.Model || '').toLowerCase().includes(term) ||
          (r.ExportCountry || '').toLowerCase().includes(term)
      )
    }

    // เรียงจากวันที่ออกใบอนุญาต (IssueDate) ล่าสุดขึ้นก่อน — แถวที่ยังไม่ระบุวันที่ไปอยู่ท้ายสุด
    rows = [...rows].sort((a, b) => {
      const da = a.IssueDate ? new Date(a.IssueDate).getTime() : NaN
      const db = b.IssueDate ? new Date(b.IssueDate).getTime() : NaN
      const va = Number.isNaN(da) ? -Infinity : da
      const vb = Number.isNaN(db) ? -Infinity : db
      return vb - va
    })

    return rows
  }, [items, selectedLot, modelFilter, expiryFilter, search, today])

  // รายการแบบ/รุ่น (unique) สำหรับ dropdown filter
  const modelOptions = useMemo(() => {
    const set = new Set(items.map((r) => r.Model).filter(Boolean))
    const list = Array.from(set).sort((a, b) => a.localeCompare(b))
    return [{ value: 'all', label: 'ทุกแบบ/รุ่น' }, ...list.map((m) => ({ value: m, label: m }))]
  }, [items])

  // ตัวเลือก filter สถานะวันหมดอายุ — เรียงตามความเร่งด่วน
  const expiryOptions = useMemo(
    () => [
      { value: 'all', label: 'ทุกสถานะวันหมดอายุ' },
      { value: EXPIRY_STATUS.NO_DATE, label: STATUS_LABEL[EXPIRY_STATUS.NO_DATE] }, // ยังไม่ระบุวันที่
      { value: EXPIRY_STATUS.EXPIRING, label: STATUS_LABEL[EXPIRY_STATUS.EXPIRING] }, // ใกล้หมดอายุ
      { value: EXPIRY_STATUS.EXPIRED, label: STATUS_LABEL[EXPIRY_STATUS.EXPIRED] }, // หมดอายุแล้ว
      { value: EXPIRY_STATUS.VALID, label: STATUS_LABEL[EXPIRY_STATUS.VALID] }, // ปกติ
    ],
    []
  )

  // รายการใบอนุญาต (ล็อต) สำหรับ dropdown filter — แทนแถวชิปเดิม
  const lotOptions = useMemo(() => {
    const opts = [{ value: '', label: 'ทุกใบอนุญาต' }]
    summary.forEach((s) => {
      opts.push({
        value: `${s.LicenseNo}|${s.InvoiceNo}`,
        label: `${s.LicenseNo} · Invoice ${s.InvoiceNo} · ${s.Total} เครื่อง`,
      })
    })
    return opts
  }, [summary])

  const counts = useMemo(
    () => ({
      total: items.length,
      licenses: new Set(items.map((r) => r.LicenseNo).filter(Boolean)).size,
      invoices: new Set(items.map((r) => r.InvoiceNo).filter(Boolean)).size,
      models: new Set(items.map((r) => r.Model).filter(Boolean)).size,
    }),
    [items]
  )

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  const currentLot = summary.find((s) => `${s.LicenseNo}|${s.InvoiceNo}` === selectedLot)

  return (
    <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Import License</h2>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      {/* ── อัปโหลดไฟล์บัญชี ─────────────────────────────────────────────── */}
      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone
            file={file}
            onSelect={(f) => {
              setFile(f)
              setUploadMsg(null)
              setPreviewData(null)
            }}
            accept=".xlsx,.xls,.csv"
            label="อัปโหลดบัญชีใบอนุญาตนำเข้า"
            hint="ไฟล์ Excel หรือ CSV ที่มีคอลัมน์ หมายเลขเครื่อง / หมายเลขการผลิต / เลขใบอนุญาตนำเข้า / เลขอินวอยซ์นำเข้า"
            disabled={uploading}
          />
          <button
            className="wh-modal-cancel"
            onClick={handlePreview}
            disabled={previewing || uploading || !file}
            style={{ marginRight: 8 }}
          >
            {previewing ? 'กำลังตรวจสอบ...' : 'ตรวจสอบก่อนอัปโหลด'}
          </button>
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>

        {previewData && <PreviewResult result={previewData} />}

        {uploadMsg?.success && (
          <p className="upload-card-msg upload-card-msg-ok wh-upload-msg">{uploadMsg.success}</p>
        )}
        {uploadMsg?.error && (
          <p className="upload-card-msg upload-card-msg-err wh-upload-msg">{uploadMsg.error}</p>
        )}
        {uploadMsg?.problems?.length > 0 && (
          <ul className="il-problem-list">
            {uploadMsg.problems.map((p, i) => (
              <li key={i}>{p}</li>
            ))}
          </ul>
        )}
      </div>

      {/* ── สรุปตัวเลข ────────────────────────────────────────────────────── */}
      <div className="dash-stats-row wh-stats-row">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>เครื่องในบัญชีทั้งหมด</span>
            <span className="dash-stat-icon dash-icon-blue">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>ใบอนุญาตนำเข้า</span>
            <span className="dash-stat-icon dash-icon-red">
              <DocumentTextIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.licenses}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>อินวอยซ์นำเข้า</span>
            <span className="dash-stat-icon dash-icon-yellow">
              <ReceiptPercentIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.invoices}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>แบบ/รุ่น</span>
            <span className="dash-stat-icon dash-icon-green">
              <CubeIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.models}</div>
        </div>
      </div>

      {/* ── เลือกล็อต (ใบอนุญาต + อินวอยซ์) เป็น dropdown filter ── */}
      {summary.length > 0 && (
        <div className="il-lot-filter">
          <label className="il-lot-filter-label">ใบอนุญาต</label>
          <div className="il-lot-filter-select">
            <SelectField value={selectedLot} onChange={setSelectedLot} options={lotOptions} />
          </div>
        </div>
      )}

      {currentLot && (
        <div className="wh-so-active-bar">
          <div>
            <span className="wh-so-active-label">ใบอนุญาตนำเข้า</span>
            <h3 className="wh-so-active-name">{currentLot.LicenseNo || '(ไม่มีเลขใบอนุญาต)'}</h3>
            <span className="wh-subtitle">
              Invoice {currentLot.InvoiceNo || '—'} · ใบขนสินค้า {currentLot.DeclarationNo || '—'} · รุ่น{' '}
              {currentLot.Model || '—'} · {currentLot.Total} เครื่อง
            </span>
          </div>
          <div className="il-lot-actions">
            <button className="wh-issue-btn il-renew-btn" onClick={() => handleRenewLicense(currentLot)}>
              ต่ออายุ
            </button>
            <button className="wh-modal-cancel" onClick={() => handleClearLicense(currentLot)}>
              ลบทั้งใบ
            </button>
          </div>
        </div>
      )}

      {/* ── ตารางบัญชี ────────────────────────────────────────────────────── */}
      <div className="tsf-history-toolbar">
        <div className="tsf-history-pagesize">
          <div className="wh-pagesize-select">
            <SelectField
              value={pageSize}
              onChange={setPageSize}
              options={[
                { value: 10, label: '10' },
                { value: 25, label: '25' },
                { value: 50, label: '50' },
                { value: 100, label: '100' },
              ]}
            />
          </div>
          entries per page
        </div>
        <div className="il-filter-search-group">
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={modelFilter} onChange={setModelFilter} options={modelOptions} />
          </div>
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={expiryFilter} onChange={setExpiryFilter} options={expiryOptions} />
          </div>
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหา หมายเลขเครื่อง / หมายเลขการผลิต / ใบอนุญาต / อินวอยซ์ / ใบขนสินค้า"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          {items.length > 0 && (
            <button className="wh-btn-danger" onClick={handleClearAllImport}>
              ลบทุกใบอนุญาต
            </button>
          )}
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              <th>ตราอักษร</th>
              <th>แบบ/รุ่น</th>
              <th>เลขใบอนุญาตนำเข้า</th>
              <th>วันที่ออกใบอนุญาต</th>
              <th>หมดอายุ (6 เดือน)</th>
              <th>เลขอินวอยซ์นำเข้า</th>
              <th>เลขใบขนสินค้าขาเข้า</th>
              <th>จำนวน (เครื่อง)</th>
              <th>หมายเลขเครื่อง</th>
              <th>หมายเลขการผลิต</th>
              <th>หมายเหตุ</th>
              <th>ส่งออกไปประเทศ</th>
              <th>คอลัมน์เพิ่ม</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={15} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              paged.map((row, i) => (
                <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {(page - 1) * pageSize + i + 1}
                  </td>
                  <td data-label="ตราอักษร">{row.Brand || '—'}</td>
                  <td data-label="แบบ/รุ่น">{row.Model || '—'}</td>
                  <td data-label="เลขใบอนุญาตนำเข้า">{row.LicenseNo || '—'}</td>
                  <td data-label="วันที่ออกใบอนุญาต">{formatThaiDate(row.IssueDate)}</td>
                  <td data-label="หมดอายุ (6 เดือน)">
                    <ExpiryCell issueDate={row.IssueDate} />
                  </td>
                  <td data-label="เลขอินวอยซ์นำเข้า">{row.InvoiceNo || '—'}</td>
                  <td data-label="เลขใบขนสินค้าขาเข้า">{row.DeclarationNo || '—'}</td>
                  <td data-label="จำนวน (เครื่อง)">{row.Qty}</td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    <strong>{row.MachineNo}</strong>
                  </td>
                  <td className="il-mono" data-label="หมายเลขการผลิต">
                    {row.ProductionNo || '—'}
                  </td>
                  <td data-label="หมายเหตุ">{row.Remark || '—'}</td>
                  <td data-label="ส่งออกไปประเทศ">{row.ExportCountry || '—'}</td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
                  </td>
                  <td className="wh-cell-action">
                    <div className="il-row-actions">
                      <button className="wh-modal-cancel" onClick={() => setDetailRow(row)}>
                        รายละเอียด
                      </button>
                      <button className="wh-btn-danger" onClick={() => handleDeleteRow(row)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={15} className="wh-empty-cell">
                  ยังไม่มีข้อมูลในบัญชี — อัปโหลดไฟล์ Excel หรือ CSV ด้านบนก่อน
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
              <ChevronDoubleLeftIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => goToPage(page - 1)} disabled={page === 1}>
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {page} / {totalPages}
            </span>
            <button
              className="wh-modal-cancel"
              onClick={() => goToPage(page + 1)}
              disabled={page === totalPages}
            >
              <ChevronRightIcon className="size-4" />
            </button>
            <button
              className="wh-modal-cancel"
              onClick={() => goToPage(totalPages)}
              disabled={page === totalPages}
            >
              <ChevronDoubleRightIcon className="size-4" />
            </button>
          </div>
        </div>
      )}

      {detailRow && (
        <ImportDetailModal row={detailRow} onClose={() => setDetailRow(null)} />
      )}
    </AppShell>
  )
}

// ═══════════════════════════════════════════════════════════════════════════
// modal รายละเอียดของ 1 แถวใบอนุญาตนำเข้า — แสดงข้อมูลทุกฟิลด์ของเครื่องแบบอ่านง่าย
// ข้อมูลมีครบอยู่ใน row แล้ว (ไม่ต้องยิง API ซ้ำ) จึงแสดงได้ทันที
// ใช้คลาส il-detail-* ชุดเดียวกับ ExportTraceModal เพื่อให้หน้าตาสอดคล้องกัน
// ═══════════════════════════════════════════════════════════════════════════
function ImportDetailModal({ row, onClose }) {
  // 1 ช่องข้อมูล (label + value) — แสดงเสมอ โชว์ '—' เมื่อไม่มีค่า
  const item = (label, value) => (
    <div className="il-detail-item">
      <span className="il-detail-label">{label}</span>
      <span className="il-detail-value">{value === 0 || value ? value : '—'}</span>
    </div>
  )

  // section หัวข้อ — จัดกลุ่มข้อมูลให้อ่านง่าย
  const section = (title, children) => (
    <div className="il-detail-section">
      <div className="il-detail-section-head">{title}</div>
      <div className="il-detail-grid">{children}</div>
    </div>
  )

  // อายุใบอนุญาต (6 เดือนนับจากวันที่ออก) — คำนวณ realtime เหมือนในตาราง
  const exp = computeLicenseExpiry(row.IssueDate)

  // สถานะยืนยันจากหน้า Part Confirmation (ถ้ามี)
  const CONFIRM_LABEL = {
    CONFIRMED: 'ยืนยันแล้ว',
    PENDING: 'รอยืนยัน',
    REJECTED: 'ไม่ผ่าน',
  }
  const confirmLabel = CONFIRM_LABEL[row.ConfirmStatus] || row.ConfirmStatus

  // คอลัมน์เพิ่ม (extra_json) — คีย์ที่ระบบไม่รู้จัก เก็บไว้ไม่ให้ข้อมูลหาย
  let extraEntries = []
  try {
    const obj = row.extra_json ? JSON.parse(row.extra_json) : null
    if (obj) extraEntries = Object.entries(obj)
  } catch {
    extraEntries = []
  }

  return (
    <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal il-detail-modal" onClick={(e) => e.stopPropagation()}>
        <h3 className="wh-modal-title">รายละเอียดใบอนุญาตนำเข้า</h3>

        <div className="il-detail-body">
        {/* ── ข้อมูลเครื่อง ── */}
        <div className="il-detail-card">
          <div className="il-detail-grid">
            {item('หมายเลขเครื่อง', row.MachineNo)}
            {item('หมายเลขการผลิต', row.ProductionNo)}
            {item('ตราอักษร', row.Brand)}
            {item('แบบ/รุ่น', row.Model)}
            {item('จำนวน (เครื่อง)', row.Qty)}
            {item('ส่งออกไปประเทศ', row.ExportCountry)}
          </div>
        </div>

        <div className="il-detail-links">
        {/* ── ข้อมูลใบอนุญาต ── */}
        {section('ข้อมูลใบอนุญาต', (
          <>
            {item('เลขใบอนุญาตนำเข้า', row.LicenseNo)}
            {item('เลขอินวอยซ์นำเข้า', row.InvoiceNo)}
            {item('เลขใบขนสินค้าขาเข้า', row.DeclarationNo)}
            {item('วันที่ออกใบอนุญาต', row.IssueDate ? formatThaiDate(row.IssueDate) : '')}
            {item('วันหมดอายุ (6 เดือน)', exp.hasDate ? formatThaiDate(exp.expiryDate) : '')}
            {item(
              'สถานะอายุ',
              <span className="il-detail-status">
                <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
                {exp.hasDate && (
                  <span className="il-detail-days">{daysLeftLabel(exp.daysLeft)}</span>
                )}
              </span>,
            )}
            {item('หมายเหตุ', row.Remark)}
          </>
        ))}

        {/* ── สถานะการยืนยัน (จากหน้า Part Confirmation) ── */}
        {section('สถานะการยืนยัน', (
          <>
            {item('สถานะ', confirmLabel)}
            {item('TAG ที่สแกนคู่', row.ConfirmedTag)}
            {item('ผู้ยืนยัน', row.ConfirmedBy)}
            {item(
              'วันเวลาที่ยืนยัน',
              row.ConfirmedDatetime ? formatThaiDate(row.ConfirmedDatetime) : '',
            )}
          </>
        ))}

        {/* ── คอลัมน์เพิ่มจากไฟล์ (ถ้ามี) ── */}
        {extraEntries.length > 0 &&
          section('คอลัมน์เพิ่มจากไฟล์', (
            <>
              {extraEntries.map(([k, v]) =>
                item(String(k).replace(/^\[\+\]\s*/, ''), v),
              )}
            </>
          ))}

        {/* ── ที่มาของข้อมูล ── */}
        {section('ที่มาของข้อมูล', (
          <>
            {item('ชื่อไฟล์', row.FileName)}
            {item('วันที่อัปโหลด', row.UploadDate ? formatThaiDate(row.UploadDate) : '')}
          </>
        ))}
        </div>
        </div>

        <div className="wh-modal-actions">
          <button className="wh-modal-cancel" onClick={onClose}>
            ปิด
          </button>
        </div>
      </div>
    </div>
  )
}

// ═══════════════════════════════════════════════════════════════════════════
// แผง Export License — บัญชีใบอนุญาตส่งออก (คู่กับ Import License)
// ตาราง: ใบขน (Date) · Exception License · Serial Number · Expire date
// ═══════════════════════════════════════════════════════════════════════════

// คำนวณสถานะจาก "วันหมดอายุ" ที่ระบุมาตรง ๆ (ไม่ใช่ +6 เดือนแบบ Import)
// คืนรูปแบบเดียวกับ computeLicenseExpiry เพื่อให้ใช้ ExportExpiryCell ร่วมกับ
// EXPIRY_BADGE_CLASS / STATUS_LABEL / daysLeftLabel ที่มีอยู่แล้วได้ทันที
function computeExpireStatus(expireRaw, withinDays = 30) {
  if (!expireRaw) {
    return { hasDate: false, expiryDate: null, daysLeft: null, status: EXPIRY_STATUS.NO_DATE }
  }
  const exp = new Date(expireRaw)
  if (Number.isNaN(exp.getTime())) {
    return { hasDate: false, expiryDate: null, daysLeft: null, status: EXPIRY_STATUS.NO_DATE }
  }
  const atMidnight = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const today = atMidnight(new Date())
  const expDay = atMidnight(exp)
  const daysLeft = Math.round((expDay - today) / 86400000)

  let status
  if (daysLeft < 0) status = EXPIRY_STATUS.EXPIRED
  else if (daysLeft <= withinDays) status = EXPIRY_STATUS.EXPIRING
  else status = EXPIRY_STATUS.VALID

  return { hasDate: true, expiryDate: expDay, daysLeft, status }
}

// เซลล์ "Expire date" — ป้ายสถานะ + วันหมดอายุ + วันคงเหลือ (ใช้ชุดสีเดียวกับ Import)
function ExportExpiryCell({ expireDate }) {
  const exp = computeExpireStatus(expireDate)
  return (
    <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && (
        <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>
      )}
    </div>
  )
}

// วันหมดอายุใบอนุญาตส่งออก = "วันที่นำออกใบอนุญาต" (Declaration date) + 1 เดือน เสมอ
// (อายุใบอนุญาตส่งออก 1 เดือน) — อ้างอิงคอลัมน์ "วันที่นำออกใบอนุญาต" ที่แสดงในตาราง
// โดยตรง เพื่อให้วันหมดอายุตรงกับวันที่ที่ผู้ใช้เห็น ไม่สลับไปใช้ Expire date จากไฟล์
// (ตกไปใช้ Expire date เฉพาะกรณีไม่มีวันที่นำออกเลย จะได้ไม่ขึ้น "ยังไม่ระบุวันที่")
// ใช้เกณฑ์ใกล้หมดอายุ 7 วัน (อายุแค่เดือนเดียว เกณฑ์ 30 วันจะเตือนตลอด)
function computeExportExpiry(row, withinDays = 7) {
  let expireRaw = null
  if (row.DeclarationDate) {
    const d = new Date(row.DeclarationDate)
    if (!Number.isNaN(d.getTime())) {
      d.setMonth(d.getMonth() + 1)
      expireRaw = d
    }
  }
  if (!expireRaw && row.ExpireDate) {
    expireRaw = row.ExpireDate
  }
  return computeExpireStatus(expireRaw, withinDays)
}

// เซลล์ "หมดอายุ (1 เดือน)" ของฝั่งส่งออก — คิดจากวันนำออก + 1 เดือน (ถ้าไม่มี Expire date)
function ExportOneMonthExpiryCell({ row }) {
  const exp = computeExportExpiry(row)
  return (
    <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && (
        <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>
      )}
    </div>
  )
}

// modal รายละเอียดของ 1 แถวใบอนุญาตส่งออก — เรียก /export-license/:id/trace
// แสดง "รายละเอียดข้อมูล" ของเครื่องแบบอ่านง่าย และแนบข้อมูลที่เชื่อมได้ (ถ้ามี)
// โดยไม่โชว์รายการ "ไม่พบ" ให้รก
function ExportTraceModal({ row, country, onClose }) {
  const [loading, setLoading] = useState(true)
  const [data, setData] = useState(null)
  const [err, setErr] = useState(null)

  useEffect(() => {
    let alive = true
    setLoading(true)
    setErr(null)
    getExportLicenseTrace(row.ID)
      .then((d) => alive && setData(d))
      .catch((e) => alive && setErr(e.message || 'โหลดข้อมูลเชื่อมโยงไม่สำเร็จ'))
      .finally(() => alive && setLoading(false))
    return () => {
      alive = false
    }
  }, [row.ID])

  // 1 ช่องข้อมูล (label + value) — ซ่อนอัตโนมัติถ้าไม่มีค่า
  const item = (label, value) =>
    value ? (
      <div className="il-detail-item">
        <span className="il-detail-label">{label}</span>
        <span className="il-detail-value">{value}</span>
      </div>
    ) : null

  // เหมือน item แต่ "แสดงเสมอ" (โชว์ '—' เมื่อไม่มีค่า)
  // ใช้กับฟิลด์สำคัญที่ต้องเห็นชัดทุกครั้ง เช่น Import License เพื่อไม่ให้ข้อมูลหายเงียบ ๆ
  const itemAlways = (label, value) => (
    <div className="il-detail-item">
      <span className="il-detail-label">{label}</span>
      <span className="il-detail-value">{value || '—'}</span>
    </div>
  )

  // section ข้อมูลที่เชื่อมได้ — แสดงเฉพาะเมื่อมีข้อมูลจริง
  const linkSection = (title, children) => (
    <div className="il-detail-section">
      <div className="il-detail-section-head">{title}</div>
      <div className="il-detail-grid">{children}</div>
    </div>
  )

  return (
    <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal il-detail-modal" onClick={(e) => e.stopPropagation()}>
        <h3 className="wh-modal-title">รายละเอียดใบอนุญาตส่งออก</h3>

        <div className="il-detail-body">
        {/* ── ข้อมูลหลักของเครื่อง ── */}
        <div className="il-detail-card">
          <div className="il-detail-grid">
            {item('Machine No', row.MachineNo)}
            {item('IT Controller S/N', row.ITControllerNo || row.SerialNumber)}
            {item('Serial Number', row.SerialNumber)}
            {item('ประเทศปลายทาง', country)}
            {item('Invoice No.', row.InvoiceNo)}
            {item('Invoice Date', row.InvoiceDate ? formatThaiDate(row.InvoiceDate) : '')}
            {item('Export Entry', row.ExportEntry)}
            {item('Export License', row.ExportLicenseNo || row.ExceptionLicense)}
            {itemAlways('Import License', row.ImportLicenseNo)}
            {item("Date Ass'y", row.AssemblyDate ? formatThaiDate(row.AssemblyDate) : '')}
            {item('Remark', row.Remark)}
          </div>
        </div>

        {loading && <p className="il-detail-note">กำลังโหลดข้อมูลที่เชื่อมโยง...</p>}
        {err && <p className="il-detail-note il-detail-note-err">{err}</p>}

        {!loading && !err && (
          <div className="il-detail-links">
            {/* Import License (บัญชี กสทช.) — แสดง "เสมอ" ไม่ให้ข้อมูลหายเงียบ ๆ
                ถ้าจับคู่บัญชีนำเข้าไม่ได้ ให้โชว์เหตุผลแทนการซ่อนทั้ง section */}
            {linkSection('Import License (บัญชี กสทช.)', (
              data?.importLicense ? (
                <>
                  {item('เลขใบอนุญาต (กสทช.)', data.importLicense.LicenseNo)}
                  {item('Invoice นำเข้า', data.importLicense.InvoiceNo)}
                  {item('รุ่น', data.importLicense.Model)}
                  {item('ประเทศส่งออก', data.importLicense.ExportCountry)}
                  {item('สถานะยืนยัน', data.importLicense.ConfirmStatus)}
                </>
              ) : (
                <div className="il-detail-item" style={{ gridColumn: '1 / -1' }}>
                  <span
                    className="il-detail-value"
                    style={{ color: 'var(--muted, #6b7280)', fontStyle: 'italic' }}
                  >
                    ไม่พบใบอนุญาตนำเข้าที่เชื่อมโยง — ตรวจสอบว่าเลข IT Controller (
                    {row.ITControllerNo || row.SerialNumber || '—'}) ตรงกับ “หมายเลขเครื่อง”
                    ในบัญชีใบอนุญาตนำเข้า และเป็นเลข 12 หลัก
                  </span>
                </div>
              )
            ))}

            {data?.mfgAssembly &&
              linkSection('MFG Assembly (ผลตรวจตอนประกอบ)', (
                <>
                  {item('สถานะ', data.mfgAssembly.Status)}
                  {item('Machine No (ที่ประกอบ)', data.mfgAssembly.MachineNo)}
                </>
              ))}

            {data?.machineSpecs && data.machineSpecs.length > 0 &&
              linkSection('Machine Spec (สเปคเครื่องจักร)', (
                <>
                  {item('ชิ้นส่วนที่ตรงกับเครื่องนี้', `${data.machineSpecs.length} รายการ`)}
                  {item('รุ่นฐาน', data.machineSpecs[0].BaseSpec)}
                  {item('ประเทศ', data.machineSpecs[0].CountryName)}
                </>
              ))}

            {data?.whStock &&
              linkSection('WH Stock (ออเดอร์คลัง)', (
                <>
                  {item('Warehouse', data.whStock.Warehouse)}
                  {item('Work Order', data.whStock.WorkOrder)}
                  {item('Parts No', data.whStock.PartsNo)}
                </>
              ))}
          </div>
        )}
        </div>

        <div className="wh-modal-actions">
          <button className="wh-modal-cancel" onClick={onClose}>
            ปิด
          </button>
        </div>
      </div>
    </div>
  )
}

export function WHExportLicensePanel() {
  useDailyTick() // ข้ามวัน → recompute สถานะ Expire date
  const params = useAppParams() // รับ focusSerial จากกระดิ่งแจ้งเตือน
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [exceptionFilter, setExceptionFilter] = useState('all') // แบบ/รุ่น (ฝั่งส่งออกใช้ Exception License)
  const [traceRow, setTraceRow] = useState(null) // แถวที่กำลังเปิดดู modal เชื่อมโยง
  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [msg, setMsg] = useState(null)
  const [previewData, setPreviewData] = useState(null)
  const [previewing, setPreviewing] = useState(false)
  const [pageSize, setPageSize] = useState(25)
  const [page, setPage] = useState(1)
  const [exportingXlsx, setExportingXlsx] = useState(false)

  // แผนที่ IT Controller No. -> ประเทศปลายทาง (ดึงจากบัญชีใบอนุญาตนำเข้า ExportCountry)
  // ใช้ทั้งแสดงคอลัมน์ Country ในตาราง และตอน Export แยกประเทศ
  const [countryByITC, setCountryByITC] = useState({})

  useEffect(() => {
    let cancelled = false
    async function loadCountryMap() {
      try {
        const imports = await getImportLicenseItems()
        const map = {}
        ;(Array.isArray(imports) ? imports : []).forEach((it) => {
          const key = String(it.MachineNo || '').trim()
          const country = String(it.ExportCountry || '').trim()
          if (key && country) map[key] = country
        })
        if (!cancelled) setCountryByITC(map)
      } catch {
        // ดึงบัญชีนำเข้าไม่ได้ — คอลัมน์ Country จะเป็น "—" (ไม่ทำให้หน้าอื่นพัง)
      }
    }
    loadCountryMap()
    return () => {
      cancelled = true
    }
  }, [])

  // ประเทศปลายทางของ 1 แถวใบอนุญาตส่งออก
  //  1) ใช้ค่า Country ที่มากับไฟล์อัปโหลดโดยตรงก่อน (ฟิลด์ใหม่)
  //  2) ข้อมูลเก่าที่อัปโหลดก่อนรู้จักคอลัมน์นี้ — ยังอยู่ใน extra_json (ไม่ต้องอัปโหลดใหม่)
  //  3) ถ้าไม่มี ค่อยเชื่อมจากบัญชีใบอนุญาตนำเข้า (ExportCountry) ผ่าน IT Controller No.
  function countryOf(r) {
    const direct = String(r.Country || '').trim()
    if (direct) return direct

    // จาก extra_json (ข้อมูลเก่า) — หา key ที่เป็น country/ประเทศ
    try {
      const extra = r.extra_json ? JSON.parse(r.extra_json) : null
      if (extra) {
        for (const [k, v] of Object.entries(extra)) {
          const nk = String(k).replace(/^\[\+\]\s*/, '').toLowerCase().replace(/[\s_./-]/g, '')
          if (['country', 'countryname', 'exportcountry', 'ประเทศ', 'ปลายทาง', 'ส่งออกไปประเทศ'].includes(nk)) {
            const val = String(v || '').trim()
            if (val) return val
          }
        }
      }
    } catch {
      // extra_json อ่านไม่ได้ — ข้ามไปใช้การเชื่อมจากบัญชีนำเข้า
    }

    const a = String(r.ITControllerNo || '').trim()
    const b = String(r.SerialNumber || '').trim()
    return countryByITC[a] || countryByITC[b] || ''
  }

  // Export Excel แยกเป็นชีตต่อประเทศ — จัด Format เหมือนฝั่ง QA (Freeze Header, Header สี Theme
  // ตัวหนากึ่งกลาง, Filter ทุกคอลัมน์, แถบสีสลับแถว, ปรับความกว้างอัตโนมัติ,
  // จัด Alignment ตามชนิดข้อมูล, Format วันที่รูปแบบเดียวกัน)
  //
  // ประเทศไม่มีในบัญชีใบอนุญาตส่งออกโดยตรง — ดึงมาจากบัญชีใบอนุญาตนำเข้า (ExportCountry)
  // โดยเชื่อมผ่าน IT Controller No. 12 หลัก (ExportLicense.ITControllerNo == ImportLicense.MachineNo)
  async function handleExportByCountry() {
    if (exportingXlsx) return
    setExportingXlsx(true)
    try {
      const list = filtered // ส่งออกตามที่กรอง/ค้นหาอยู่ (ทุกหน้า)
      if (!list.length) {
        toastError('ไม่มีรายการให้ Export')
        return
      }

      // จัดกลุ่มตามประเทศ (คงลำดับที่พบ) — ไม่มีประเทศ -> "ไม่ระบุประเทศ"
      const UNKNOWN = 'ไม่ระบุประเทศ'
      const groups = new Map()
      list.forEach((r) => {
        const key = countryOf(r) || UNKNOWN
        if (!groups.has(key)) groups.set(key, [])
        groups.get(key).push(r)
      })

      // เรียงชื่อประเทศ A→Z แต่ให้ "ไม่ระบุประเทศ" อยู่ท้ายสุด
      const countryNames = Array.from(groups.keys()).sort((a, b) => {
        if (a === UNKNOWN) return 1
        if (b === UNKNOWN) return -1
        return a.localeCompare(b)
      })

      const columns = [
        { key: 'item', header: 'Item', type: 'number', width: 6 },
        { key: 'assemblyDate', header: "Date Ass'y", type: 'center', width: 14 },
        { key: 'machineNo', header: 'Machine No', type: 'text' },
        { key: 'itControllerNo', header: 'IT Controller S/N', type: 'text' },
        { key: 'invoiceNo', header: 'Invoice', type: 'text' },
        { key: 'invoiceDate', header: 'Invoice Date', type: 'center', width: 14 },
        { key: 'exportEntry', header: 'Export Entry', type: 'text' },
        { key: 'importLicenseNo', header: 'Import License', type: 'text' },
        { key: 'exportLicenseNo', header: 'Export License', type: 'text' },
        { key: 'country', header: 'Country', type: 'center', width: 14 },
        { key: 'remark', header: 'Remark', type: 'text' },
      ]

      const dash2 = (v) => (v && String(v).trim() !== '' ? String(v) : '—')

      const sheets = countryNames.map((country) => ({
        // ชื่อชีตต้องไม่เกิน 31 ตัว/ไม่มีอักขระต้องห้าม (lib ตัดให้อยู่แล้ว)
        sheetName: country,
        columns,
        rows: groups.get(country).map((r, i) => ({
          item: i + 1,
          assemblyDate: r.AssemblyDate ? formatThaiDate(r.AssemblyDate) : '—',
          machineNo: dash2(r.MachineNo),
          itControllerNo: dash2(r.ITControllerNo || r.SerialNumber),
          invoiceNo: dash2(r.InvoiceNo),
          invoiceDate: r.InvoiceDate ? formatThaiDate(r.InvoiceDate) : '—',
          exportEntry: dash2(r.ExportEntry),
          importLicenseNo: dash2(r.ImportLicenseNo),
          exportLicenseNo: dash2(r.ExportLicenseNo || r.ExceptionLicense),
          country: country === UNKNOWN ? '—' : country,
          remark: dash2(r.Remark),
        })),
      }))

      const blob = buildStyledXlsxWorkbookBlob({ sheets })
      const stamp = new Date().toISOString().slice(0, 10)
      downloadBlob(blob, `ExportLicense-by-country-${stamp}.xlsx`)
      toastSuccess(`Export สำเร็จ — ${countryNames.length} ประเทศ (${list.length} รายการ)`)
    } catch (err) {
      toastError(err.message || 'Export ไม่สำเร็จ')
    } finally {
      setExportingXlsx(false)
    }
  }

  async function handlePreview() {
    if (!file) {
      setMsg({ error: 'กรุณาเลือกไฟล์ก่อนตรวจสอบ' })
      return
    }
    setPreviewing(true)
    setPreviewData(null)
    try {
      const data = await previewExportLicense(file)
      setPreviewData(data)
    } catch (err) {
      setMsg({ error: err.message || 'ตรวจสอบไฟล์ไม่สำเร็จ' })
    } finally {
      setPreviewing(false)
    }
  }

  async function load() {
    setLoading(true)
    try {
      setRows(await getExportLicense())
    } catch (err) {
      toastError(err.message || 'โหลดบัญชีใบอนุญาตส่งออกไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => {
    load()
  }, [])

  useEffect(() => {
    setPage(1)
  }, [search, exceptionFilter, pageSize])

  // ── มาจากกระดิ่งแจ้งเตือน (ฝั่งส่งออก): auto-search ด้วย S/N ที่คลิก ──────────
  useEffect(() => {
    const sn = (params?.focusSerial || '').trim()
    if (!sn) return
    setExceptionFilter('all')
    setSearch(sn)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params?.focusSerial, params?.focusException, params?.focusTs])

  async function handleUpload() {
    if (!file) {
      setMsg({ error: 'กรุณาเลือกไฟล์ Excel หรือ CSV ก่อน' })
      return
    }
    setUploading(true)
    setMsg(null)
    try {
      const r = await uploadExportLicense(file)
      setMsg({ success: `นำเข้าสำเร็จ — ${r.imported} แถว, ข้าม ${r.skipped} แถว` })
      setFile(null)
      setPreviewData(null)
      await load()
    } catch (err) {
      setMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(row) {
    const ok = await confirmDelete({ text: `ลบ Serial Number ${row.SerialNumber || '—'} ออกจากบัญชี?` })
    if (!ok) return
    try {
      await deleteExportLicense(row.ID)
      await load()
      toastSuccess(`ลบ ${row.SerialNumber || ''} แล้ว`)
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleClearAll() {
    const ok = await confirmDelete({
      text: 'ลบบัญชีใบอนุญาตส่งออกทั้งหมด? กู้คืนไม่ได้',
      confirmText: 'ลบทั้งหมด',
    })
    if (!ok) return
    try {
      await clearExportLicense()
      await load()
      toastSuccess('ลบบัญชีใบอนุญาตส่งออกทั้งหมดแล้ว')
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  const filtered = useMemo(() => {
    let list = rows

    // กรองตามแบบ/รุ่น — ฝั่งส่งออกไม่มีคอลัมน์ Model จึงใช้ Exception License เป็นตัวจัดกลุ่ม
    if (exceptionFilter !== 'all') {
      list = list.filter((r) => (r.ExceptionLicense || '') === exceptionFilter)
    }

    const term = search.trim().toLowerCase()
    if (term) {
      list = list.filter((r) =>
        [
          r.SerialNumber,
          r.ExceptionLicense,
          r.MachineNo,
          r.ITControllerNo,
          r.InvoiceNo,
          r.ExportEntry,
          r.ImportLicenseNo,
          r.ExportLicenseNo,
          countryOf(r), // ค้นด้วยชื่อประเทศได้ในช่องค้นหาเดียวกัน
        ]
          .filter(Boolean)
          .some((v) => String(v).toLowerCase().includes(term))
      )
    }

    return list
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, exceptionFilter, search, countryByITC])

  // รายการ Exception License (unique) สำหรับ dropdown filter — เทียบเท่า "แบบ/รุ่น" ของฝั่งนำเข้า
  const exceptionOptions = useMemo(() => {
    const set = new Set(rows.map((r) => r.ExceptionLicense).filter(Boolean))
    const list = Array.from(set).sort((a, b) => a.localeCompare(b))
    return [{ value: 'all', label: 'Export License(ทุกใบ)' }, ...list.map((m) => ({ value: m, label: m }))]
  }, [rows])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)

  return (
    <>
      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone
            file={file}
            onSelect={(f) => {
              setFile(f)
              setMsg(null)
              setPreviewData(null)
            }}
            accept=".xlsx,.xls,.csv"
            label="อัปโหลดบัญชีใบอนุญาตส่งออก"
            hint="ไฟล์ Excel หรือ CSV ที่มีคอลัมน์ ใบขน (Date) / Exception License / Serial Number / Expire date (อัปโหลดซ้ำ Serial เดิม ระบบทับให้)"
            disabled={uploading}
          />
          <button
            className="wh-modal-cancel"
            onClick={handlePreview}
            disabled={previewing || uploading || !file}
            style={{ marginRight: 8 }}
          >
            {previewing ? 'กำลังตรวจสอบ...' : 'ตรวจสอบก่อนอัปโหลด'}
          </button>
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>
        {previewData && <PreviewResult result={previewData} />}
        {msg?.success && <p className="upload-card-msg upload-card-msg-ok wh-upload-msg">{msg.success}</p>}
        {msg?.error && <p className="upload-card-msg upload-card-msg-err wh-upload-msg">{msg.error}</p>}
      </div>

      <div className="tsf-history-toolbar">
        <div className="tsf-history-pagesize">
          <div className="wh-pagesize-select">
            <SelectField
              value={pageSize}
              onChange={setPageSize}
              options={[
                { value: 10, label: '10' },
                { value: 25, label: '25' },
                { value: 50, label: '50' },
                { value: 100, label: '100' },
              ]}
            />
          </div>
          entries per page
        </div>
        <div className="il-filter-search-group">
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={exceptionFilter} onChange={setExceptionFilter} options={exceptionOptions} />
          </div>
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหา Machine No / IT Controller / Invoice / License / ประเทศ"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <button
            className="wh-issue-btn"
            onClick={handleExportByCountry}
            disabled={exportingXlsx || rows.length === 0}
            title="ดาวน์โหลด Excel แยกชีตตามประเทศปลายทาง"
          >
            {exportingXlsx ? 'กำลัง Export...' : 'Export Excel (แยกประเทศ)'}
          </button>
          {rows.length > 0 && (
            <button className="wh-btn-danger" onClick={handleClearAll}>
              ลบทุกใบอนุญาต
            </button>
          )}
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Item</th>
              <th>Date Ass'y</th>
              <th>Machine No</th>
              <th>IT Controller S/N</th>
              <th>Country</th>
              <th>Invoice</th>
              <th>Export Entry</th>
              <th>Import License</th>
              <th>Export License</th>
              <th>วันที่นำออกใบอนุญาต</th>
              <th>หมดอายุ (1 เดือน)</th>
              <th>Remark</th>
              <th>คอลัมน์เพิ่ม</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={14} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              paged.map((row, i) => (
                <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="Item">
                    {(page - 1) * pageSize + i + 1}
                  </td>
                  <td data-label="Date Ass'y">{formatThaiDate(row.AssemblyDate)}</td>
                  <td className="il-mono wh-cell-head" data-label="Machine No">
                    <strong>{row.MachineNo || '—'}</strong>
                  </td>
                  <td className="il-mono" data-label="IT Controller S/N">
                    {row.ITControllerNo || row.SerialNumber || '—'}
                  </td>
                  <td data-label="Country">{countryOf(row) || '—'}</td>
                  <td data-label="Invoice">
                    <div className="il-mono">{row.InvoiceNo || '—'}</div>
                    {row.InvoiceDate && (
                      <div className="il-invoice-date">
                        {formatThaiDate(row.InvoiceDate)}
                      </div>
                    )}
                  </td>
                  <td className="il-mono" data-label="Export Entry">
                    {row.ExportEntry || '—'}
                  </td>
                  <td className="il-mono" data-label="Import License">
                    {row.ImportLicenseNo || '—'}
                  </td>
                  <td className="il-mono" data-label="Export License">
                    {row.ExportLicenseNo || row.ExceptionLicense || '—'}
                  </td>
                  <td data-label="วันที่นำออกใบอนุญาต">
                    {row.DeclarationDate ? formatThaiDate(row.DeclarationDate) : '—'}
                  </td>
                  <td data-label="หมดอายุ (1 เดือน)">
                    <ExportOneMonthExpiryCell row={row} />
                  </td>
                  <td data-label="Remark">{row.Remark || '—'}</td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
                  </td>
                  <td className="wh-cell-action">
                    <div className="il-row-actions">
                      <button className="wh-modal-cancel" onClick={() => setTraceRow(row)}>
                        รายละเอียด
                      </button>
                      <button className="wh-btn-danger" onClick={() => handleDelete(row)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={14} className="wh-empty-cell">
                  ยังไม่มีข้อมูลใบอนุญาตส่งออก — อัปโหลดไฟล์ Excel หรือ CSV ด้านบนก่อน
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {traceRow && (
        <ExportTraceModal row={traceRow} country={countryOf(traceRow)} onClose={() => setTraceRow(null)} />
      )}

      {!loading && filtered.length > pageSize && (
        <div className="tsf-pagination">
          <span className="wh-subtitle" style={{ fontSize: 13 }}>
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of{' '}
            {filtered.length} entries
          </span>
          <div className="tsf-pagination-buttons">
            <button
              className="wh-modal-cancel"
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
            >
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {page} / {totalPages}
            </span>
            <button
              className="wh-modal-cancel"
              onClick={() => setPage(Math.min(totalPages, page + 1))}
              disabled={page === totalPages}
            >
              <ChevronRightIcon className="size-4" />
            </button>
          </div>
        </div>
      )}
    </>
  )
}