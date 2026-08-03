import { useEffect, useMemo, useState } from 'react'
import {
  getImportLicenseItems,
  getImportLicenseSummary,
  uploadImportLicense,
  deleteImportLicenseItem,
  clearImportLicense,
} from '../api/importLicense.js'
import {
  getWHMachineStock,
  uploadWHMachineStock,
  deleteWHMachineStock,
  clearWHMachineStock,
  getWHInvoice,
  uploadWHInvoice,
  deleteWHInvoice,
  clearWHInvoice,
} from '../api/whStock.js'
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

// เมนูของ role WH — เหลือ 2 หน้า: อัปโหลดบัญชีใบอนุญาต แล้วไปสแกนยืนยัน
export const WH_NAV_ITEMS = [
  { to: '/warehouse', label: 'Import License', icon: <DocumentTextIcon className="size-4" /> },
  {
    to: '/warehouse/confirm',
    label: 'Part Confirmation',
    icon: <ClipboardDocumentCheckIcon className="size-4" />,
  },
]

// หมายเหตุการออกแบบ:
// หน้านี้เป็น "ตารางอ้างอิง" ล้วนๆ ไม่มีสถานะรอยืนยัน/ยืนยันแล้ว เพราะบัญชี
// แนบท้ายใบอนุญาตผ่านการตรวจจาก กสทช. มาแล้วตั้งแต่ต้นทาง — ของที่อยู่ในนี้
// คือของที่ถูกต้องโดยนิยาม
// สถานะการสแกนยืนยันไปอยู่ที่หน้า Part Confirmation ซึ่งเป็นคนสแกนของจริง

// แท็บของหน้า WH — อัปโหลดตารางอ้างอิง 3 ชนิดจากไฟล์ Excel เล่มเดียวกัน
//   serial = บัญชีแนบใบอนุญาต (ชีต Serail)  · เดิม
//   mc     = สต๊อกเครื่อง/ออเดอร์ (ชีต MC)   · เอาไว้เช็คของเข้าคลัง
//   inv    = รายการอินวอยซ์ (ชีต Inv)        · ตำแหน่งจัดเก็บ
const WH_TABS = [
  { key: 'serial', label: 'Import License' },
  { key: 'mc', label: 'MC' },
  { key: 'inv', label: 'Inv' },
]

// คอลัมน์ทั้งหมดของชีต MC (เรียงตามไฟล์จริง) — key ตรงกับชื่อฟิลด์ที่ backend ส่งกลับ
// mono = ใช้ฟอนต์ monospace (เลข/รหัส), head = ตัวหนา (คีย์หลัก Order No)
const MC_COLUMNS = [
  { key: 'Warehouse', label: 'Warehouse' },
  { key: 'ForwardingWarehouse', label: 'Forwarding Warehouse' },
  { key: 'StockOutInstDate', label: 'Stock out Inst date' },
  { key: 'STLC', label: 'ST/LC' },
  { key: 'OrderNo', label: 'Order No', mono: true, head: true },
  { key: 'ShippingFinish', label: 'Shipping finish' },
  { key: 'WorkOrder', label: 'Work order', mono: true },
  { key: 'WDetailNo', label: 'W-Detail No.' },
  { key: 'WorkOrderFinish', label: 'Work order finish' },
  { key: 'StockOutNo', label: 'Stock out No.', mono: true },
  { key: 'StockOutFinish', label: 'Stock out finish' },
  { key: 'PartsNo', label: 'Parts No', mono: true },
  { key: 'Name', label: 'Name' },
  { key: 'Pick', label: 'Pick' },
  { key: 'Inst', label: 'Inst' },
  { key: 'Ship', label: 'Ship' },
  { key: 'Remain', label: 'Remain' },
  { key: 'Shortage', label: 'Shortage' },
  { key: 'Mismatch', label: 'Mismatch' },
  { key: 'Pr', label: 'Pr' },
  { key: 'Sp', label: 'Sp' },
  { key: 'AB', label: 'AB' },
  { key: 'StandardCost', label: 'Standard cost' },
  { key: 'Shelf1', label: 'Shelf-1' },
  { key: 'Shelf2', label: 'Shelf-2' },
  { key: 'Note', label: 'Note' },
  { key: 'AssemblyPartsNumber', label: 'Assembly Parts Number' },
  { key: 'AssemblyPartsName', label: 'Assembly Parts Name' },
  { key: 'DL', label: 'DL' },
  { key: 'ReservationNo', label: 'Reservation No.', mono: true },
  { key: 'RDetailNo', label: 'R-Detail No.' },
  { key: 'FinalColor', label: 'Final Color' },
]

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
  const [tab, setTab] = useState('serial')
  const [items, setItems] = useState([])
  const [summary, setSummary] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [selectedLot, setSelectedLot] = useState('') // 'licenseNo|invoiceNo'
  const [search, setSearch] = useState('')
  const [modelFilter, setModelFilter] = useState('all')
  const [pageSize, setPageSize] = useState(25)
  const [page, setPage] = useState(1)

  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadMsg, setUploadMsg] = useState(null)

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
  }, [selectedLot, search, modelFilter, pageSize])

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

  async function handleClearLicense(licenseNo) {
    const ok = await confirmDelete({
      text: `ลบทั้งใบอนุญาต ${licenseNo} ออกจากระบบ? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งใบ',
    })
    if (!ok) return
    try {
      await clearImportLicense(licenseNo)
      setSelectedLot('')
      await loadAll()
      toastSuccess(`ลบใบอนุญาต ${licenseNo} แล้ว`)
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
  }, [items, selectedLot, modelFilter, search])

  // รายการแบบ/รุ่น (unique) สำหรับ dropdown filter
  const modelOptions = useMemo(() => {
    const set = new Set(items.map((r) => r.Model).filter(Boolean))
    const list = Array.from(set).sort((a, b) => a.localeCompare(b))
    return [{ value: 'all', label: 'ทุกแบบ/รุ่น' }, ...list.map((m) => ({ value: m, label: m }))]
  }, [items])

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
          <h2 className="wh-title">Import Data</h2>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      {/* ── อัปโหลดไฟล์บัญชี ─────────────────────────────────────────────── */}
      <div className="vr-tabs il-wh-tabs">
        {WH_TABS.map((t) => (
          <button
            key={t.key}
            className={'vr-tab' + (tab === t.key ? ' vr-tab-active' : '')}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'serial' && (
        <>
      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone
            file={file}
            onSelect={(f) => {
              setFile(f)
              setUploadMsg(null)
            }}
            accept=".xlsx,.xls,.csv"
            label="อัปโหลดบัญชีใบอนุญาตนำเข้า"
            hint="ไฟล์ Excel หรือ CSV ที่มีคอลัมน์ หมายเลขเครื่อง / หมายเลขการผลิต / เลขใบอนุญาตนำเข้า / เลขอินวอยซ์นำเข้า"
            disabled={uploading}
          />
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>

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
            <h3 className="wh-so-active-name">{currentLot.LicenseNo}</h3>
            <span className="wh-subtitle">
              Invoice {currentLot.InvoiceNo} · ใบขนสินค้า {currentLot.DeclarationNo || '—'} · รุ่น{' '}
              {currentLot.Model || '—'} · {currentLot.Total} เครื่อง
            </span>
          </div>
          <button className="wh-modal-cancel" onClick={() => handleClearLicense(currentLot.LicenseNo)}>
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
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหา หมายเลขเครื่อง / หมายเลขการผลิต / ใบอนุญาต / อินวอยซ์ / ใบขนสินค้า"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
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
              paged.map((row) => (
                <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {row.ItemNo || '—'}
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
                  <td className="wh-cell-action">
                    <button className="wh-modal-cancel" onClick={() => handleDeleteRow(row)}>
                      ลบ
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={14} className="wh-empty-cell">
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
        </>
      )}

      {tab === 'mc' && <WHMachineStockPanel />}
      {tab === 'inv' && <WHInvoicePanel />}
    </AppShell>
  )
}
// ═══════════════════════════════════════════════════════════════════════════
// แผง MC — สต๊อกเครื่อง/ออเดอร์ (ชีต MC) เอาไว้เช็คของเข้าคลัง
// ═══════════════════════════════════════════════════════════════════════════
function WHMachineStockPanel() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [msg, setMsg] = useState(null)
  const [pageSize, setPageSize] = useState(25)
  const [page, setPage] = useState(1)

  async function load() {
    setLoading(true)
    try {
      setRows(await getWHMachineStock())
    } catch (err) {
      toastError(err.message || 'โหลดข้อมูล MC ไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => {
    load()
  }, [])

  async function handleUpload() {
    if (!file) {
      setMsg({ error: 'กรุณาเลือกไฟล์ Excel หรือ CSV ก่อน' })
      return
    }
    setUploading(true)
    setMsg(null)
    try {
      const r = await uploadWHMachineStock(file)
      setMsg({ success: `นำเข้าสำเร็จ — ${r.imported} แถว, ข้าม ${r.skipped} แถว` })
      setFile(null)
      await load()
    } catch (err) {
      setMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(row) {
    const ok = await confirmDelete({ text: `ลบ Order No ${row.OrderNo}?` })
    if (!ok) return
    try {
      await deleteWHMachineStock(row.ID)
      await load()
      toastSuccess(`ลบ ${row.OrderNo} แล้ว`)
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleClearAll() {
    const ok = await confirmDelete({
      text: 'ลบสต๊อกเครื่อง (MC) ทั้งหมด? กู้คืนไม่ได้',
      confirmText: 'ลบทั้งหมด',
    })
    if (!ok) return
    try {
      await clearWHMachineStock()
      await load()
      toastSuccess('ลบสต๊อกเครื่องทั้งหมดแล้ว')
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return rows
    return rows.filter(
      (r) =>
        (r.OrderNo || '').toLowerCase().includes(term) ||
        (r.PartsNo || '').toLowerCase().includes(term) ||
        (r.WorkOrder || '').toLowerCase().includes(term) ||
        (r.Warehouse || '').toLowerCase().includes(term) ||
        (r.Name || '').toLowerCase().includes(term)
    )
  }, [rows, search])

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
            }}
            accept=".xlsx,.xls,.csv"
            label="อัปโหลดสต๊อกเครื่อง (ชีต MC)"
            hint="ไฟล์ Excel ที่มีชีต 'MC' — คอลัมน์ Warehouse / Order No / Work order / Parts No / Name (อัปโหลดไฟล์เดิมซ้ำได้ ระบบทับให้)"
            disabled={uploading}
          />
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>
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
        <input
          className="wh-search"
          type="text"
          placeholder="ค้นหา Order No / Parts No / Work order / Warehouse / Name"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        {rows.length > 0 && (
          <button className="wh-modal-cancel" onClick={handleClearAll}>
            ลบทั้งหมด
          </button>
        )}
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              {MC_COLUMNS.map((col) => (
                <th key={col.key}>{col.label}</th>
              ))}
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={MC_COLUMNS.length + 2} className="wh-empty-cell">
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
                  {MC_COLUMNS.map((col) => (
                    <td
                      key={col.key}
                      data-label={col.label}
                      className={
                        (col.mono ? 'il-mono' : '') + (col.head ? ' wh-cell-head' : '')
                      }
                    >
                      {col.head ? <strong>{row[col.key] || '—'}</strong> : row[col.key] || '—'}
                    </td>
                  ))}
                  <td className="wh-cell-action">
                    <button className="wh-modal-cancel" onClick={() => handleDelete(row)}>
                      ลบ
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={MC_COLUMNS.length + 2} className="wh-empty-cell">
                  ยังไม่มีข้อมูล MC — อัปโหลดไฟล์ Excel (ชีต MC) ด้านบนก่อน
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

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

// ═══════════════════════════════════════════════════════════════════════════
// แผง Inv — รายการอินวอยซ์ + ตำแหน่งจัดเก็บ (ชีต Inv)
// ═══════════════════════════════════════════════════════════════════════════
function WHInvoicePanel() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [file, setFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [msg, setMsg] = useState(null)

  async function load() {
    setLoading(true)
    try {
      setRows(await getWHInvoice())
    } catch (err) {
      toastError(err.message || 'โหลดข้อมูล Inv ไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => {
    load()
  }, [])

  async function handleUpload() {
    if (!file) {
      setMsg({ error: 'กรุณาเลือกไฟล์ Excel หรือ CSV ก่อน' })
      return
    }
    setUploading(true)
    setMsg(null)
    try {
      const r = await uploadWHInvoice(file)
      setMsg({ success: `นำเข้าสำเร็จ — ${r.imported} แถว, ข้าม ${r.skipped} แถว` })
      setFile(null)
      await load()
    } catch (err) {
      setMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(row) {
    const ok = await confirmDelete({ text: `ลบแถวอินวอยซ์ P.O. ${row.PONo || '—'} (${row.PartsNo || '—'})?` })
    if (!ok) return
    try {
      await deleteWHInvoice(row.ID)
      await load()
      toastSuccess('ลบแถวแล้ว')
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleClearAll() {
    const ok = await confirmDelete({
      text: 'ลบรายการอินวอยซ์ (Inv) ทั้งหมด? กู้คืนไม่ได้',
      confirmText: 'ลบทั้งหมด',
    })
    if (!ok) return
    try {
      await clearWHInvoice()
      await load()
      toastSuccess('ลบรายการอินวอยซ์ทั้งหมดแล้ว')
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return rows
    return rows.filter(
      (r) =>
        (r.PONo || '').toLowerCase().includes(term) ||
        (r.PartsNo || '').toLowerCase().includes(term) ||
        (r.CNo || '').toLowerCase().includes(term) ||
        (r.Sloc || '').toLowerCase().includes(term) ||
        (r.Shelf || '').toLowerCase().includes(term)
    )
  }, [rows, search])

  return (
    <>
      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone
            file={file}
            onSelect={(f) => {
              setFile(f)
              setMsg(null)
            }}
            accept=".xlsx,.xls,.csv"
            label="อัปโหลดรายการอินวอยซ์ (ชีต Inv)"
            hint="ไฟล์ Excel ที่มีชีต 'Inv' — คอลัมน์ P.O.NO / C/NO. / PARTS NO. / Q'TY / Sloc / Shelf (อัปโหลดซ้ำ P.O. เดิม ระบบทับให้)"
            disabled={uploading}
          />
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>
        {msg?.success && <p className="upload-card-msg upload-card-msg-ok wh-upload-msg">{msg.success}</p>}
        {msg?.error && <p className="upload-card-msg upload-card-msg-err wh-upload-msg">{msg.error}</p>}
      </div>

      <div className="tsf-history-toolbar">
        <input
          className="wh-search"
          type="text"
          placeholder="ค้นหา P.O.NO / Parts No / C/NO. / Sloc / Shelf"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        {rows.length > 0 && (
          <button className="wh-modal-cancel" onClick={handleClearAll}>
            ลบทั้งหมด
          </button>
        )}
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              <th>P.O.NO</th>
              <th>Line No.</th>
              <th>Container</th>
              <th>Package</th>
              <th>C/NO.</th>
              <th>Parts No</th>
              <th>Description</th>
              <th>Q'TY</th>
              <th>Sloc</th>
              <th>Shelf</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={12} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              filtered.map((row, i) => (
                <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {i + 1}
                  </td>
                  <td className="il-mono wh-cell-head" data-label="P.O.NO">
                    <strong>{row.PONo || '—'}</strong>
                  </td>
                  <td data-label="Line No.">{row.LineNo || '—'}</td>
                  <td className="il-mono" data-label="Container">
                    {row.Container || '—'}
                  </td>
                  <td data-label="Package">{row.Package || '—'}</td>
                  <td className="il-mono" data-label="C/NO.">
                    {row.CNo || '—'}
                  </td>
                  <td className="il-mono" data-label="Parts No">
                    {row.PartsNo || '—'}
                  </td>
                  <td data-label="Description">{row.Description || '—'}</td>
                  <td data-label="Q'TY">{row.Qty}</td>
                  <td data-label="Sloc">{row.Sloc || '—'}</td>
                  <td data-label="Shelf">{row.Shelf || '—'}</td>
                  <td className="wh-cell-action">
                    <button className="wh-modal-cancel" onClick={() => handleDelete(row)}>
                      ลบ
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={12} className="wh-empty-cell">
                  ยังไม่มีข้อมูล Inv — อัปโหลดไฟล์ Excel (ชีต Inv) ด้านบนก่อน
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}
