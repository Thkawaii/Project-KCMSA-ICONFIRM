import { useEffect, useMemo, useRef, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import { getMasterData, uploadMasterData, deleteMasterData } from '../api/masterData.js'

const CATEGORIES = [
  { type: 'it_controller', label: 'IT Controller', icon: '🛰️' },
  { type: 'control_valve', label: 'Control Valve', icon: '🔧' },
  { type: 'swing_motor', label: 'Swing Motor', icon: '⚙️' },
  { type: 'motor_propel', label: 'Motor Propel', icon: '🚜' },
  { type: 'pump_assy_hyd', label: 'Pump Assy HYD', icon: '💧' },
]

const navItems = [{ to: '/master-data', label: 'ทะเบียน Master Data', icon: '🗂️' }]

// ค่าที่แสดงแทนช่องว่าง — อะไหล่ชนิดอื่นไม่มี IT Controller no./IMEI
const DASH = '—'

export default function MasterDataPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [componentType, setComponentType] = useState('it_controller')
  const [keyword, setKeyword] = useState('')

  const [pendingFile, setPendingFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadMsg, setUploadMsg] = useState(null)
  const [deletingId, setDeletingId] = useState(0)
  const fileInputRef = useRef(null)

  // นับรอบโหลดใหม่ เพื่อสั่ง refresh ตารางหลังอัปโหลด/ลบ
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    let cancelled = false

    async function load() {
      setLoading(true)
      setLoadError('')
      try {
        const data = await getMasterData({ componentType: componentType || undefined })
        if (!cancelled) setRows(data || [])
      } catch (err) {
        if (!cancelled) setLoadError(err.message || 'โหลดทะเบียนไม่สำเร็จ')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [componentType, reloadKey])

  // ค้นหาแบบ client-side เพราะข้อมูลชุดนี้มีไม่กี่ร้อยแถว โหลดมาทีเดียวแล้ว
  // กรองในเครื่องจะไวกว่ายิง API ทุกตัวอักษร — ค้นได้ทั้ง S/N, IT Controller
  // no., IMEI, P/N และชื่อ จึงยิงบาร์โค้ดใส่ช่องนี้ได้เลย
  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return rows

    return rows.filter((row) =>
      [row.Name, row.Model, row.PartNo, row.SerialNo, row.ITControllerNo, row.IMEI]
        .filter(Boolean)
        .some((field) => String(field).toLowerCase().includes(kw)),
    )
  }, [rows, keyword])

  // ไฟล์ต้นทางนับ Item No. เริ่มที่ 0 แต่หน้าจอให้เริ่มที่ 1
  //
  // คำนวณจาก rows ทั้งชุด (ไม่ใช่ filtered) โดยตั้งใจ — เลขลำดับจะได้ไม่เปลี่ยน
  // ตอนพิมพ์ค้นหา ถ้าคิดจาก filtered แถวเดิมจะกลายเป็นลำดับ 1 ทุกครั้งที่ค้นเจอ
  // แถวเดียว ซึ่งอ่านแล้วสับสนเวลาเทียบกับเอกสาร
  //
  // ใช้ลำดับที่นับเอง ไม่ใช่ ItemNo+1 เพราะแถวที่นำเข้ามาโดยไม่มีคอลัมน์ Item No.
  // จะมีค่าเป็น 0 ทั้งหมด ถ้าบวกหนึ่งตรงๆ จะกลายเป็นเลข 1 ซ้ำกันหลายแถว
  const seqByID = useMemo(() => {
    const map = new Map()
    rows.forEach((row, i) => map.set(row.ID, i + 1))
    return map
  }, [rows])

  const stats = useMemo(() => {
    const withImei = rows.filter((row) => row.IMEI).length
    const models = new Set(rows.map((row) => row.Model).filter(Boolean))
    const partNos = new Set(rows.map((row) => row.PartNo).filter(Boolean))

    return { total: rows.length, withImei, models: models.size, partNos: partNos.size }
  }, [rows])

  const uploadTargetLabel =
    CATEGORIES.find((cat) => cat.type === (componentType || 'it_controller'))?.label || 'IT Controller'

  function handleFileChange(e) {
    setPendingFile(e.target.files?.[0] || null)
    setUploadMsg(null)
  }

  async function handleUpload() {
    if (!pendingFile) {
      setUploadMsg({ error: 'กรุณาเลือกไฟล์ Excel ก่อน' })
      return
    }

    setUploading(true)
    setUploadMsg(null)

    try {
      // ไม่ส่ง componentType ว่างไปให้ backend เดา — ถ้าเลือก "ทุกชนิด" ไว้
      // ให้ลงเป็น it_controller ตาม default ฝั่ง backend
      const result = await uploadMasterData(pendingFile, componentType || 'it_controller')

      setUploadMsg({
        success: `นำเข้าสำเร็จ — เพิ่มใหม่ ${result.imported} รายการ, อัปเดตของเดิม ${result.updated} รายการ`,
        problems: result.problems || [],
      })

      setPendingFile(null)
      if (fileInputRef.current) fileInputRef.current.value = ''
      setReloadKey((n) => n + 1)
    } catch (err) {
      setUploadMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(row) {
    const label = row.SerialNo || row.Name || `รายการ #${row.ID}`
    if (!window.confirm(`ลบ ${label} ออกจากทะเบียนใช่ไหม? กู้คืนไม่ได้`)) return

    setDeletingId(row.ID)
    setLoadError('')

    try {
      await deleteMasterData(row.ID)
      setReloadKey((n) => n + 1)
    } catch (err) {
      setLoadError(err.message || 'ลบไม่สำเร็จ')
    } finally {
      setDeletingId(0)
    }
  }

  function handleExportCsv() {
    const header = ['Item No', 'Part Name', 'Model', 'Part No', 'Serial No', 'IT Controller no.', 'IMEI']

    const body = filtered.map((row) => [
      seqByID.get(row.ID) ?? '',
      row.Name || '',
      row.Model || '',
      excelText(row.PartNo),
      excelText(row.SerialNo),
      excelText(row.ITControllerNo),
      excelText(row.IMEI),
    ])

    // \uFEFF (BOM) เพื่อให้ Excel อ่านภาษาไทยไม่เป็นตัวยึกยือ
    const csv = '\uFEFF' + [header, ...body].map((cols) => cols.map(csvCell).join(',')).join('\r\n')

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `master-data-${componentType || 'all'}-${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <AppShell navItems={navItems} roleLabel="Upload View">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Upload Master Data</h2>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      {/* ===== อัปโหลดจาก Excel ===== */}
      <div className="upload-panel">
        <div className="upload-panel-field">
          <label className="upload-panel-label" htmlFor="md-category-select">
            ชนิดอะไหล่ที่จะนำเข้า
          </label>
          <select
            id="md-category-select"
            className="upload-panel-select"
            value={componentType}
            onChange={(e) => setComponentType(e.target.value)}
          >
            <option value="">ทุกชนิด (นำเข้าเป็น IT Controller)</option>
            {CATEGORIES.map((cat) => (
              <option key={cat.type} value={cat.type}>
                {cat.icon} {cat.label}
              </option>
            ))}
          </select>
        </div>

        <label
          className={'upload-dropzone upload-panel-dropzone' + (pendingFile ? ' upload-dropzone-filled' : '')}
          htmlFor="md-file"
        >
          <input
            id="md-file"
            ref={fileInputRef}
            type="file"
            accept=".xlsx,.xls"
            onChange={handleFileChange}
            className="upload-card-input-hidden"
          />
          <UploadCloudIcon />
          <span className="upload-dropzone-text">
            {pendingFile ? pendingFile.name : 'คลิกเพื่อเลือกไฟล์ Excel'}
          </span>
          <span className="upload-dropzone-hint">
            .xlsx, .xls
          </span>
        </label>

        <button className="wh-issue-btn upload-panel-btn" disabled={uploading} onClick={handleUpload}>
          {uploading ? 'กำลังอัปโหลด...' : `อัปโหลดเข้า ${uploadTargetLabel}`}
        </button>

        {uploadMsg?.success && <p className="upload-card-msg upload-card-msg-ok">{uploadMsg.success}</p>}
        {uploadMsg?.error && <p className="upload-card-msg upload-card-msg-err">{uploadMsg.error}</p>}

        {uploadMsg?.problems?.length > 0 && (
          <ul className="upload-card-msg upload-card-msg-err" style={{ textAlign: 'left', margin: '8px 0 0' }}>
            {uploadMsg.problems.map((problem, i) => (
              <li key={i}>{problem}</li>
            ))}
          </ul>
        )}
      </div>

      {/* ===== สรุป ===== */}
      <div className="dash-stats-row wh-stats-row" style={{ marginTop: 28 }}>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>รายการทั้งหมด</span>
            <span className="dash-stat-icon dash-icon-blue">▦</span>
          </div>
          <div className="dash-stat-value">{stats.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>มี IMEI</span>
            <span className="dash-stat-icon dash-icon-green">🛰️</span>
          </div>
          <div className="dash-stat-value">{stats.withImei}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>จำนวน Model</span>
            <span className="dash-stat-icon dash-icon-yellow">🏷️</span>
          </div>
          <div className="dash-stat-value">{stats.models}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>จำนวน Part No.</span>
            <span className="dash-stat-icon dash-icon-red">🔩</span>
          </div>
          <div className="dash-stat-value">{stats.partNos}</div>
        </div>
      </div>

      {/* ===== ตาราง ===== */}
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            รายการ {keyword.trim() && `(พบ ${filtered.length} จาก ${rows.length})`}
          </h2>
        </div>
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <input
            className="wh-search"
            type="search"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="สแกนหรือพิมพ์ S/N, IT Controller no., IMEI, P/N"
            style={{ minWidth: 280 }}
          />
          <button className="wh-issue-btn" onClick={handleExportCsv} disabled={filtered.length === 0}>
            ⬇ Export CSV
          </button>
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Item No.</th>
              <th>Part Name</th>
              <th>Model</th>
              <th>Part No.</th>
              <th>Serial No.</th>
              <th>IT Controller no.</th>
              <th>IMEI</th>
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
              filtered.map((row) => (
                <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="Item No.">
                    <strong>{seqByID.get(row.ID) ?? DASH}</strong>
                  </td>
                  <td data-label="Part Name">{row.Name || DASH}</td>
                  <td data-label="Model">{row.Model || DASH}</td>
                  <td data-label="Part No." style={codeStyle}>
                    {row.PartNo || DASH}
                  </td>
                  <td data-label="Serial No." style={codeStyle}>
                    {row.SerialNo || DASH}
                  </td>
                  <td data-label="IT Controller no." style={codeStyle}>
                    {row.ITControllerNo || DASH}
                  </td>
                  <td data-label="IMEI" style={codeStyle}>
                    {row.IMEI || DASH}
                  </td>
                  <td className="wh-cell-action">
                    <button
                      className="qa-fail-btn"
                      disabled={deletingId === row.ID}
                      onClick={() => handleDelete(row)}
                    >
                      {deletingId === row.ID ? 'กำลังลบ...' : 'ลบ'}
                    </button>
                  </td>
                </tr>
              ))}

            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  {keyword.trim() ? 'ไม่พบรายการที่ค้นหา' : 'ยังไม่มีข้อมูลในทะเบียน'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </AppShell>
  )
}

// เลขรหัสทุกช่องใช้ฟอนต์ monospace เพื่อให้นับหลักตอนเทียบกับตัวเครื่องได้ง่าย
// (IMEI 15 หลัก / IT Controller no. 12 หลัก อ่านด้วยตายากมากถ้าใช้ฟอนต์ปกติ)
const codeStyle = { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }

// ครอบด้วย ="..." เพื่อบังคับให้ Excel อ่านเป็นข้อความ ไม่งั้น IMEI 15 หลัก
// จะโดนแปลงเป็น 3.00234E+14 และเลข 0 นำหน้าจะหายไป
function excelText(value) {
  if (!value) return ''
  return `="${String(value)}"`
}

function csvCell(value) {
  const s = value == null ? '' : String(value)
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function UploadCloudIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M7 18a4.5 4.5 0 01-.4-8.98A5.5 5.5 0 0117 8.5a4 4 0 01-.5 7.98"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M12 12v7m0-7l-2.5 2.5M12 12l2.5 2.5"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
