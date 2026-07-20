import { useEffect, useMemo, useRef, useState } from 'react'
import {
  getTsfScans,
  getTsfByMachine,
  createTsfScan,
  updateTsfScan,
  uploadPhoto,
  photoUrl,
  getUsers,
} from '../api/tsf.js'
import { getMachineSpecByMachineNo } from '../api/machineSpecLookup.js'
import AppShell from '../components/AppShell.jsx'

const navItems = [
  { to: '/tsf', label: 'Scan & Validate', icon: '⇄' },
  { to: '/tsf/receive', label: 'Receive Material', icon: '📥' },
]

const CATEGORIES = [
  { type: 'it_controller', label: 'IT Controller', icon: '🛰️', needsPN: true, specKey: (s) => `${s.ITController || '-'} / ${s.ITControllerSN || '-'}` },
  { type: 'control_valve', label: 'Control Valve', icon: '🔧', needsPN: false, specKey: (s) => s.ControlValve || '-' },
  { type: 'swing_motor', label: 'Swing Motor', icon: '⚙️', needsPN: false, specKey: (s) => s.SwingMotor || '-' },
  { type: 'motor_propel', label: 'Motor Propel', icon: '🚜', needsPN: false, specKey: (s) => s.MotorPropel || '-' },
  { type: 'pump_assy_hyd', label: 'Pump Assy HYD', icon: '💧', needsPN: false, specKey: (s) => s.PumpAssyHyd || '-' },
]

// TODO: ย้ายไปดึงจาก backend เป็น dropdown ที่แก้ไขได้ ถ้าแผนกมีการเปลี่ยนบ่อย
const DEPARTMENTS = ['สายประกอบ 1', 'สายประกอบ 2', 'QC', 'คลังสินค้า']

export default function TSFOperatorPage() {
  // ===== ประวัติทั้งหมด =====
  const [submissions, setSubmissions] = useState([])
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  // ===== ตารางประวัติ: filter/search/pagination =====
  const [dateTab, setDateTab] = useState('all')
  const [historySearch, setHistorySearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  // ===== Scan-first flow: idle -> select -> capture -> result -> select (loop) =====
  const [scanStep, setScanStep] = useState('idle')
  const [idleValue, setIdleValue] = useState('')
  const [idleError, setIdleError] = useState('')
  const [idleLoading, setIdleLoading] = useState(false)

  const [machineSpec, setMachineSpec] = useState(null)
  const [machineChecks, setMachineChecks] = useState([])

  const [selPart, setSelPart] = useState(null)
  const [selDept, setSelDept] = useState('')
  const [selInspector, setSelInspector] = useState('')
  const [selectError, setSelectError] = useState('')

  const [scanPN, setScanPN] = useState('')
  const [scanSN, setScanSN] = useState('')
  const [photo, setPhoto] = useState(null)
  const [photoPreview, setPhotoPreview] = useState('')
  const [captureError, setCaptureError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const [result, setResult] = useState(null)

  const idleInputRef = useRef(null)
  const pnInputRef = useRef(null)
  const snInputRef = useRef(null)
  const photoInputRef = useRef(null)

  // ===== ประวัติ: ดูรูป / แก้ไข =====
  const [lightboxUrl, setLightboxUrl] = useState('')
  const [editRow, setEditRow] = useState(null)
  const [editMode, setEditMode] = useState(false)
  const [editForm, setEditForm] = useState({ SerialNumber: '', ActualPartNo: '' })
  const [savingEdit, setSavingEdit] = useState(false)
  const [editError, setEditError] = useState('')

  async function loadHistory() {
    setLoading(true)
    setLoadError('')
    try {
      const [scans, userList] = await Promise.all([getTsfScans(), getUsers('TSF')])
      setSubmissions(scans || [])
      setUsers(userList || [])
      if (!selInspector) {
        const savedName = localStorage.getItem('iconfirm_name') || ''
        const match = (userList || []).find((u) => u.Name === savedName)
        setSelInspector(match ? match.Name : '')
      }
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadHistory()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    setPage(1)
  }, [dateTab, historySearch, pageSize])

  // โฟกัสช่อง input ที่ถูกต้องอัตโนมัติทุกครั้งที่เปลี่ยน step — นี่คือหัวใจ
  // ของ scan-first: เครื่องสแกนเนอร์ยิงไปที่ input ที่ focus อยู่เสมอ
  useEffect(() => {
    if (scanStep === 'idle') {
      idleInputRef.current?.focus()
    } else if (scanStep === 'capture') {
      if (selPart?.needsPN) pnInputRef.current?.focus()
      else snInputRef.current?.focus()
    }
  }, [scanStep, selPart])

  // ===== STEP 1: ยิง Machine No. ที่หน้า SCAN HERE =====
  async function handleIdleSubmit(e) {
    e.preventDefault()
    const machineNo = idleValue.trim()
    if (!machineNo) return

    setIdleLoading(true)
    setIdleError('')
    try {
      const [spec, checks] = await Promise.all([
        getMachineSpecByMachineNo(machineNo),
        getTsfByMachine(machineNo),
      ])
      setMachineSpec(spec)
      setMachineChecks(checks || [])
      setIdleValue('')
      setSelPart(null)
      setScanStep('select')
    } catch (err) {
      setIdleError(err.message || 'ไม่พบข้อมูลเครื่องนี้')
      setIdleValue('')
    } finally {
      setIdleLoading(false)
    }
  }

  function checksFor(type) {
    return machineChecks.filter((c) => c.ComponentType === type)
  }

  // ===== STEP 2: popup เลือก Part / Machine / แผนก / พนักงาน =====
  function confirmSelection() {
    setSelectError('')
    if (!selPart) {
      setSelectError('กรุณาเลือก Part')
      return
    }
    if (!selDept) {
      setSelectError('กรุณาเลือกแผนก')
      return
    }
    if (!selInspector) {
      setSelectError('กรุณาเลือกพนักงาน')
      return
    }
    setScanPN('')
    setScanSN('')
    setPhoto(null)
    setPhotoPreview('')
    setCaptureError('')
    setScanStep('capture')
  }

  function changeMachine() {
    setMachineSpec(null)
    setMachineChecks([])
    setSelPart(null)
    setScanStep('idle')
  }

  // ===== STEP 3: ยิง P/N + S/N (auto-advance) แล้วถ่ายรูป (auto-submit) =====
  function handlePnKeyDown(e) {
    if (e.key !== 'Enter') return
    e.preventDefault()
    if (!scanPN.trim()) {
      setCaptureError('กรุณายิงหรือกรอก P/N')
      return
    }
    setCaptureError('')
    snInputRef.current?.focus()
  }

  function handleSnKeyDown(e) {
    if (e.key !== 'Enter') return
    e.preventDefault()
    if (!scanSN.trim()) {
      setCaptureError('กรุณายิงหรือกรอก S/N')
      return
    }
    setCaptureError('')
    // ยิง S/N เสร็จ -> เปิดกล้องถ่ายรูปให้อัตโนมัติทันที (ขั้นตอนเดียวที่ต้องใช้คน)
    photoInputRef.current?.click()
  }

  function handlePhotoChange(e) {
    const file = e.target.files?.[0]
    if (!file) return
    setPhoto(file)
    const preview = URL.createObjectURL(file)
    setPhotoPreview(preview)
    // ถ่ายรูปเสร็จ -> auto-submit ทันที ไม่ต้องกดปุ่มบันทึกเอง
    submitScan(file)
  }

  async function submitScan(fileArg) {
    if (selPart.needsPN && !scanPN.trim()) {
      setCaptureError('กรุณายิงหรือกรอก P/N')
      return
    }
    if (!scanSN.trim()) {
      setCaptureError('กรุณายิงหรือกรอก S/N')
      return
    }
    const fileToUpload = fileArg || photo
    if (!fileToUpload) {
      setCaptureError('กรุณาถ่ายรูป Part ที่ติดตั้ง')
      return
    }

    setSubmitting(true)
    setCaptureError('')
    try {
      const uploaded = await uploadPhoto(fileToUpload)

      const response = await createTsfScan({
        MachineNo: machineSpec.MachineNo,
        ComponentType: selPart.type,
        Department: selDept,
        SerialNumber: scanSN.trim(),
        ActualPartNo: scanPN.trim(),
        InspectedBy: selInspector,
        FileName: uploaded.file_name,
        PhotoURL: uploaded.url,
      })

      setResult(response)
      setScanStep('result')
    } catch (err) {
      setCaptureError(err.message || 'บันทึกไม่สำเร็จ')
    } finally {
      setSubmitting(false)
    }
  }

  // ===== STEP 4: ผลลัพธ์ -> วนกลับไปเลือก Part ถัดไปของเครื่องเดิม =====
  async function backToSelectNext() {
    try {
      const checks = await getTsfByMachine(machineSpec.MachineNo)
      setMachineChecks(checks || [])
    } catch {
      // เงียบไว้ — รอบหน้าจะรีเฟรชอีกที
    }
    await loadHistory()
    setResult(null)
    setSelPart(null)
    setScanStep('select')
  }

  useEffect(() => {
    if (scanStep === 'result' && result?.pass) {
      const t = setTimeout(backToSelectNext, 2200)
      return () => clearTimeout(t)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scanStep, result])

  // ===== ประวัติ: ดูรูป / แก้ไข =====
  function openEdit(row) {
    setEditRow(row)
    setEditMode(false)
    setEditForm({ SerialNumber: row.SerialNumber, ActualPartNo: row.ActualPartNo })
    setEditError('')
  }

  async function saveEdit() {
    setSavingEdit(true)
    setEditError('')
    try {
      await updateTsfScan(editRow.ID, editForm)
      setEditRow(null)
      setEditMode(false)
      await loadHistory()
    } catch (err) {
      setEditError(err.message || 'บันทึกไม่สำเร็จ')
    } finally {
      setSavingEdit(false)
    }
  }

  const categoryCounts = useMemo(() => {
    const counts = {}
    for (const cat of CATEGORIES) {
      counts[cat.type] = submissions.filter((s) => s.ComponentType === cat.type).length
    }
    return counts
  }, [submissions])

  const filteredHistory = useMemo(() => {
    const now = new Date()
    let rows = submissions

    if (dateTab !== 'all') {
      rows = rows.filter((s) => {
        const d = new Date(s.UploadDate)
        const diffDays = (now - d) / (1000 * 60 * 60 * 24)
        if (dateTab === 'day') return diffDays <= 1
        if (dateTab === 'week') return diffDays <= 7
        if (dateTab === 'month') return diffDays <= 31
        return true
      })
    }

    const term = historySearch.trim().toLowerCase()
    if (term) {
      rows = rows.filter(
        (s) =>
          (s.MachineNo || '').toLowerCase().includes(term) ||
          (s.InspectedBy || '').toLowerCase().includes(term) ||
          (s.SerialNumber || '').toLowerCase().includes(term)
      )
    }

    return rows
  }, [submissions, dateTab, historySearch])

  const totalPages = Math.max(1, Math.ceil(filteredHistory.length / pageSize))
  const pagedHistory = filteredHistory.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  return (
    <>
    <AppShell navItems={navItems} roleLabel="TSF Operator">
        <div className="wh-heading-row">
          <div>
            <h2 className="wh-title">TSF Operator — สแกนและบันทึกข้อมูลชิ้นส่วน</h2>
            <p className="wh-subtitle">ยิงบาร์โค้ด Machine No. เพื่อเริ่ม — ระบบขับเคลื่อนด้วยเครื่องสแกนเนอร์เป็นหลัก</p>
          </div>
        </div>

        {loadError && (
          <p className="form-error" role="alert">
            {loadError}
          </p>
        )}

        {/* ===== STEP 1: SCAN HERE (idle) ===== */}
        {scanStep === 'idle' && (
          <div className="scan-hero">
            <div className="scan-hero-graphic">
              <BigBarcode />
              <div className="scan-hero-label">SCAN HERE</div>
            </div>
            <p className="scan-hero-hint">ยิงบาร์โค้ด Machine No. บนตัวเครื่องเพื่อเริ่มตรวจสอบ</p>
            <form onSubmit={handleIdleSubmit} className="scan-hero-form">
              <input
                ref={idleInputRef}
                className="scan-hero-input"
                type="text"
                placeholder="รอรับสัญญาณจากเครื่องสแกน..."
                value={idleValue}
                onChange={(e) => setIdleValue(e.target.value)}
                autoFocus
              />
            </form>
            {idleLoading && <p className="wh-subtitle">กำลังโหลดข้อมูลเครื่อง...</p>}
            {idleError && (
              <p className="form-error" role="alert">
                {idleError}
              </p>
            )}
          </div>
        )}

        {/* ===== STEP 2: popup เลือก Part / Machine / แผนก / พนักงาน ===== */}
        {scanStep === 'select' && machineSpec && (
          <div className="wh-modal-overlay">
            <div className="wh-modal" style={{ maxWidth: 420 }}>
              <h3 className="wh-modal-title">เลือกรายการตรวจสอบ</h3>
              <p className="wh-modal-line">
                Machine No: <strong>{machineSpec.MachineNo}</strong>
              </p>

              <label className="wh-modal-label">Part</label>
              <select
                className="wh-modal-input"
                value={selPart?.type || ''}
                onChange={(e) => setSelPart(CATEGORIES.find((c) => c.type === e.target.value) || null)}
              >
                <option value="">-- เลือก Part --</option>
                {CATEGORIES.map((cat) => {
                  const expected = cat.specKey(machineSpec)
                  const notEquipped = expected.replace(/\s|\/|-/g, '') === ''
                  const checks = checksFor(cat.type)
                  return (
                    <option key={cat.type} value={cat.type} disabled={notEquipped}>
                      {cat.icon} {cat.label} {notEquipped ? '(ไม่มีติดตั้ง)' : checks[0] ? `(ตรวจแล้ว: ${checks[0].ValidationStatus})` : ''}
                    </option>
                  )
                })}
              </select>

              <label className="wh-modal-label">Machine No.</label>
              <input className="wh-modal-input" type="text" value={machineSpec.MachineNo} disabled />

              <label className="wh-modal-label">แผนก</label>
              <select className="wh-modal-input" value={selDept} onChange={(e) => setSelDept(e.target.value)}>
                <option value="">-- เลือกแผนก --</option>
                {DEPARTMENTS.map((d) => (
                  <option key={d} value={d}>
                    {d}
                  </option>
                ))}
              </select>

              <label className="wh-modal-label">พนักงานที่ตรวจสอบ</label>
              <select className="wh-modal-input" value={selInspector} onChange={(e) => setSelInspector(e.target.value)}>
                <option value="">-- เลือกพนักงาน --</option>
                {users.map((u) => (
                  <option key={u.ID} value={u.Name}>
                    {u.Name}
                  </option>
                ))}
              </select>

              {selectError && (
                <p className="form-error" role="alert">
                  {selectError}
                </p>
              )}

              <div className="wh-modal-actions">
                <button className="wh-modal-cancel" onClick={changeMachine}>
                  เปลี่ยนเครื่อง
                </button>
                <button className="wh-modal-confirm" onClick={confirmSelection}>
                  ถัดไป — ยิง P/N &amp; S/N
                </button>
              </div>
            </div>
          </div>
        )}

        {/* ===== STEP 3: ยิง P/N + S/N แล้วถ่ายรูป (auto-submit) ===== */}
        {scanStep === 'capture' && selPart && machineSpec && (
          <div className="scan-hero">
            <div className="scan-capture-card">
              <h3 className="tsf-form-title">
                {selPart.icon} {selPart.label} — Machine {machineSpec.MachineNo}
              </h3>

              {selPart.needsPN && (
                <>
                  <label className="wh-modal-label">1. ยิง P/N</label>
                  <input
                    ref={pnInputRef}
                    className="scan-capture-input"
                    type="text"
                    placeholder="รอรับสัญญาณ P/N..."
                    value={scanPN}
                    onChange={(e) => setScanPN(e.target.value)}
                    onKeyDown={handlePnKeyDown}
                  />
                  {scanPN && <span className="scan-capture-check">✓ {scanPN}</span>}
                </>
              )}

              <label className="wh-modal-label">{selPart.needsPN ? '2. ยิง S/N' : '1. ยิง S/N'}</label>
              <input
                ref={snInputRef}
                className="scan-capture-input"
                type="text"
                placeholder="รอรับสัญญาณ S/N..."
                value={scanSN}
                onChange={(e) => setScanSN(e.target.value)}
                onKeyDown={handleSnKeyDown}
              />
              {scanSN && <span className="scan-capture-check">✓ {scanSN}</span>}

              <label className="wh-modal-label">{selPart.needsPN ? '3.' : '2.'} ถ่ายรูป (เปิดกล้องอัตโนมัติหลังยิง S/N)</label>
              <input
                ref={photoInputRef}
                type="file"
                accept="image/*"
                capture="environment"
                onChange={handlePhotoChange}
                className="upload-card-input-hidden"
                id="capturePhoto"
              />
              <label htmlFor="capturePhoto" className={'upload-dropzone' + (photo ? ' upload-dropzone-filled' : '')}>
                <CameraPlusIcon />
                <span className="upload-dropzone-text">{photo ? photo.name : 'แตะเพื่อถ่ายรูป (หรือรอเปิดอัตโนมัติ)'}</span>
              </label>
              {photoPreview && <img className="tsf-photo-preview" src={photoPreview} alt="preview" />}

              {captureError && (
                <p className="form-error" role="alert">
                  {captureError}
                </p>
              )}
              {submitting && <p className="wh-subtitle">กำลังบันทึก...</p>}

              <div className="wh-modal-actions" style={{ justifyContent: 'space-between' }}>
                <button className="wh-modal-cancel" onClick={() => setScanStep('select')} disabled={submitting}>
                  ย้อนกลับ
                </button>
                <button className="wh-modal-confirm" onClick={() => submitScan()} disabled={submitting}>
                  {submitting ? 'กำลังบันทึก...' : 'บันทึก (ถ้ายังไม่ auto)'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* ===== STEP 4: ผลลัพธ์ ===== */}
        {scanStep === 'result' && result && (
          <div className="scan-hero">
            <div className={'scan-result-card' + (result.pass ? ' scan-result-pass' : ' scan-result-fail')}>
              {result.pass ? (
                <>
                  <div className="scan-result-icon">✅</div>
                  <h3 className="scan-result-title">PASS</h3>
                  <p className="wh-subtitle">ข้อมูลตรงกับ Master Data — ส่งข้อมูลตรวจสอบไปยัง QA แล้ว</p>
                  <p className="wh-subtitle">กำลังไปหน้ารายการถัดไปอัตโนมัติ...</p>
                </>
              ) : (
                <>
                  <div className="scan-result-icon">❌</div>
                  <h3 className="scan-result-title">FAIL</h3>
                  <p className="form-error">{result.mismatch_detail}</p>
                  <button className="wh-modal-confirm" onClick={backToSelectNext}>
                    RE-SCAN
                  </button>
                </>
              )}
            </div>
          </div>
        )}

        {/* ===== ประวัติทั้งหมด ===== */}
        <div className="wh-heading-row" style={{ marginTop: 32 }}>
          <div>
            <h2 className="wh-title" style={{ fontSize: 19 }}>
              รายการที่ส่งแล้ว ({filteredHistory.length})
            </h2>
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
            placeholder="ค้นหา Machine No / ผู้ตรวจสอบ / S/N"
            value={historySearch}
            onChange={(e) => setHistorySearch(e.target.value)}
          />
        </div>

        <div className="wh-table-card">
          <table className="wh-table">
            <thead>
              <tr>
                <th>Machine No</th>
                <th>Part</th>
                <th>แผนก</th>
                <th>S/N</th>
                <th>P/N</th>
                <th>ผู้ตรวจสอบ</th>
                <th>วันที่</th>
                <th>รูปภาพ</th>
                <th>สถานะ</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {loading && (
                <tr>
                  <td colSpan={10} className="wh-empty-cell">
                    กำลังโหลดข้อมูล...
                  </td>
                </tr>
              )}
              {!loading &&
                pagedHistory.map((s) => (
                  <tr key={s.ID}>
                    <td>{s.MachineNo || '—'}</td>
                    <td>{CATEGORIES.find((c) => c.type === s.ComponentType)?.label || s.ComponentType}</td>
                    <td>{s.Department || '—'}</td>
                    <td>{s.SerialNumber}</td>
                    <td>{s.ActualPartNo || '—'}</td>
                    <td>{s.InspectedBy}</td>
                    <td>{s.UploadDate ? new Date(s.UploadDate).toLocaleString('th-TH') : '—'}</td>
                    <td>
                      {s.PhotoURL ? (
                        <button className="tsf-thumb-btn" onClick={() => setLightboxUrl(photoUrl(s.PhotoURL))}>
                          <img className="tsf-thumb" src={photoUrl(s.PhotoURL)} alt={s.FileName} />
                        </button>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td>
                      <span
                        className={
                          s.ValidationStatus === 'PASS' ? 'wh-qty-badge' : 'wh-qty-badge wh-qty-zero'
                        }
                      >
                        {s.ValidationStatus}
                      </span>
                    </td>
                    <td>
                      <button className="tsf-action-btn" onClick={() => openEdit(s)}>
                        แก้ไข
                      </button>
                    </td>
                  </tr>
                ))}
              {!loading && pagedHistory.length === 0 && (
                <tr>
                  <td colSpan={10} className="wh-empty-cell">
                    ไม่พบรายการ
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {!loading && filteredHistory.length > 0 && (
          <div className="tsf-pagination">
            <span className="wh-subtitle" style={{ fontSize: 13 }}>
              Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filteredHistory.length)} of{' '}
              {filteredHistory.length} entries
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
              <button className="wh-modal-cancel" onClick={() => goToPage(page + 1)} disabled={page === totalPages}>
                ›
              </button>
              <button className="wh-modal-cancel" onClick={() => goToPage(totalPages)} disabled={page === totalPages}>
                »
              </button>
            </div>
          </div>
        )}
    </AppShell>

    {lightboxUrl && (
      <div className="wh-modal-overlay" onClick={() => setLightboxUrl('')}>
        <div className="lightbox-content" onClick={(e) => e.stopPropagation()}>
          <img src={lightboxUrl} alt="รูปยืนยันการติดตั้ง" />
          <button className="wh-modal-cancel" style={{ marginTop: 14 }} onClick={() => setLightboxUrl('')}>
            ปิด
          </button>
        </div>
      </div>
    )}

    {editRow && (
      <div className="wh-modal-overlay" onClick={() => setEditRow(null)}>
        <div className="tsf-detail-modal" onClick={(e) => e.stopPropagation()}>
          <h3 className="wh-modal-title">
            {editMode ? 'แก้ไข' : 'รายละเอียด'} Machine: {editRow.MachineNo}
          </h3>

          <div className="tsf-detail-grid">
            <div className="tsf-detail-col">
              <label className="wh-modal-label">Machine No.</label>
              <div className="tsf-detail-readbox">{editRow.MachineNo || '-'}</div>

              <label className="wh-modal-label">Part</label>
              <div className="tsf-detail-readbox">
                {CATEGORIES.find((c) => c.type === editRow.ComponentType)?.label || editRow.ComponentType}
              </div>

              <label className="wh-modal-label">Serial Number (S/N)</label>
              {editMode ? (
                <input
                  className="wh-modal-input"
                  type="text"
                  value={editForm.SerialNumber}
                  onChange={(e) => setEditForm((prev) => ({ ...prev, SerialNumber: e.target.value }))}
                />
              ) : (
                <div className="tsf-detail-readbox">{editRow.SerialNumber || '-'}</div>
              )}

              <label className="wh-modal-label">Part Number (P/N)</label>
              {editMode ? (
                <input
                  className="wh-modal-input"
                  type="text"
                  value={editForm.ActualPartNo}
                  onChange={(e) => setEditForm((prev) => ({ ...prev, ActualPartNo: e.target.value }))}
                />
              ) : (
                <div className="tsf-detail-readbox">{editRow.ActualPartNo || '-'}</div>
              )}
            </div>

            <div className="tsf-detail-col">
              <label className="wh-modal-label">แผนก</label>
              <div className="tsf-detail-readbox">{editRow.Department || '-'}</div>

              <label className="wh-modal-label">ผู้ตรวจสอบ</label>
              <div className="tsf-detail-readbox">{editRow.InspectedBy || '-'}</div>

              <label className="wh-modal-label">วันที่ตรวจสอบ</label>
              <div className="tsf-detail-readbox">
                {editRow.UploadDate ? new Date(editRow.UploadDate).toLocaleString('th-TH') : '-'}
              </div>

              <label className="wh-modal-label">ผลตรวจสอบ</label>
              <div className="tsf-detail-readbox">
                <span
                  className={
                    editRow.ValidationStatus === 'PASS' ? 'qa-result-badge qa-pass' : 'qa-result-badge qa-fail'
                  }
                >
                  {editRow.ValidationStatus}
                </span>
              </div>

              {editRow.PhotoURL && (
                <img className="tsf-detail-photo" src={photoUrl(editRow.PhotoURL)} alt="รูปยืนยันการติดตั้ง" />
              )}
            </div>
          </div>

          {editError && (
            <p className="form-error" role="alert">
              {editError}
            </p>
          )}

          <div className="wh-modal-actions">
            <button className="wh-modal-cancel" onClick={() => setEditRow(null)} disabled={savingEdit}>
              กลับ
            </button>
            {editMode ? (
              <button className="wh-modal-confirm" onClick={saveEdit} disabled={savingEdit}>
                {savingEdit ? 'กำลังบันทึก...' : 'บันทึก'}
              </button>
            ) : (
              <button className="wh-modal-confirm" onClick={() => setEditMode(true)}>
                ✎ แก้ไข
              </button>
            )}
          </div>
        </div>
      </div>
    )}
    </>
  )
}

function BigBarcode() {
  const bars = [3, 1, 2, 1, 3, 2, 1, 1, 2, 3, 1, 2, 1, 3, 1, 1, 2, 1, 3, 2, 1, 3, 1, 2, 1, 1, 3, 2, 1, 2]
  let x = 0
  return (
    <svg viewBox="0 0 320 90" className="scan-hero-barcode">
      {bars.map((w, i) => {
        const bx = x
        x += w * 4 + 3
        return <rect key={i} x={bx} y={0} width={w * 3} height={70} fill="currentColor" />
      })}
    </svg>
  )
}

function CameraPlusIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M4 8a2 2 0 012-2h1.2l.8-1.4A1 1 0 018.86 4h6.28a1 1 0 01.86.6L16.8 6H18a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2V8z"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="13" r="3" stroke="currentColor" strokeWidth="1.6" />
      <path d="M12 20v0" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  )
}