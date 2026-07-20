import { useEffect, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import {
  getMachineSpecs,
  uploadMachineSpec,
  deleteMachineSpec,
  exportMachineSpecs,
} from '../api/machineSpec.js'

const CATEGORIES = [
  { type: 'it_controller', label: 'IT Controller P/N & S/N', icon: '🛰️' },
  { type: 'control_valve', label: 'Control Valve S/N', icon: '🔧' },
  { type: 'swing_motor', label: 'Swing Motor S/N', icon: '⚙️' },
  { type: 'motor_propel', label: 'Motor Propel S/N', icon: '🚜' },
  { type: 'pump_assy_hyd', label: 'Pump Assy HYD', icon: '💧' },
]

const navItems = [{ to: '/upload', label: 'Upload Master Data', icon: '⬆️' }]

export default function UploadViewPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [filterType, setFilterType] = useState('')

  const [pendingFiles, setPendingFiles] = useState({})
  const [uploadingType, setUploadingType] = useState('')
  const [uploadMsg, setUploadMsg] = useState({})

  const [exporting, setExporting] = useState(false)

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const data = await getMachineSpecs(filterType || undefined)
      setRows(data || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดรายการไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterType])

  function handleFileChange(type, e) {
    const file = e.target.files?.[0]
    setPendingFiles((prev) => ({ ...prev, [type]: file || null }))
  }

  async function handleUpload(type) {
    const file = pendingFiles[type]
    if (!file) {
      setUploadMsg((prev) => ({ ...prev, [type]: { error: 'กรุณาเลือกไฟล์ Excel ก่อน' } }))
      return
    }

    setUploadingType(type)
    setUploadMsg((prev) => ({ ...prev, [type]: null }))

    try {
      const result = await uploadMachineSpec(type, file)
      setUploadMsg((prev) => ({
        ...prev,
        [type]: { success: `นำเข้าสำเร็จ ${result.imported} รายการ` },
      }))
      setPendingFiles((prev) => ({ ...prev, [type]: null }))
      await loadRows()
    } catch (err) {
      setUploadMsg((prev) => ({ ...prev, [type]: { error: err.message || 'อัปโหลดไม่สำเร็จ' } }))
    } finally {
      setUploadingType('')
    }
  }

  async function handleDelete(id, e) {
    e.stopPropagation()
    if (!window.confirm('ลบรายการนี้ใช่ไหม? กู้คืนไม่ได้')) return
    try {
      await deleteMachineSpec(id)
      await loadRows()
    } catch (err) {
      setLoadError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleExport() {
    setExporting(true)
    try {
      await exportMachineSpecs(filterType || undefined)
    } catch (err) {
      setLoadError(err.message || 'Export ไม่สำเร็จ')
    } finally {
      setExporting(false)
    }
  }

  function categoryLabel(type) {
    return CATEGORIES.find((c) => c.type === type)?.label || type
  }

  function pnSnForRow(row) {
    switch (row.ComponentType) {
      case 'it_controller':
        return { pn: row.ITController || '—', sn: row.ITControllerSN || '—' }
      case 'control_valve':
        return { pn: '—', sn: row.ControlValve || '—' }
      case 'swing_motor':
        return { pn: '—', sn: row.SwingMotor || '—' }
      case 'motor_propel':
        return { pn: '—', sn: row.MotorPropel || '—' }
      case 'pump_assy_hyd':
        return { pn: '—', sn: row.PumpAssyHyd || '—' }
      default:
        return { pn: '—', sn: '—' }
    }
  }

  return (
    <AppShell navItems={navItems} roleLabel="Upload View">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Upload Master Data</h2>
          <p className="wh-subtitle">
            อัปโหลดไฟล์ Excel รายการ P/N &amp; S/N ของแต่ละชิ้นส่วน — แถวแรกของไฟล์ต้องเป็นหัวคอลัมน์
          </p>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      {/* ===== ส่วนบน: การ์ดอัปโหลด 5 หมวด ===== */}
      <div className="upload-card-grid">
        {CATEGORIES.map((cat) => (
          <div className="upload-card" key={cat.type}>
            <div className="upload-card-icon">{cat.icon}</div>
            <h3 className="upload-card-title">{cat.label}</h3>
            <label
              className={'upload-dropzone' + (pendingFiles[cat.type] ? ' upload-dropzone-filled' : '')}
              htmlFor={`file-${cat.type}`}
            >
              <input
                id={`file-${cat.type}`}
                type="file"
                accept=".xlsx,.xls"
                onChange={(e) => handleFileChange(cat.type, e)}
                className="upload-card-input-hidden"
              />
              <UploadCloudIcon />
              <span className="upload-dropzone-text">
                {pendingFiles[cat.type] ? pendingFiles[cat.type].name : 'คลิกเพื่อเลือกไฟล์ Excel'}
              </span>
              <span className="upload-dropzone-hint">.xlsx, .xls</span>
            </label>

            <button
              className="wh-issue-btn upload-card-btn"
              disabled={uploadingType === cat.type}
              onClick={() => handleUpload(cat.type)}
            >
              {uploadingType === cat.type ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
            </button>
            {uploadMsg[cat.type]?.success && (
              <p className="upload-card-msg upload-card-msg-ok">{uploadMsg[cat.type].success}</p>
            )}
            {uploadMsg[cat.type]?.error && (
              <p className="upload-card-msg upload-card-msg-err">{uploadMsg[cat.type].error}</p>
            )}
          </div>
        ))}
      </div>

      {/* ===== ส่วนล่าง: รายการที่อัปโหลดแล้ว ===== */}
      <div className="wh-heading-row" style={{ marginTop: 36 }}>
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            รายการที่อัปโหลดแล้ว
          </h2>
          <p className="wh-subtitle">คลิกแถวเพื่อดูรายละเอียดเต็มของสเปกเครื่องจักร</p>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <select
            className="wh-search"
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
            style={{ minWidth: 200 }}
          >
            <option value="">ทุกหมวด</option>
            {CATEGORIES.map((cat) => (
              <option key={cat.type} value={cat.type}>
                {cat.label}
              </option>
            ))}
          </select>
          <button className="wh-issue-btn" onClick={handleExport} disabled={exporting}>
            {exporting ? 'กำลัง Export...' : '⬇ Export Excel'}
          </button>
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Machine No</th>
              <th>Part Type</th>
              <th>P/N</th>
              <th>S/N</th>
              <th>File Name</th>
              <th>Upload By</th>
              <th>Upload Date</th>
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
              rows.map((row) => {
                const { pn, sn } = pnSnForRow(row)
                return (
                  <tr key={row.ID}>
                    <td>
                      <strong>{row.MachineNo || '—'}</strong>
                    </td>
                    <td>{categoryLabel(row.ComponentType)}</td>
                    <td>{pn}</td>
                    <td>{sn}</td>
                    <td>{row.FileName}</td>
                    <td>{row.User?.Name || '—'}</td>
                    <td>{row.UploadDate ? new Date(row.UploadDate).toLocaleString('th-TH') : '—'}</td>
                    <td>
                      <button className="qa-fail-btn" onClick={(e) => handleDelete(row.ID, e)}>
                        ลบ
                      </button>
                    </td>
                  </tr>
                )
              })}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  ยังไม่มีรายการที่อัปโหลด
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

    </AppShell>
  )
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