import { useEffect, useMemo, useRef, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import { getMasterData, uploadMasterData, deleteMasterData } from '../api/masterData.js'
import { confirmDelete, toastError, toastSuccess } from '../lib/toast.js'
import { CloudArrowUpIcon } from '../components/icons.jsx'
import {
  ArrowDownTrayIcon,
  CpuChipIcon,
  CubeIcon,
  RectangleStackIcon,
  Squares2X2Icon,
  TagIcon,
} from '../components/icons.jsx'

const CATEGORIES = [
  { type: 'it_controller', label: 'IT Controller' },
  { type: 'control_valve', label: 'Control Valve' },
  { type: 'swing_motor', label: 'Swing Motor' },
  { type: 'motor_propel', label: 'Motor Propel' },
  { type: 'pump_assy_hyd', label: 'Pump Assy HYD' },
]

const navItems = [
  { to: '/master-data', label: 'ทะเบียน Master Data', icon: <RectangleStackIcon className="size-4" /> },
]

// ค่าที่แสดงแทนช่องว่าง — อะไหล่ชนิดอื่นไม่มี IT Controller no./IMEI
const DASH = '—'

export default function MasterDataPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  // ชนิดอะไหล่ที่ใช้ "กรอง/เรียงดู" ในตารางทะเบียน — ไม่เกี่ยวกับตอนอัปโหลดแล้ว
  // (ตอนอัปโหลด backend จะอ่านชนิดจากคอลัมน์ในไฟล์เอง ไม่ต้องเลือกก่อน)
  const [filterType, setFilterType] = useState('')
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
        const data = await getMasterData({ componentType: filterType || undefined })
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
  }, [filterType, reloadKey])

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
      // ไม่ต้องส่งชนิดอะไหล่ไปแล้ว — backend จะอ่านชนิดจากคอลัมน์ในไฟล์เอง
      // (รองรับไฟล์ที่มีอะไหล่หลายชนิดปนกันในไฟล์เดียว)
      const result = await uploadMasterData(pendingFile)

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
    const ok = await confirmDelete({ text: `ลบ ${label} ออกจากทะเบียน? กู้คืนไม่ได้` })
    if (!ok) return

    setDeletingId(row.ID)
    setLoadError('')

    try {
      await deleteMasterData(row.ID)
      setReloadKey((n) => n + 1)
      toastSuccess(`ลบ ${label} แล้ว`)
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ'
      setLoadError(msg)
      toastError(msg)
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
    a.download = `master-data-${filterType || 'all'}-${Date.now()}.csv`
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
      {/* จัดเป็น 2 คอลัมน์เต็มความกว้าง: ซ้าย = ช่องวางไฟล์, ขวา = คำอธิบาย
          คอลัมน์ที่ระบบอ่าน + ปุ่มอัปโหลด — เดิมการ์ดกว้างแค่ 420px เลยเหลือ
          ที่ว่างด้านขวาเป็นครึ่งจอ ไม่สมดุลกับแถวการ์ดสรุปด้านล่าง */}
      <div className="upload-panel upload-panel-wide">
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
          <CloudArrowUpIcon className="size-[26px]" />
          <span className="upload-dropzone-text">
            {pendingFile ? pendingFile.name : 'คลิกเพื่อเลือกไฟล์ Excel'}
          </span>
          <span className="upload-dropzone-hint">
            .xlsx, .xls
          </span>
        </label>

        <div className="upload-panel-side">
          <div className="upload-panel-hint">
            <strong className="upload-panel-hint-title">คอลัมน์ที่ระบบอ่าน</strong>
            Item No. · Part Name · Model · Part No. · Serial No. · IT Controller no. · IMEI
            <br />
            รองรับไฟล์ที่มีอะไหล่หลายชนิดปนกัน — ระบบอ่านชนิดจากคอลัมน์ในไฟล์เอง
            <br />
            ยึด Serial No. เป็นคีย์ อัปโหลดไฟล์เดิมซ้ำจะอัปเดตทับ ไม่เพิ่มซ้ำ
          </div>

          <button className="wh-issue-btn upload-panel-btn" disabled={uploading} onClick={handleUpload}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด Master Data'}
          </button>
        </div>

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
            <span className="dash-stat-icon dash-icon-blue">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>มี IMEI</span>
            <span className="dash-stat-icon dash-icon-green">
              <CpuChipIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.withImei}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>จำนวน Model</span>
            <span className="dash-stat-icon dash-icon-yellow">
              <TagIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.models}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>จำนวน Part No.</span>
            <span className="dash-stat-icon dash-icon-red">
              <CubeIcon className="size-4" />
            </span>
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
        <div className="uv-list-tools md-list-tools" style={{ display: 'flex', gap: 10 }}>
          <select
            className="wh-search md-type-select"
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
            aria-label="กรองดูตามชนิดอะไหล่"
            style={{ minWidth: 150 }}
          >
            <option value="">ทุกชนิด</option>
            {CATEGORIES.map((cat) => (
              <option key={cat.type} value={cat.type}>
                {cat.label}
              </option>
            ))}
          </select>
          <input
            className="wh-search"
            type="search"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="สแกนหรือพิมพ์ S/N, IT Controller no., IMEI, P/N"
            style={{ minWidth: 200, flex: '1 1 200px' }}
          />
          <button className="wh-issue-btn" onClick={handleExportCsv} disabled={filtered.length === 0}>
            <ArrowDownTrayIcon className="size-4" /> Export CSV
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
