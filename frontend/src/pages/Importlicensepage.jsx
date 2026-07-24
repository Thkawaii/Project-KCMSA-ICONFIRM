import { useEffect, useMemo, useState } from 'react'
import {
  getImportLicenseItems,
  getImportLicenseSummary,
  uploadImportLicense,
  deleteImportLicenseItem,
  clearImportLicense,
} from '../api/importLicense.js'
import AppShell from '../components/AppShell.jsx'
import FileDropZone from '../components/Filedropzone.jsx'

// เมนูของ role WH — เหลือ 2 หน้า: อัปโหลดบัญชีใบอนุญาต แล้วไปสแกนยืนยัน
export const WH_NAV_ITEMS = [
  { to: '/warehouse', label: 'Import License', icon: '📄' },
  { to: '/warehouse/confirm', label: 'Part Confirmation', icon: '✅' },
]

// หมายเหตุการออกแบบ:
// หน้านี้เป็น "ตารางอ้างอิง" ล้วนๆ ไม่มีสถานะรอยืนยัน/ยืนยันแล้ว เพราะบัญชี
// แนบท้ายใบอนุญาตผ่านการตรวจจาก กสทช. มาแล้วตั้งแต่ต้นทาง — ของที่อยู่ในนี้
// คือของที่ถูกต้องโดยนิยาม
// สถานะการสแกนยืนยันไปอยู่ที่หน้า Part Confirmation ซึ่งเป็นคนสแกนของจริง

export default function ImportLicensePage() {
  const [items, setItems] = useState([])
  const [summary, setSummary] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [selectedLot, setSelectedLot] = useState('') // 'licenseNo|invoiceNo'
  const [search, setSearch] = useState('')
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
  }, [selectedLot, search, pageSize])

  async function handleUpload() {
    if (!file) {
      setUploadMsg({ error: 'กรุณาเลือกไฟล์ Excel ก่อน' })
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
    if (!window.confirm(`ลบหมายเลขเครื่อง ${row.MachineNo} ออกจากบัญชี?`)) return
    try {
      await deleteImportLicenseItem(row.ID)
      await loadAll()
    } catch (err) {
      setLoadError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleClearLicense(licenseNo) {
    if (!window.confirm(`ลบทั้งใบอนุญาต ${licenseNo} ออกจากระบบ?`)) return
    try {
      await clearImportLicense(licenseNo)
      setSelectedLot('')
      await loadAll()
    } catch (err) {
      setLoadError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  const filtered = useMemo(() => {
    let rows = items

    if (selectedLot) {
      const [licenseNo, invoiceNo] = selectedLot.split('|')
      rows = rows.filter((r) => r.LicenseNo === licenseNo && r.InvoiceNo === invoiceNo)
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
  }, [items, selectedLot, search])

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
          <p className="wh-subtitle">
            บัญชีแสดงหมายเลขเครื่องแนบท้ายใบอนุญาตนำเข้า — ใช้เป็นตัวอ้างอิงให้หน้า Part
            Confirmation เทียบตอนสแกน
          </p>
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
            }}
            accept=".xlsx,.xls"
            label="อัปโหลดบัญชีใบอนุญาตนำเข้า"
            hint="ไฟล์ Excel ที่มีคอลัมน์ หมายเลขเครื่อง / หมายเลขการผลิต / เลขใบอนุญาตนำเข้า / เลขอินวอยซ์นำเข้า"
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
            <span className="dash-stat-icon dash-icon-blue">▦</span>
          </div>
          <div className="dash-stat-value">{counts.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>ใบอนุญาตนำเข้า</span>
            <span className="dash-stat-icon dash-icon-red">📄</span>
          </div>
          <div className="dash-stat-value">{counts.licenses}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>อินวอยซ์นำเข้า</span>
            <span className="dash-stat-icon dash-icon-yellow">🧾</span>
          </div>
          <div className="dash-stat-value">{counts.invoices}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>แบบ/รุ่น</span>
            <span className="dash-stat-icon dash-icon-green">⚙</span>
          </div>
          <div className="dash-stat-value">{counts.models}</div>
        </div>
      </div>

      {/* ── เลือกล็อต (ใบอนุญาต + อินวอยซ์) ──────────────────────────────── */}
      {summary.length > 0 && (
        <div className="il-lot-row">
          <button
            className={'il-lot-chip' + (selectedLot === '' ? ' il-lot-chip-active' : '')}
            onClick={() => setSelectedLot('')}
          >
            ทุกใบอนุญาต
          </button>
          {summary.map((s) => {
            const key = `${s.LicenseNo}|${s.InvoiceNo}`
            return (
              <button
                key={key}
                className={'il-lot-chip' + (selectedLot === key ? ' il-lot-chip-active' : '')}
                onClick={() => setSelectedLot(key)}
              >
                <strong>{s.LicenseNo}</strong>
                <span className="il-lot-chip-sub">
                  Invoice {s.InvoiceNo} · {s.Total} เครื่อง
                </span>
              </button>
            )
          })}
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
          placeholder="ค้นหา หมายเลขเครื่อง / หมายเลขการผลิต / ใบอนุญาต / อินวอยซ์ / ใบขนสินค้า"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              <th>ตราอักษร</th>
              <th>แบบ/รุ่น</th>
              <th>เลขใบอนุญาตนำเข้า</th>
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
                <td colSpan={12} className="wh-empty-cell">
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
                <td colSpan={12} className="wh-empty-cell">
                  ยังไม่มีข้อมูลในบัญชี — อัปโหลดไฟล์ Excel ด้านบนก่อน
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
    </AppShell>
  )
}
