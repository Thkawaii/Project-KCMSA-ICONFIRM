import { useEffect, useMemo, useState } from 'react'
import {
  getImportLicenseItems,
  getImportLicenseSummary,
  uploadImportLicense,
  previewImportLicense,
  deleteImportLicenseItem,
  clearImportLicense,
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
import { confirmDelete, toastError, toastSuccess } from '../lib/toast.js'
import {
  computeLicenseExpiry,
  formatThaiDate,
  daysLeftLabel,
  STATUS_LABEL,
  EXPIRY_STATUS,
} from '../lib/licenseExpiry.js'
import { useDailyTick } from '../lib/useDailyTick.js'
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
    roles: ['WH_MANAGER'],
  },
  {
    to: '/warehouse/export-license',
    label: 'Export License',
    icon: <ReceiptPercentIcon className="size-4" />,
    roles: ['WH_MANAGER'],
  },
  {
    to: '/warehouse/confirm',
    label: 'Part Confirmation',
    icon: <ClipboardDocumentCheckIcon className="size-4" />,
    roles: ['WH', 'WH_MANAGER'],
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
          <button className="wh-modal-cancel" onClick={() => handleClearLicense(currentLot)}>
            ลบทั้งใบ
          </button>
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
            <button className="wh-modal-cancel" onClick={handleClearAllImport}>
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
                    <button className="wh-modal-cancel" onClick={() => handleDeleteRow(row)}>
                      ลบ
                    </button>
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
    </AppShell>
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

// modal ลากเส้นทางของ 1 แถว — เรียก /export-license/:id/trace
function ExportTraceModal({ row, onClose }) {
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

  const line = (label, value) =>
    value ? (
      <p className="wh-modal-line">
        <span style={{ color: '#64748b' }}>{label}: </span>
        <strong>{value}</strong>
      </p>
    ) : null

  const section = (title, ok, children) => (
    <div style={{ marginTop: 14 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
        <span className={ok ? 'il-badge il-badge-ok' : 'il-badge il-badge-muted'}>{ok ? 'เชื่อมแล้ว' : 'ไม่พบ'}</span>
        <strong style={{ fontSize: 14 }}>{title}</strong>
      </div>
      {ok && <div style={{ paddingLeft: 4 }}>{children}</div>}
    </div>
  )

  return (
    <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal" style={{ maxWidth: 480 }} onClick={(e) => e.stopPropagation()}>
        <h3 className="wh-modal-title">การเชื่อมโยงของเครื่องนี้</h3>

        <div style={{ background: '#f8fafc', borderRadius: 10, padding: '10px 12px' }}>
          {line('Machine No', row.MachineNo)}
          {line('IT Controller S/N', row.ITControllerNo || row.SerialNumber)}
          {line('Invoice No.', row.InvoiceNo)}
          {line('Export Entry', row.ExportEntry)}
        </div>

        {loading && <p className="wh-modal-line" style={{ marginTop: 14 }}>กำลังโหลด...</p>}
        {err && <p className="wh-modal-line" style={{ marginTop: 14, color: '#b42318' }}>{err}</p>}

        {!loading && !err && data && (
          <>
            {section('Import License (บัญชี กสทช.)', !!data.importLicense,
              data.importLicense && (
                <>
                  {line('เลขใบอนุญาต (กสทช.)', data.importLicense.LicenseNo)}
                  {line('Invoice นำเข้า', data.importLicense.InvoiceNo)}
                  {line('รุ่น', data.importLicense.Model)}
                  {line('ประเทศส่งออก', data.importLicense.ExportCountry)}
                  {line('สถานะยืนยัน', data.importLicense.ConfirmStatus)}
                </>
              ))}

            {section('MFG Assembly (ผลตรวจตอนประกอบ)', !!data.mfgAssembly,
              data.mfgAssembly && (
                <>
                  {line('สถานะ', data.mfgAssembly.Status)}
                  {line('Machine No (ที่ประกอบ)', data.mfgAssembly.MachineNo)}
                </>
              ))}

            {section('Machine Spec (สเปคเครื่องจักร)', !!(data.machineSpecs && data.machineSpecs.length),
              data.machineSpecs && data.machineSpecs.length > 0 && (
                <>
                  {line('จำนวนชิ้นส่วนที่ตรงกับเครื่องนี้', `${data.machineSpecs.length} รายการ`)}
                  {line('รุ่นฐาน', data.machineSpecs[0].BaseSpec)}
                  {line('ประเทศ', data.machineSpecs[0].CountryName)}
                </>
              ))}

            {section('WH Stock (ออเดอร์คลัง)', !!data.whStock,
              data.whStock && (
                <>
                  {line('Warehouse', data.whStock.Warehouse)}
                  {line('Work Order', data.whStock.WorkOrder)}
                  {line('Parts No', data.whStock.PartsNo)}
                </>
              ))}

            {!data.importLicense &&
              !data.mfgAssembly &&
              !(data.machineSpecs && data.machineSpecs.length) &&
              !data.whStock && (
                <p className="wh-modal-line" style={{ marginTop: 14, color: '#64748b' }}>
                  ยังไม่พบข้อมูลที่เชื่อมได้ — อาจยังไม่ได้อัปโหลด Import License / Machine Spec / MFG /
                  WH Stock ของเครื่องนี้ หรือเลข IT Controller / Machine No ไม่ตรงกัน
                </p>
              )}
          </>
        )}

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
        ]
          .filter(Boolean)
          .some((v) => String(v).toLowerCase().includes(term))
      )
    }

    return list
  }, [rows, exceptionFilter, search])

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
            placeholder="ค้นหา Machine No / IT Controller / Invoice / License"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          {rows.length > 0 && (
            <button className="wh-modal-cancel" onClick={handleClearAll}>
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
              <th>Invoice</th>
              <th>Export Entry</th>
              <th>Import License</th>
              <th>Export License</th>
              <th>Remark</th>
              <th>คอลัมน์เพิ่ม</th>
              <th></th>
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
                  <td data-label="Invoice">
                    <div className="il-mono">{row.InvoiceNo || '—'}</div>
                    {row.InvoiceDate && (
                      <div style={{ fontSize: 12, color: '#64748b' }}>
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
                  <td data-label="Remark">{row.Remark || '—'}</td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
                  </td>
                  <td className="wh-cell-action">
                    <button className="wh-modal-cancel" onClick={() => setTraceRow(row)}>
                      รายละเอียด
                    </button>
                    <button className="wh-modal-cancel" onClick={() => handleDelete(row)}>
                      ลบ
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={11} className="wh-empty-cell">
                  ยังไม่มีข้อมูลใบอนุญาตส่งออก — อัปโหลดไฟล์ Excel หรือ CSV ด้านบนก่อน
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {traceRow && <ExportTraceModal row={traceRow} onClose={() => setTraceRow(null)} />}

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
