import { useEffect, useMemo, useRef, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx'
import SelectField from '../components/Selectfield.jsx'
import { getMasterData, uploadMasterData, deleteMasterData, clearMasterData, previewMasterDataChanges } from '../api/masterData.js'
import {
  getUploadData,
  uploadDataFile,
  deleteUploadDataRow,
  updateUploadDataRow,
  clearUploadData,
  previewUploadData,
  generateAssembly,
} from '../api/uploadData.js'
import {
  PreviewResult,
  ChangePreview,
  MasterDataEditModal,
} from '../components/FormatTools.jsx'
import { confirmDelete, toastError, toastSuccess } from '../lib/toast.js'
import { buildStyledXlsxBlob, downloadBlob } from '../lib/xlsx.js'
import { CloudArrowUpIcon } from '../components/icons.jsx'
import {
  ArrowDownTrayIcon,
  CpuChipIcon,
  CubeIcon,
  RectangleStackIcon,
  Squares2X2Icon,
  TagIcon,
} from '../components/icons.jsx'
import '../UploadData.css'

// ── ประเภทอะไหล่ในทะเบียน Master Data (อัปโหลดผ่าน /master-data/upload) ──
// ใช้ร่วมกันทั้ง dropdown "ประเภทที่อัปโหลด" และตัวกรอง (Filter) ในตาราง
// รหัสตรงกับ MasterData.ComponentType ที่ backend เก็บ
// noLabel = ชื่อคอลัมน์ "หมายเลข" เฉพาะของอะไหล่ชนิดนั้น (แสดงแทน "IT Controller no.")
const COMPONENT_TYPES = [
  { value: 'it_controller', label: 'IT Controller', noLabel: 'IT Controller no.' },
  { value: 'swing_motor', label: 'Swing Motor', noLabel: 'Swing Motor No.' },
  { value: 'pump_assy_hyd', label: 'Pump Assy HYD', noLabel: 'Pump Assy HYD NO.' },
  { value: 'motor_propel', label: 'Motor Propel', noLabel: 'Motor Propel NO.' },
  { value: 'control_valve', label: 'Control Valve', noLabel: 'Control Valve NO.' },
]

// เซ็ตรหัสอะไหล่ — เช็คว่า "ประเภทที่อัปโหลด" เป็นทะเบียน Master Data หรือชุดข้อมูลไฟล์
const COMPONENT_TYPE_VALUES = new Set(COMPONENT_TYPES.map((t) => t.value))

// ── ชุดข้อมูลไฟล์ภายนอก (อัปโหลดผ่าน /upload-data) — ตารางไดนามิกตามคอลัมน์ ──
const DATASET_TYPES = [
  { value: 'planning', label: 'Planning' },
  { value: 'wh1', label: 'WH1' },
  { value: 'wh2', label: 'WH2' },
  { value: 'engine', label: 'Engine' },
  { value: 'assembly', label: 'Assembly' },
]

// dropdown "ประเภทที่อัปโหลด" = อะไหล่ทุกชนิด + ชุดข้อมูลไฟล์
const UPLOAD_TYPE_OPTIONS = [
  ...COMPONENT_TYPES.map((t) => ({ value: t.value, label: t.label })),
  ...DATASET_TYPES,
]

// dropdown "ดูในตาราง" = ทะเบียน Master Data (ALL = อะไหล่ทุกชนิดรวมกัน) + ชุดข้อมูลไฟล์
// เลือก ALL แล้วเจาะดูแต่ละชนิดผ่านตัวกรอง (Filter) ในตาราง — ไม่แยกชนิดเป็นตัวเลือกที่นี่
const TYPE_OPTIONS = [
  { value: 'it_controller', label: 'ALL PART' },
  ...DATASET_TYPES,
]

// ป้ายชื่อรวมทุกประเภท (ใช้กับข้อความ/ปุ่มอัปโหลด)
const ALL_TYPE_LABELS = Object.fromEntries(
  [...COMPONENT_TYPES, ...DATASET_TYPES].map((t) => [t.value, t.label]),
)

function typeLabel(value) {
  return ALL_TYPE_LABELS[value] || value
}

// ชนิดอะไหล่ในทะเบียน Master Data — ใช้เป็นตัวกรอง (Filter) ในตาราง IT Controller
const COMPONENT_TYPE_FILTER = [
  { value: 'all', label: 'ทุกชนิด' },
  ...COMPONENT_TYPES.map((t) => ({ value: t.value, label: t.label })),
]

// ชื่อคอลัมน์ "หมายเลข" ตามตัวกรองที่เลือก — all (ทุกชนิด) ใช้ "No." กลางๆ เพราะรวมหลายชนิด
const NO_LABEL_BY_TYPE = {
  all: 'No.',
  ...Object.fromEntries(COMPONENT_TYPES.map((t) => [t.value, t.noLabel])),
}

// ชื่อที่แสดงต่อท้าย "รายการ —" ตามตัวกรอง — all แสดงเป็น ALL PART (ดูอะไหล่ทุกชนิดรวมกัน)
const HEADING_LABEL_BY_TYPE = {
  all: 'ALL PART',
  ...Object.fromEntries(COMPONENT_TYPES.map((t) => [t.value, t.label])),
}

// ── ชนิดการเชื่อมต่อของ IT Controller (ตรงกับค่าคงที่ฝั่ง backend) ──────────────
// ใช้ทำ "รายงานแยก Mobile4G / Satellite" — โชว์เป็นชิปสรุป + คอลัมน์ + ตัวกรอง
const CONNECTIVITY_LABELS = {
  SATELLITE_IRIDIUM: 'Satellite (Iridium)',
  MOBILE_4G_HIGH: '4G (High speed)',
  MOBILE_4G_NORMAL: '4G (Normal speed)',
  UNKNOWN: 'ไม่ระบุ',
}

// ลำดับที่ใช้แสดงชิป/ตัวกรอง (คงที่ อ่านง่าย)
const CONNECTIVITY_ORDER = ['SATELLITE_IRIDIUM', 'MOBILE_4G_HIGH', 'MOBILE_4G_NORMAL', 'UNKNOWN']

const CONNECTIVITY_FILTER = [
  { value: 'all', label: 'ทุก Connectivity' },
  ...CONNECTIVITY_ORDER.map((v) => ({ value: v, label: CONNECTIVITY_LABELS[v] })),
]

const uploadNavItems = [
  { to: '/master-data', label: 'ทะเบียน Master Data', icon: <RectangleStackIcon className="size-4" /> },
  { to: '/format-settings', label: 'Setting', icon: <RectangleStackIcon className="size-4" /> },
]

// ใช้ร่วมกับหน้า Format Settings (role UPLOAD) — export เพื่อไม่ให้ต้องประกาศเมนูซ้ำ
export const FORMAT_NAV_ITEMS = uploadNavItems

// ค่าที่แสดงแทนช่องว่าง — อะไหล่ชนิดอื่นไม่มี IT Controller no./IMEI
const DASH = '—'

export default function MasterDataPage() {
  // ถ้าเข้าจาก role ADMIN ให้ใช้เมนู Admin (User Management + Upload Master Data)
  // ถ้าเป็น role UPLOAD (uploadview) ใช้เมนูเดิม
  // หมายเหตุ: เดิมค่านี้ถูกคำนวณครั้งเดียวตอน module โหลด (นอก component) ซึ่งเกิดขึ้น
  // ก่อน login เสร็จ ทำให้ localStorage['iconfirm_role'] ยังว่างอยู่ -> IS_ADMIN เป็น false
  // ค้างไปตลอด session และ AppShell ซ่อนแถบแท็บเพราะเหลือแค่ 1 เมนู (ทะเบียน Master Data)
  // ต้อง refresh ทั้งหน้าให้ module โหลดใหม่ถึงจะอ่าน role ที่ถูกต้อง — ย้ายมาคำนวณในนี้
  // เพื่อให้อ่าน role สดใหม่ทุกครั้งที่ component render เหมือนที่ AppShell.jsx ทำ
  const isAdmin = (localStorage.getItem('iconfirm_role') || '').toUpperCase() === 'ADMIN'
  const navItems = isAdmin ? ADMIN_NAV_ITEMS : uploadNavItems
  const shellRoleLabel = isAdmin ? 'Admin' : 'Upload View'

  // ประเภทที่จะอัปโหลด (เลือกก่อนอัปโหลด) และประเภทที่กำลังดูในตาราง
  const [uploadType, setUploadType] = useState('it_controller')
  const [viewType, setViewType] = useState('it_controller')
  // ตัวกรองชนิดอะไหล่ในตาราง IT Controller — ยกมาไว้ที่นี่เพื่อให้ตั้งค่าอัตโนมัติหลังอัปโหลดได้
  const [compType, setCompType] = useState('all')

  const [pendingFile, setPendingFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadMsg, setUploadMsg] = useState(null)
  const [previewData, setPreviewData] = useState(null)
  const [previewing, setPreviewing] = useState(false)
  const fileInputRef = useRef(null)

  // Preview รองรับทั้งชุดข้อมูลไฟล์ (dry-run แม็ปคอลัมน์) และทะเบียน Master Data
  // (ตรวจจับการเปลี่ยนข้อมูล NEW/UPDATED/CHANGED/UNCHANGED) — ทั้งคู่ไม่เขียน DB
  const isMasterType = COMPONENT_TYPE_VALUES.has(uploadType)
  const canPreview = true

  async function handlePreview() {
    if (!pendingFile) {
      setUploadMsg({ error: 'กรุณาเลือกไฟล์ก่อนตรวจสอบ' })
      return
    }
    setPreviewing(true)
    setPreviewData(null)
    try {
      const data = isMasterType
        ? await previewMasterDataChanges(pendingFile, uploadType)
        : await previewUploadData(uploadType, pendingFile)
      setPreviewData({ ...data, _mode: isMasterType ? 'change' : 'map' })
    } catch (err) {
      setUploadMsg({ error: err.message || 'ตรวจสอบไฟล์ไม่สำเร็จ' })
    } finally {
      setPreviewing(false)
    }
  }

  // นับรอบโหลดใหม่ เพื่อสั่ง refresh ตารางหลังอัปโหลด/ลบ (ใช้ร่วมทั้ง master-data และ dataset)
  const [reloadKey, setReloadKey] = useState(0)

  function handleFileChange(e) {
    setPendingFile(e.target.files?.[0] || null)
    setUploadMsg(null)
    setPreviewData(null)
  }

  async function handleUpload() {
    if (!pendingFile) {
      setUploadMsg({ error: 'กรุณาเลือกไฟล์ Excel ก่อน' })
      return
    }

    setUploading(true)
    setUploadMsg(null)

    try {
      if (COMPONENT_TYPE_VALUES.has(uploadType)) {
        // ทะเบียน Master Data (IT Controller / Swing Motor / Pump Assy HYD / Motor Propel /
        // Control Valve) — ส่ง component_type เป็นชนิดตั้งต้น ถ้าไฟล์มีคอลัมน์ชนิดอะไหล่เอง
        // backend จะใช้ค่าจากไฟล์ก่อน, ยึด Serial No. เป็นคีย์ อัปโหลดทับได้
        const result = await uploadMasterData(pendingFile, uploadType)
        setUploadMsg({
          success: `นำเข้าสำเร็จ — เพิ่มใหม่ ${result.imported} รายการ, อัปเดตของเดิม ${result.updated} รายการ`,
          problems: result.problems || [],
        })
      } else {
        // Planning / WH1 / WH2 / Engine / Assembly — เพิ่มต่อท้ายข้อมูลเดิม (ไม่ทับ)
        const result = await uploadDataFile(uploadType, pendingFile)
        const parts = []
        if (result.skipped) parts.push(`ข้าม ${result.skipped} แถว`)
        if (result.duplicate) parts.push(`ซ้ำ ${result.duplicate} แถว`)
        const extra = parts.length ? ` (${parts.join(', ')})` : ''
        setUploadMsg({ success: `เพิ่มข้อมูลสำเร็จ ${result.imported} รายการ${extra}` })
      }

      setPendingFile(null)
      setPreviewData(null)
      if (fileInputRef.current) fileInputRef.current.value = ''

      // อัปโหลดเสร็จแล้วสลับตารางไปดูประเภทที่เพิ่งอัปโหลด แล้วรีโหลด
      if (COMPONENT_TYPE_VALUES.has(uploadType)) {
        // อะไหล่ทะเบียน Master Data ดูผ่านตาราง IT Controller + ตั้ง Filter ให้ตรงชนิด
        // (it_controller อาจเป็นไฟล์รวมหลายชนิด จึงตั้ง Filter เป็น "ทุกชนิด")
        setViewType('it_controller')
        setCompType(uploadType === 'it_controller' ? 'all' : uploadType)
      } else {
        setViewType(uploadType)
      }
      setReloadKey((n) => n + 1)
    } catch (err) {
      setUploadMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  return (
    <AppShell navItems={navItems} roleLabel={shellRoleLabel}>
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Upload Master Data</h2>
        </div>
      </div>

      {/* ===== อัปโหลดจาก Excel — เลือกประเภทก่อน แล้วค่อยอัปโหลด ===== */}
      <div className="upload-panel upload-panel-wide">
        <label
          className={'upload-dropzone upload-panel-dropzone' + (pendingFile ? ' upload-dropzone-filled' : '')}
          htmlFor="md-file"
        >
          <input
            id="md-file"
            ref={fileInputRef}
            type="file"
            accept=".xlsx,.xls,.csv"
            onChange={handleFileChange}
            className="upload-card-input-hidden"
          />
          <CloudArrowUpIcon className="size-[26px]" />
          <span className="upload-dropzone-text">
            {pendingFile ? pendingFile.name : `คลิกเพื่อเลือกไฟล์ (${typeLabel(uploadType)})`}
          </span>
          <span className="upload-dropzone-hint">.xlsx, .xls, .csv</span>
        </label>

        <div className="upload-panel-side">
          <div className="upload-panel-field">
            <label className="upload-panel-label" htmlFor="md-upload-type">
              ประเภทที่อัปโหลด
            </label>
            <SelectField
              value={uploadType}
              onChange={(v) => {
                setUploadType(v)
                setPendingFile(null)
                setUploadMsg(null)
                setPreviewData(null)
                if (fileInputRef.current) fileInputRef.current.value = ''
              }}
              options={UPLOAD_TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label }))}
            />
          </div>

          {/* flexWrap: ปุ่มทั้งสองมี class upload-panel-btn (width:100%) — ถ้าไม่ wrap
              บนมือถือปุ่มจะพยายามยืดเต็มแถวพร้อมกันสองปุ่ม ข้อความยาวดันความกว้างรวม
              เกินจอ ทำให้กล่องอัปโหลดล้นออกนอกจอ (ถูก .shell overflow-x:hidden ตัดทิ้ง
              มองไม่เห็นเนื้อหาส่วนที่ล้น) — ให้ wrap แล้วแต่ละปุ่มขึ้นบรรทัดของตัวเองแทน */}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {canPreview && (
              <button
                className="qa-fail-btn upload-panel-btn"
                style={{ background: '#eef2ff', color: '#4338ca', borderColor: '#c7d2fe' }}
                disabled={previewing || uploading}
                onClick={handlePreview}
              >
                {previewing ? 'กำลังตรวจสอบ...' : 'ตรวจสอบก่อนอัปโหลด'}
              </button>
            )}
            <button className="wh-issue-btn upload-panel-btn" disabled={uploading} onClick={handleUpload}>
              {uploading ? 'กำลังอัปโหลด...' : `อัปโหลด ${typeLabel(uploadType)}`}
            </button>
          </div>
        </div>

        {previewData &&
          (previewData._mode === 'change' ? (
            <ChangePreview result={previewData} />
          ) : (
            <PreviewResult result={previewData} />
          ))}

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

      {/* ===== ตัวเลือกประเภทที่จะดูในตาราง ===== */}
      <div className="wh-heading-row" style={{ marginTop: 28 }}>
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            รายการ — {viewType === 'it_controller' ? HEADING_LABEL_BY_TYPE[compType] : typeLabel(viewType)}
          </h2>
        </div>
        <div className="md-type-field">
          <SelectField
            value={viewType}
            onChange={setViewType}
            options={TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label }))}
          />
        </div>
      </div>

      {viewType === 'it_controller' ? (
        <ITControllerView
          reloadKey={reloadKey}
          bumpReload={() => setReloadKey((n) => n + 1)}
          compType={compType}
          setCompType={setCompType}
        />
      ) : (
        <DatasetView key={`${viewType}-${reloadKey}`} dataset={viewType} />
      )}
    </AppShell>
  )
}

/* ─────────────────────────────────────────────────────────────────────────
   IT Controller = ทะเบียน Master Data เดิม (สรุป + ตาราง + ค้นหา + Export CSV)
   ยกเนื้อในเดิมมาทั้งหมด ไม่แตะพฤติกรรม
   ───────────────────────────────────────────────────────────────────────── */
function ITControllerView({ reloadKey, bumpReload, compType, setCompType }) {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [deletingId, setDeletingId] = useState(0)
  const [editRow, setEditRow] = useState(null)
  const [connFilter, setConnFilter] = useState('all')

  // ชื่อคอลัมน์ "หมายเลข" ของตารางนี้ — เปลี่ยนตามตัวกรอง (Filter) ที่เลือก
  const noLabel = NO_LABEL_BY_TYPE[compType] || 'IT Controller no.'

  // ชนิดอะไหล่สำหรับ dropdown ในกล่องแก้ไข + panel จับคู่ค่ารหัส
  const editComponentOptions = COMPONENT_TYPES.map((t) => ({ value: t.value, label: t.label }))

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setLoadError('')
      try {
        const data = await getMasterData({})
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
  }, [reloadKey])

  const filtered = useMemo(() => {
    let result = rows

    // กรองตามชนิดอะไหล่ (Filter)
    if (compType !== 'all') {
      result = result.filter((row) => row.ComponentType === compType)
    }

    const kw = keyword.trim().toLowerCase()
    if (kw) {
      result = result.filter((row) =>
        [row.Name, row.Model, row.PartNo, row.SerialNo, row.ITControllerNo, row.IMEI]
          .filter(Boolean)
          .some((field) => String(field).toLowerCase().includes(kw)),
      )
    }

    // กรองตามชนิดการเชื่อมต่อ (Connectivity) — 'UNKNOWN' = แถวที่ยังไม่ระบุ
    if (connFilter !== 'all') {
      result = result.filter((row) => (row.ConnectivityType || 'UNKNOWN') === connFilter)
    }

    return result
  }, [rows, keyword, compType, connFilter])

  const stats = useMemo(() => {
    const withImei = rows.filter((row) => row.IMEI).length
    const models = new Set(rows.map((row) => row.Model).filter(Boolean))
    const partNos = new Set(rows.map((row) => row.PartNo).filter(Boolean))
    return { total: rows.length, withImei, models: models.size, partNos: partNos.size }
  }, [rows])

  async function handleDelete(row) {
    const label = row.SerialNo || row.Name || `รายการ #${row.ID}`
    const ok = await confirmDelete({ text: `ลบ ${label} ออกจากทะเบียน? กู้คืนไม่ได้` })
    if (!ok) return
    setDeletingId(row.ID)
    setLoadError('')
    try {
      await deleteMasterData(row.ID)
      bumpReload()
      toastSuccess(`ลบ ${label} แล้ว`)
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ'
      setLoadError(msg)
      toastError(msg)
    } finally {
      setDeletingId(0)
    }
  }

  async function handleClearAll() {
    const isAll = compType === 'all'
    const label = isAll ? 'ทะเบียนทั้งหมด' : HEADING_LABEL_BY_TYPE[compType] || compType
    const ok = await confirmDelete({
      text: `ลบ ${label} ออกจากทะเบียนทั้งหมด? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งหมด',
    })
    if (!ok) return
    setLoadError('')
    try {
      const res = isAll
        ? await clearMasterData({ all: true })
        : await clearMasterData({ componentType: compType })
      bumpReload()
      toastSuccess(`ลบแล้ว ${res.deleted ?? 0} รายการ`)
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ'
      setLoadError(msg)
      toastError(msg)
    }
  }

  // Export เป็น Excel (.xlsx) แบบจัด Format ให้เหมือนตารางอื่น — Freeze Header,
  // Header สี Theme ตัวหนากึ่งกลาง, Filter ทุกคอลัมน์ (Excel Table), แถบสีสลับแถว,
  // ปรับความกว้างอัตโนมัติ, Border, จัด Alignment ตามชนิดข้อมูล
  // (รหัส S/N, IT Controller no., IMEI, P/N คงเป็น "ข้อความ" กันเลขยาวเพี้ยนใน Excel)
  function handleExport() {
    const columns = [
      { key: 'itemNo', header: 'Item No.', type: 'number', width: 8 },
      { key: 'name', header: 'Part Name', type: 'text' },
      { key: 'model', header: 'Model', type: 'text' },
      { key: 'partNo', header: 'Part No.', type: 'text' },
      { key: 'serialNo', header: 'Serial No.', type: 'text' },
      { key: 'itcNo', header: noLabel, type: 'text' },
      { key: 'imei', header: 'IMEI', type: 'text' },
    ]
    const rows = filtered.map((row, i) => ({
      itemNo: i + 1,
      name: row.Name || '',
      model: row.Model || '',
      partNo: row.PartNo || '',
      serialNo: row.SerialNo || '',
      itcNo: row.ITControllerNo || '',
      imei: row.IMEI || '',
    }))
    const sheetName = (compType === 'all' ? 'ALL PART' : HEADING_LABEL_BY_TYPE[compType] || 'Master Data').slice(0, 31)
    const blob = buildStyledXlsxBlob({ sheetName, columns, rows })
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
    downloadBlob(blob, `master-data-${compType}-${stamp}.xlsx`)
  }

  return (
    <>
      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="dash-stats-row wh-stats-row" style={{ marginTop: 4, marginBottom: 24 }}>
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

      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 17 }}>
            {(keyword.trim() || compType !== 'all') && `พบ ${filtered.length} จาก ${rows.length}`}
          </h2>
        </div>
        <div className="uv-list-tools md-list-tools" style={{ display: 'flex', gap: 10 }}>
          <div className="md-type-field" style={{ minWidth: 170 }}>
            <SelectField
              value={compType}
              onChange={setCompType}
              options={COMPONENT_TYPE_FILTER.map((t) => ({ value: t.value, label: t.label }))}
            />
          </div>
          <div className="md-type-field" style={{ minWidth: 190 }}>
            <SelectField
              value={connFilter}
              onChange={setConnFilter}
              options={CONNECTIVITY_FILTER}
            />
          </div>
          <input
            className="wh-search"
            type="search"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder={`สแกนหรือพิมพ์ S/N, ${noLabel}, IMEI, P/N`}
            style={{ minWidth: 200, flex: '1 1 200px' }}
          />
          <button className="wh-issue-btn" onClick={handleExport} disabled={filtered.length === 0}>
            <ArrowDownTrayIcon className="size-4" /> Export Excel
          </button>
          <button className="qa-fail-btn" onClick={handleClearAll} disabled={rows.length === 0}>
            {compType === 'all' ? 'ลบทั้งหมด' : 'ลบทั้งชนิด'}
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
              <th>{noLabel}</th>
              <th>IMEI</th>
              <th>Connectivity</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={9} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}

            {!loading &&
              filtered.map((row, i) => (
                <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="Item No.">
                    <strong>{i + 1}</strong>
                  </td>
                  <td data-label="Part Name">{row.Name || DASH}</td>
                  <td data-label="Model">{row.Model || DASH}</td>
                  <td data-label="Part No." style={codeStyle}>
                    {row.PartNo || DASH}
                  </td>
                  <td data-label="Serial No." style={codeStyle}>
                    {row.SerialNo || DASH}
                  </td>
                  <td data-label={noLabel} style={codeStyle}>
                    {row.ITControllerNo || DASH}
                  </td>
                  <td data-label="IMEI" style={codeStyle}>
                    {row.IMEI || DASH}
                  </td>
                  <td data-label="Connectivity">
                    {row.ComponentType === 'it_controller'
                      ? CONNECTIVITY_LABELS[row.ConnectivityType || 'UNKNOWN']
                      : DASH}
                  </td>
                  <td className="wh-cell-action">
                    <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                      <button className="wh-issue-btn" onClick={() => setEditRow(row)}>
                        แก้ไข
                      </button>
                      <button
                        className="qa-fail-btn"
                        disabled={deletingId === row.ID}
                        onClick={() => handleDelete(row)}
                      >
                        {deletingId === row.ID ? 'กำลังลบ...' : 'ลบ'}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}

            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={9} className="wh-empty-cell">
                  {keyword.trim() || compType !== 'all'
                    ? 'ไม่พบรายการตามตัวกรอง'
                    : 'ยังไม่มีข้อมูลในทะเบียน'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {editRow && (
        <MasterDataEditModal
          row={editRow}
          componentOptions={editComponentOptions}
          itcLabel={noLabel}
          onClose={() => setEditRow(null)}
          onSaved={bumpReload}
        />
      )}
    </>
  )
}

/* ─────────────────────────────────────────────────────────────────────────
   Planning / WH1 / WH2 / Engine — ตารางไดนามิกตามคอลัมน์ที่ backend ส่งมา
   ───────────────────────────────────────────────────────────────────────── */
const UD_PAGE_SIZE = 100

function DatasetView({ dataset }) {
  const label = typeLabel(dataset)

  const [columns, setColumns] = useState([])
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [exporting, setExporting] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [localReload, setLocalReload] = useState(0)
  const [editRow, setEditRow] = useState(null)

  // ── pagination ──
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [totalPages, setTotalPages] = useState(1)

  // ค้นทันทีที่พิมพ์ (แบบ debounce) — กลับหน้า 1 เสมอ แล้ว trigger โหลด
  // (setPage+setLocalReload ถูก batch เป็น render เดียวใน React 18 effect จึงยิง
  // ครั้งเดียวด้วย page=1)
  function runSearch() {
    setPage(1)
    setLocalReload((n) => n + 1)
  }

  // debounce การค้น: หน่วง 350ms หลังหยุดพิมพ์ค่อยยิง query กัน request ถี่ทุกตัวอักษร
  // ข้าม run แรก (ตอน mount keyword='' โหลดหลักทำงานอยู่แล้ว) กันโหลดซ้ำ
  const firstKeywordRun = useRef(true)
  useEffect(() => {
    if (firstKeywordRun.current) {
      firstKeywordRun.current = false
      return
    }
    const t = setTimeout(runSearch, 350)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword])

  // ── ปั๊มตาราง Assembly อัตโนมัติตอนเปิดหน้า (ไม่ต้องกดปุ่ม) ─────────────────
  // เมื่อผู้ใช้เปิดตาราง Assembly ระบบจะดึงข้อมูลจาก Planning / WH1 / Engine +
  // ทะเบียนกลาง มาปั๊มให้เองก่อน แล้วค่อยโหลดตารางมาแสดง (คอลัมน์ IT Controller
  // จึงขึ้นค่าอัตโนมัติทันที ไม่ต้องรอผู้ใช้กดปุ่ม "สร้าง Assembly อัตโนมัติ")
  // ทำครั้งเดียวต่อการเปิดหน้า (เงียบ ๆ ไม่เด้ง toast กวน) — ปุ่มยังกดสร้างซ้ำได้
  const autoGenDone = useRef(false)
  useEffect(() => {
    if (dataset !== 'assembly' || autoGenDone.current) return
    autoGenDone.current = true
    let cancelled = false
    ;(async () => {
      try {
        await generateAssembly()
      } catch {
        // ปั๊มอัตโนมัติล้มเหลว (เช่นยังไม่มีข้อมูลต้นทาง) — ไม่รบกวนผู้ใช้
        // ตารางจะโหลดข้อมูลเดิมที่มีอยู่ตามปกติ และยังกดปุ่มสร้างเองได้
      }
      if (!cancelled) setLocalReload((n) => n + 1)
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dataset])

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setLoadError('')
      try {
        const data = await getUploadData(dataset, keyword || undefined, page, UD_PAGE_SIZE)
        if (!cancelled) {
          setColumns(data?.columns || [])
          setRows(data?.rows || [])
          setTotal(data?.total ?? 0)
          setTotalPages(data?.totalPages || 1)
        }
      } catch (err) {
        if (!cancelled) setLoadError(err.message || 'โหลดรายการไม่สำเร็จ')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => {
      cancelled = true
    }
    // keyword ค้นแบบ debounce (ดู effect ด้านบน) ไม่ผูกกับทุกตัวอักษรใน render นี้
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dataset, localReload, page])

  async function handleDelete(id) {
    const ok = await confirmDelete({ text: 'ลบแถวนี้? กู้คืนไม่ได้' })
    if (!ok) return
    try {
      await deleteUploadDataRow(id)
      setLocalReload((n) => n + 1)
      toastSuccess('ลบแถวแล้ว')
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  // บันทึกการแก้ไขแถว (data = { ชื่อคอลัมน์: ค่า }) แล้วโหลดตารางใหม่
  async function handleSaveEdit(id, data) {
    await updateUploadDataRow(id, data)
    setEditRow(null)
    setLocalReload((n) => n + 1)
    toastSuccess('บันทึกการแก้ไขแล้ว')
  }

  async function handleClear() {
    const ok = await confirmDelete({ text: `ล้างข้อมูล ${label} ทั้งหมด? กู้คืนไม่ได้` })
    if (!ok) return
    try {
      const res = await clearUploadData(dataset)
      setPage(1)
      setLocalReload((n) => n + 1)
      toastSuccess(`ล้างแล้ว ${res.deleted ?? 0} แถว`)
    } catch (err) {
      toastError(err.message || 'ล้างไม่สำเร็จ')
    }
  }

  // Export เป็น Excel (.xlsx) แบบจัด Format ให้เหมือนฝั่ง QA — Freeze Header, Header สี Theme
  // ตัวหนากึ่งกลาง, Filter ทุกคอลัมน์ (Excel Table), แถบสีสลับแถว, ปรับความกว้างอัตโนมัติ,
  // Border, จัด Alignment ตามชนิดข้อมูล และ Format ตัวเลขเป็น number จริง
  // (สร้างฝั่ง client ด้วย lib/xlsx.js — ตัวเดียวกับ QA — จึงได้หน้าตาตรงกันแน่นอน)
  async function handleExport() {
    setExporting(true)
    setLoadError('')
    try {
      // ดึงข้อมูล "ทั้งหมด" ของ dataset (ไล่ทีละหน้า หน้าละ 500) — export ครบทุกแถวไม่ตัดตามหน้าจอ
      const PAGE = 500
      let all = []
      let cols = []
      let p = 1
      // กันลูปไม่รู้จบ: จำกัดจำนวนหน้าไว้ที่ 2000 หน้า (1,000,000 แถว)
      for (let guard = 0; guard < 2000; guard++) {
        const data = await getUploadData(dataset, undefined, p, PAGE)
        if (p === 1) cols = data?.columns || []
        const batch = data?.rows || []
        all = all.concat(batch)
        const totalPages = data?.totalPages || 1
        if (p >= totalPages || batch.length === 0) break
        p += 1
      }

      if (all.length === 0) {
        setLoadError('ยังไม่มีข้อมูลให้ Export')
        return
      }

      // parse DataJSON ของแต่ละแถวเป็น object ล่วงหน้า
      const parsed = all.map((row) => {
        try {
          return JSON.parse(row.DataJSON || '{}')
        } catch {
          return {}
        }
      })

      // ตรวจว่าคอลัมน์ไหน "เป็นตัวเลขล้วน" เพื่อจัดเป็น number (Format ตามชนิดข้อมูล)
      // เงื่อนไข: ทุกค่าที่ไม่ว่างต้องเป็นตัวเลข และไม่ใช่รหัสที่ต้องคงเลข 0 นำหน้า/ยาวเกิน 15 หลัก
      const numericByCol = cols.map((label) => {
        let sawValue = false
        for (const obj of parsed) {
          const raw = obj[label]
          if (raw == null || String(raw).trim() === '') continue
          sawValue = true
          const s = String(raw).trim().replace(/,/g, '')
          if (!/^-?\d+(\.\d+)?$/.test(s)) return false
          // เลขยาว (เช่น IMEI/Serial 12–15 หลัก) หรือมี 0 นำหน้า ให้คงเป็นข้อความ กัน Excel แปลงเพี้ยน
          if (s.length > 11 || /^0\d/.test(s)) return false
        }
        return sawValue
      })

      const columns = [
        { key: '_no', header: '#', type: 'number', width: 6 },
        ...cols.map((label, i) => ({
          key: `c${i}`,
          header: label,
          type: numericByCol[i] ? 'number' : 'text',
        })),
      ]

      const rows = parsed.map((obj, idx) => {
        const out = { _no: all[idx]?.RowNo || idx + 1 }
        cols.forEach((label, i) => {
          const raw = obj[label]
          if (numericByCol[i]) {
            const s = raw == null ? '' : String(raw).trim().replace(/,/g, '')
            out[`c${i}`] = s === '' ? null : Number(s)
          } else {
            out[`c${i}`] = raw == null || String(raw).trim() === '' ? '' : String(raw)
          }
        })
        return out
      })

      const sheetName = (label || 'Data').slice(0, 31)
      const blob = buildStyledXlsxBlob({ sheetName, columns, rows })
      const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
      downloadBlob(blob, `${dataset}-export-${stamp}.xlsx`)
    } catch (err) {
      setLoadError(err.message || 'Export ไม่สำเร็จ')
    } finally {
      setExporting(false)
    }
  }

  // ปั๊มตาราง Assembly อัตโนมัติจาก Planning / WH1 / Engine + ทะเบียนกลาง
  async function handleGenerate() {
    if (generating) return
    setGenerating(true)
    try {
      const res = await generateAssembly()
      const created = res?.created ?? 0
      const updated = res?.updated ?? 0
      const skipped = res?.skipped ?? 0
      toastSuccess(
        `ปั๊ม Assembly สำเร็จ — เพิ่มใหม่ ${created}, อัปเดต ${updated}, ไม่เปลี่ยน ${skipped} (จาก ${res?.machines ?? 0} เครื่อง)`
      )
      setLocalReload((n) => n + 1)
    } catch (err) {
      toastError(err.message || 'ปั๊ม Assembly ไม่สำเร็จ')
    } finally {
      setGenerating(false)
    }
  }

  function cellValue(row, colName) {
    try {
      const data = JSON.parse(row.DataJSON || '{}')
      return data[colName] ?? ''
    } catch {
      return ''
    }
  }

  return (
    <>
      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 17 }}>
            {total.toLocaleString()} รายการ
          </h2>
        </div>
        <div className="uv-list-tools" style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <input
            className="wh-search"
            placeholder="ค้นหา (เลขเครื่อง / LOT / Order / Parts)"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            style={{ minWidth: 240 }}
          />
          <button className="wh-issue-btn" onClick={handleExport} disabled={exporting}>
            {exporting ? 'กำลัง Export...' : (
              <>
                <ArrowDownTrayIcon className="size-4" /> Export Excel
              </>
            )}
          </button>
          {dataset === 'assembly' && (
            <button
              className="wh-modal-confirm"
              onClick={handleGenerate}
              disabled={generating}
              title="ระบบปั๊มตาราง Assembly ให้อัตโนมัติตอนเปิดหน้าอยู่แล้ว — กดปุ่มนี้เพื่อดึงข้อมูลล่าสุดซ้ำ (จาก Planning / WH1 / Engine + ทะเบียนกลาง จับคู่ด้วยหมายเลขเครื่อง)"
            >
              {generating ? 'กำลังปั๊ม...' : 'Stamp Assembly'}
            </button>
          )}
          <button className="qa-fail-btn" onClick={handleClear} disabled={total === 0}>
            ล้างทั้งหมด
          </button>
        </div>
      </div>

      <div className="wh-table-card ud-table-scroll">
        <table className="wh-table ud-table">
          <thead>
            <tr>
              <th className="ud-th-sticky">#</th>
              {columns.map((c) => (
                <th key={c}>{c}</th>
              ))}
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={columns.length + 2} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              rows.map((row, i) => (
                <tr key={row.ID}>
                  <td className="ud-td-sticky">{row.RowNo || (page - 1) * UD_PAGE_SIZE + i + 1}</td>
                  {columns.map((c) => (
                    <td key={c} data-label={c}>
                      {cellValue(row, c) || DASH}
                    </td>
                  ))}
                  <td className="wh-cell-action">
                    <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                      <button className="wh-issue-btn" onClick={() => setEditRow(row)}>
                        แก้ไข
                      </button>
                      <button className="qa-fail-btn" onClick={() => handleDelete(row.ID)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={columns.length + 2} className="wh-empty-cell">
                  ยังไม่มีรายการที่อัปโหลด
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div
          className="ud-pager"
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 12,
            marginTop: 12,
          }}
        >
          <button
            className="wh-issue-btn"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1 || loading}
          >
            ก่อนหน้า
          </button>
          <span style={{ fontSize: 14 }}>
            หน้า {page.toLocaleString()} / {totalPages.toLocaleString()}
          </span>
          <button
            className="wh-issue-btn"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages || loading}
          >
            ถัดไป
          </button>
        </div>
      )}

      {editRow && (
        <UploadRowEditModal
          row={editRow}
          columns={columns}
          datasetLabel={label}
          onClose={() => setEditRow(null)}
          onSave={handleSaveEdit}
        />
      )}
    </>
  )
}

// เลขรหัสทุกช่องใช้ฟอนต์ monospace เพื่อให้นับหลักตอนเทียบกับตัวเครื่องได้ง่าย
const codeStyle = { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }

/* ─────────────────────────────────────────────────────────────────────────
   UploadRowEditModal — แก้ไขข้อมูล 1 แถวของ Planning/WH1/WH2/Engine/Assembly
   แสดงทุกคอลัมน์ของ dataset เป็นช่องกรอก (ค่าเริ่มต้นจาก DataJSON) แล้วบันทึก
   ───────────────────────────────────────────────────────────────────────── */
function UploadRowEditModal({ row, columns, datasetLabel, onClose, onSave }) {
  const initial = useMemo(() => {
    let data = {}
    try {
      data = JSON.parse(row.DataJSON || '{}')
    } catch {
      data = {}
    }
    const out = {}
    columns.forEach((c) => {
      out[c] = data[c] == null ? '' : String(data[c])
    })
    return out
  }, [row, columns])

  const [form, setForm] = useState(initial)
  const [saving, setSaving] = useState(false)

  const set = (col) => (e) => setForm((f) => ({ ...f, [col]: e.target.value }))

  async function submit() {
    setSaving(true)
    try {
      await onSave(row.ID, form)
    } catch (err) {
      toastError(err.message || 'บันทึกไม่สำเร็จ')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="wh-modal-overlay" onClick={onClose}>
      <div
        className="wh-modal"
        style={{ maxWidth: 720, maxHeight: '85vh', overflowY: 'auto' }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="wh-modal-title">แก้ไขข้อมูล {datasetLabel}</h3>

        <div className="fmt-form fmt-form-compact" style={{ marginTop: 12 }}>
          {columns.map((col) => (
            <div className="fmt-field" key={col}>
              <label className="fmt-label">{col}</label>
              <input className="fmt-input" value={form[col] ?? ''} onChange={set(col)} />
            </div>
          ))}
        </div>

        {columns.length === 0 && (
          <p style={{ fontSize: 13, color: '#94a3b8', marginTop: 10 }}>
            ยังไม่มีคอลัมน์ให้แก้ไข
          </p>
        )}

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 16 }}>
          <button className="wh-modal-cancel" onClick={onClose} disabled={saving}>
            ยกเลิก
          </button>
          <button className="wh-modal-confirm" onClick={submit} disabled={saving || columns.length === 0}>
            {saving ? 'กำลังบันทึก...' : 'บันทึก'}
          </button>
        </div>
      </div>
    </div>
  )
}