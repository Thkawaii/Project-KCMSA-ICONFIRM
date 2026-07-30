import { useEffect, useMemo, useRef, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { getMasterData, uploadMasterData, deleteMasterData } from '../api/masterData.js'
import {
  getUploadData,
  uploadDataFile,
  deleteUploadDataRow,
  clearUploadData,
  exportUploadData,
} from '../api/uploadData.js'
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
import '../UploadData.css'

// ประเภทข้อมูลที่เลือก "อัปโหลด" และ "ดูในตาราง" ได้
// it_controller = ทะเบียน Master Data เดิม (อ่านชนิดจากคอลัมน์ในไฟล์, ยึด Serial No.
// เป็นคีย์, อัปโหลดทับได้) ที่เหลือคือไฟล์ Planning/WH1/WH2/Engine ผ่าน /upload-data
const TYPE_OPTIONS = [
  { value: 'it_controller', label: 'IT Controller' },
  { value: 'planning', label: 'Planning' },
  { value: 'wh1', label: 'WH1' },
  { value: 'wh2', label: 'WH2' },
  { value: 'engine', label: 'Engine' },
]

function typeLabel(value) {
  return TYPE_OPTIONS.find((t) => t.value === value)?.label || value
}

const navItems = [
  { to: '/master-data', label: 'ทะเบียน Master Data', icon: <RectangleStackIcon className="size-4" /> },
]

// ค่าที่แสดงแทนช่องว่าง — อะไหล่ชนิดอื่นไม่มี IT Controller no./IMEI
const DASH = '—'

export default function MasterDataPage() {
  // ประเภทที่จะอัปโหลด (เลือกก่อนอัปโหลด) และประเภทที่กำลังดูในตาราง
  const [uploadType, setUploadType] = useState('it_controller')
  const [viewType, setViewType] = useState('it_controller')

  const [pendingFile, setPendingFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [uploadMsg, setUploadMsg] = useState(null)
  const fileInputRef = useRef(null)

  // นับรอบโหลดใหม่ เพื่อสั่ง refresh ตารางหลังอัปโหลด/ลบ (ใช้ร่วมทั้ง master-data และ dataset)
  const [reloadKey, setReloadKey] = useState(0)

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
      if (uploadType === 'it_controller') {
        // ทะเบียน Master Data เดิม — backend อ่านชนิดจากคอลัมน์ในไฟล์เอง, ยึด Serial No. เป็นคีย์
        const result = await uploadMasterData(pendingFile)
        setUploadMsg({
          success: `นำเข้าสำเร็จ — เพิ่มใหม่ ${result.imported} รายการ, อัปเดตของเดิม ${result.updated} รายการ`,
          problems: result.problems || [],
        })
      } else {
        // Planning / WH1 / WH2 / Engine — แทนที่ข้อมูลเดิมของประเภทนั้นทั้งชุด
        const result = await uploadDataFile(uploadType, pendingFile)
        const extra = result.skipped ? ` (ข้าม ${result.skipped} แถว)` : ''
        setUploadMsg({ success: `นำเข้าสำเร็จ ${result.imported} รายการ${extra}` })
      }

      setPendingFile(null)
      if (fileInputRef.current) fileInputRef.current.value = ''

      // อัปโหลดเสร็จแล้วสลับตารางไปดูประเภทที่เพิ่งอัปโหลด แล้วรีโหลด
      setViewType(uploadType)
      setReloadKey((n) => n + 1)
    } catch (err) {
      setUploadMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setUploading(false)
    }
  }

  return (
    <AppShell navItems={navItems} roleLabel="Upload View">
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
                if (fileInputRef.current) fileInputRef.current.value = ''
              }}
              options={TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label }))}
            />
          </div>

          <button className="wh-issue-btn upload-panel-btn" disabled={uploading} onClick={handleUpload}>
            {uploading ? 'กำลังอัปโหลด...' : `อัปโหลด ${typeLabel(uploadType)}`}
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

      {/* ===== ตัวเลือกประเภทที่จะดูในตาราง ===== */}
      <div className="wh-heading-row" style={{ marginTop: 28 }}>
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            รายการ — {typeLabel(viewType)}
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
        <ITControllerView reloadKey={reloadKey} bumpReload={() => setReloadKey((n) => n + 1)} />
      ) : (
        <DatasetView key={viewType} dataset={viewType} reloadKey={reloadKey} />
      )}
    </AppShell>
  )
}

/* ─────────────────────────────────────────────────────────────────────────
   IT Controller = ทะเบียน Master Data เดิม (สรุป + ตาราง + ค้นหา + Export CSV)
   ยกเนื้อในเดิมมาทั้งหมด ไม่แตะพฤติกรรม
   ───────────────────────────────────────────────────────────────────────── */
function ITControllerView({ reloadKey, bumpReload }) {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [deletingId, setDeletingId] = useState(0)

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
    const kw = keyword.trim().toLowerCase()
    if (!kw) return rows
    return rows.filter((row) =>
      [row.Name, row.Model, row.PartNo, row.SerialNo, row.ITControllerNo, row.IMEI]
        .filter(Boolean)
        .some((field) => String(field).toLowerCase().includes(kw)),
    )
  }, [rows, keyword])

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
    const csv = '\uFEFF' + [header, ...body].map((cols) => cols.map(csvCell).join(',')).join('\r\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `master-data-${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
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
            {keyword.trim() && `พบ ${filtered.length} จาก ${rows.length}`}
          </h2>
        </div>
        <div className="uv-list-tools md-list-tools" style={{ display: 'flex', gap: 10 }}>
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
    </>
  )
}

/* ─────────────────────────────────────────────────────────────────────────
   Planning / WH1 / WH2 / Engine — ตารางไดนามิกตามคอลัมน์ที่ backend ส่งมา
   ───────────────────────────────────────────────────────────────────────── */
function DatasetView({ dataset, reloadKey }) {
  const label = typeLabel(dataset)

  const [columns, setColumns] = useState([])
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [exporting, setExporting] = useState(false)
  const [localReload, setLocalReload] = useState(0)

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setLoadError('')
      try {
        const data = await getUploadData(dataset, keyword || undefined)
        if (!cancelled) {
          setColumns(data?.columns || [])
          setRows(data?.rows || [])
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
    // keyword ค้นเมื่อกดปุ่ม/Enter (ผ่าน localReload) ไม่ผูกกับทุกตัวอักษร
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dataset, reloadKey, localReload])

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

  async function handleClear() {
    const ok = await confirmDelete({ text: `ล้างข้อมูล ${label} ทั้งหมด? กู้คืนไม่ได้` })
    if (!ok) return
    try {
      const res = await clearUploadData(dataset)
      setLocalReload((n) => n + 1)
      toastSuccess(`ล้างแล้ว ${res.deleted ?? 0} แถว`)
    } catch (err) {
      toastError(err.message || 'ล้างไม่สำเร็จ')
    }
  }

  async function handleExport() {
    setExporting(true)
    try {
      await exportUploadData(dataset)
    } catch (err) {
      setLoadError(err.message || 'Export ไม่สำเร็จ')
    } finally {
      setExporting(false)
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
            {rows.length} รายการ
          </h2>
        </div>
        <div className="uv-list-tools" style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <input
            className="wh-search"
            placeholder="ค้นหา (เลขเครื่อง / LOT / Order / Parts)"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && setLocalReload((n) => n + 1)}
            style={{ minWidth: 240 }}
          />
          <button className="wh-issue-btn" onClick={() => setLocalReload((n) => n + 1)}>
            ค้นหา
          </button>
          <button className="wh-issue-btn" onClick={handleExport} disabled={exporting}>
            {exporting ? 'กำลัง Export...' : (
              <>
                <ArrowDownTrayIcon className="size-4" /> Export Excel
              </>
            )}
          </button>
          <button className="qa-fail-btn" onClick={handleClear} disabled={rows.length === 0}>
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
                  <td className="ud-td-sticky">{row.RowNo || i + 1}</td>
                  {columns.map((c) => (
                    <td key={c} data-label={c}>
                      {cellValue(row, c) || DASH}
                    </td>
                  ))}
                  <td className="wh-cell-action">
                    <button className="qa-fail-btn" onClick={() => handleDelete(row.ID)}>
                      ลบ
                    </button>
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
    </>
  )
}

// เลขรหัสทุกช่องใช้ฟอนต์ monospace เพื่อให้นับหลักตอนเทียบกับตัวเครื่องได้ง่าย
const codeStyle = { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }

// ครอบด้วย ="..." เพื่อบังคับให้ Excel อ่านเป็นข้อความ ไม่งั้น IMEI 15 หลักจะเพี้ยน
function excelText(value) {
  if (!value) return ''
  return `="${String(value)}"`
}

function csvCell(value) {
  const s = value == null ? '' : String(value)
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}
