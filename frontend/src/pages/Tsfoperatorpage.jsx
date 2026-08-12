import { useEffect, useMemo, useRef, useState } from 'react'
import {
  getMFGAssemblies,
  scanMFGAssembly,
  updateMFGAssembly,
  deleteMFGAssembly,
} from '../api/mfgAssembly.js'
import { confirmDelete, toastSuccess, toastError } from '../lib/toast.js'
import { scanStep, scanLoading, scanClose, scanSuccessToast, scanErrorAlert } from '../lib/scanPopup.js'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import bcMachine from '../assets/barcodes/Machine_Barcode.gif'
import {
  ArrowsRightLeftIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from '../components/icons.jsx'

// MFG มีหน้าเดียว — AppShell จะซ่อนแถบเมนูย่อยให้เองเมื่อมีรายการเดียว
export const MFG_NAV_ITEMS = [
  { to: '/mfg-assembly', label: 'MFG Assembly', icon: <ArrowsRightLeftIcon className="size-4" />, roles: ['MFG'] },
  { to: '/tsf', label: 'Scan & Validate', icon: <ArrowsRightLeftIcon className="size-4" />, roles: ['TSF'] },
]

// ป้ายสถานะ — เหลือ 3 แบบ: Matched / Not Matched / Duplicate (ซ้ำ)
const STATUS_META = {
  MATCHED: { label: 'Matched', cls: 'il-badge il-badge-ok' },
  NOT_MATCHED: { label: 'Not Matched', cls: 'il-badge il-badge-bad' },
  DUPLICATE: { label: 'Duplicate (ซ้ำ)', cls: 'il-badge il-badge-warn' },
}

const STATUS_OPTIONS = [
  { value: 'MATCHED', label: 'Matched' },
  { value: 'NOT_MATCHED', label: 'Not Matched' },
  { value: 'DUPLICATE', label: 'Duplicate (ซ้ำ)' },
]

const EMPTY_FORM = {
  item: '',
  dateAssembly: '',
  machineNo: '',
  itControllerNo: '',
  country: '',
  checkDate: '',
  status: '',
}

// วันที่-เวลา แสดงผล (รับ ISO string / null) เช่น 6/8/2569 14:35:32
function fmtDate(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  const buddhistYear = d.getFullYear() + 543
  const day = d.getDate()
  const month = d.getMonth() + 1
  const pad = (n) => String(n).padStart(2, '0')
  const time = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  return `${day}/${month}/${buddhistYear} ${time}`
}

// แปลงเป็นค่าใส่ <input type="date"> (yyyy-mm-dd)
function toDateInput(value) {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${m}-${day}`
}

// แยกค่าที่ยิง/พิมพ์เข้ามา
// - ปกติป้าย Machine Part Confirmation จะมีแค่ "หมายเลขเครื่อง" (เช่น LX10400690)
//   -> ระบบจะไปดึง IT Controller No. ให้เองที่ backend
// - เผื่อ QR บรรจุทั้งคู่: โทเคนตัวเลขล้วน 10–15 หลัก = IT Controller No.
function parseAssemblyCode(raw) {
  const s = (raw || '').trim()
  if (!s) return { machineNo: '', itControllerNo: '' }
  const tokens = s.split(/[\s,;|]+/).map((t) => t.trim()).filter(Boolean)
  const itc = tokens.find((t) => /^\d{10,15}$/.test(t)) || ''
  const mc = tokens.find((t) => t !== itc) || ''
  if (itc || mc) return { machineNo: mc, itControllerNo: itc }
  return { machineNo: tokens[0] || '', itControllerNo: tokens[1] || '' }
}

export default function TSFOperatorPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('') // '' = ทั้งหมด
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  // ── โมดัลแก้ไข ─────────────────────────────────────────────────────────
  const [modalOpen, setModalOpen] = useState(false)
  const [editId, setEditId] = useState(null)
  const [form, setForm] = useState(EMPTY_FORM)
  const [saving, setSaving] = useState(false)

  const busyRef = useRef(false) // กันเปิด popup สแกนซ้อน
  const fireRef = useRef(() => {}) // ตัวรับสัญญาณจากเครื่องสแกน (ตั้งค่าใหม่ทุกเรนเดอร์)

  function friendlyError(err, fallback) {
    if (err?.status === 404 || err?.status === 405) {
      return 'ยังไม่พบ API /mfg-assembly ที่ฝั่งเซิร์ฟเวอร์ — ต้อง rebuild แล้ว restart backend ก่อน'
    }
    return err?.message || fallback
  }

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const list = await getMFGAssemblies()
      setRows(list || [])
    } catch (err) {
      setLoadError(friendlyError(err, 'โหลดข้อมูลไม่สำเร็จ'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRows()
  }, [])

  useEffect(() => {
    setPage(1)
  }, [search, statusFilter, pageSize])

  // ── ตัวดักสัญญาณเครื่องสแกนระดับหน้าเว็บ (แบบเดียวกับหน้า WH) ──────────────
  // เครื่องสแกน = คีย์บอร์ดที่พิมพ์เร็วมาก (เว้นแต่ละตัว < ~50ms) แล้วปิดท้ายด้วย Enter
  // จับจาก "ความเร็วการยิง" จึงเด้ง flow ได้ไม่ว่าโฟกัสจะอยู่ตรงไหน (รวมถึงตอนที่
  // เคอร์เซอร์ค้างในช่องค้นหา) + กันไม่ให้ตัวอักษรบาร์โค้ดตกลงช่องค้นหา/ช่องอื่น
  useEffect(() => {
    let buffer = ''
    let lastTime = 0
    let flushTimer = null
    let startedClean = false

    function fireBuffered() {
      if (flushTimer) {
        clearTimeout(flushTimer)
        flushTimer = null
      }
      const code = buffer.trim()
      const clean = startedClean
      buffer = ''
      startedClean = false
      if (busyRef.current) return // มี popup/flow เปิดอยู่ -> ให้ช่องใน popup รับเอง
      if (clean && code.length >= 2) fireRef.current(code)
    }

    function onKeydown(e) {
      // ระหว่าง flow กำลังทำงาน (popup เปิด): ล้าง buffer ทิ้ง ให้ช่องใน popup รับเอง
      if (busyRef.current) {
        lastTime = Date.now()
        buffer = ''
        startedClean = false
        return
      }

      const now = Date.now()
      const gap = now - lastTime
      lastTime = now
      if (gap > 50) {
        // เว้นเกิน 50ms = เริ่มยิงชุดใหม่จากจังหวะว่าง -> เริ่มนับใหม่แบบ "สะอาด"
        buffer = ''
        startedClean = true
      }

      if (e.key === 'Enter') {
        if (startedClean && buffer.trim().length >= 2) {
          e.preventDefault()
          fireBuffered()
        } else {
          buffer = ''
          startedClean = false
        }
        return
      }

      if (e.key && e.key.length === 1) {
        buffer += e.key
        // ยิงเป็นชุดเร็ว ๆ (บาร์โค้ด) -> ดักไว้ ไม่ให้ตัวอักษรตกลงช่องค้นหา/ช่องอื่น
        if (buffer.length >= 2) e.preventDefault()
        // เผื่อเครื่องสแกนไม่มี Enter suffix: ยิงเองหลังเงียบ ~120ms
        if (flushTimer) clearTimeout(flushTimer)
        flushTimer = setTimeout(fireBuffered, 120)
      }
    }

    // Fallback สำหรับสแกนเนอร์ที่ "วาง" ข้อความทั้งก้อนทีเดียว (Android/PDA/IME)
    // แทนการจำลองปุ่มกดทีละตัว — onKeydown จะไม่เห็นอะไรเลย จึงต้องดัก 'input' เพิ่ม
    function onGlobalInput(e) {
      if (busyRef.current) return
      const inserted = typeof e.data === 'string' ? e.data : ''
      const code = inserted.trim()
      if (code.length < 2) return // ตัวอักษรเดียว -> น่าจะเป็นคนพิมพ์เอง ปล่อยผ่าน

      // เอาข้อความที่เพิ่งแทรกออกจากช่องเดิม กันไปปนกับค่าที่มีอยู่ (เช่น ช่องค้นหา)
      const target = e.target
      if (target && typeof target.value === 'string') {
        try {
          target.value = target.value.slice(0, Math.max(0, target.value.length - inserted.length))
        } catch {
          /* ignore */
        }
      }

      buffer = ''
      startedClean = false
      fireRef.current(code)
    }

    window.addEventListener('keydown', onKeydown)
    window.addEventListener('input', onGlobalInput, true)
    return () => {
      window.removeEventListener('keydown', onKeydown)
      window.removeEventListener('input', onGlobalInput, true)
      if (flushTimer) clearTimeout(flushTimer)
    }
  }, [])

  // ── SCAN FLOW (แบบเดียวกับ WH) ───────────────────────────────────────────
  // คลิกการ์ด -> popup ให้ "ยิงบาร์โค้ด หรือพิมพ์เอง" Machine No แล้วกดปุ่ม
  // -> ระบบดึง IT Controller No. + Country ให้ แล้วขึ้นในตาราง

  // แยกส่วน "บันทึกโค้ดที่ได้" ออกมาใช้ร่วมกัน ทั้งจากการคลิกการ์ด (พิมพ์/ยิงในป๊อปอัป)
  // และจากเครื่องสแกนที่ยิงตรงเข้าหน้าเว็บ (ตัวดักด้านล่าง)
  async function submitAssemblyCode(code) {
    const { machineNo, itControllerNo } = parseAssemblyCode(code)
    if (!machineNo) return

    scanLoading('กำลังบันทึก...')
    try {
      const res = await scanMFGAssembly({ machineNo, itControllerNo: itControllerNo || '' })
      scanClose()
      const msg = res?.message || 'บันทึกแล้ว'
      if (res?.status === 'MATCHED') {
        scanSuccessToast(msg)
      } else {
        // DUPLICATE (ซ้ำ) หรือ NOT_MATCHED — แจ้งเตือนให้ผู้ใช้ตรวจสอบ
        toastError(msg)
      }
      await loadRows()
    } catch (err) {
      scanClose()
      await scanErrorAlert(friendlyError(err, 'บันทึกไม่สำเร็จ'))
    }
  }

  // เปิด popup "ว่าง" ให้ยิง/พิมพ์ Machine No แล้วกด "บันทึก" ค่อยบันทึก (ไม่บันทึกทันที)
  // ใช้ร่วมกันทั้งตอนคลิกการ์ด และตอนเครื่องสแกนยิงเข้าหน้าเว็บ (เหมือนหน้า WH)
  // — ช่องในป๊อปอัปเว้นว่างเสมอ ให้ยิงบาร์โค้ดจริงเข้ามาในนี้ (ไม่ prefill ค่าเดิม)
  async function runScanFlow() {
    if (busyRef.current) return
    busyRef.current = true
    try {
      const code = await scanStep({
        title: 'Machine Part Confirmation',
        placeholder: 'ยิงบาร์โค้ด หรือพิมพ์ Machine No แล้วกดปุ่ม',
        confirmText: 'บันทึก',
      })
      if (!code) return
      await submitAssemblyCode(code)
    } finally {
      busyRef.current = false
    }
  }

  // เครื่องสแกนยิงบาร์โค้ดเข้าหน้าเว็บโดยตรง (ไม่ต้องคลิกการ์ดก่อน)
  // -> เปิด popup "ว่าง" ให้ยิงซ้ำในป๊อปอัปก่อนบันทึก (เหมือนหน้า WH) ไม่บันทึกทันที
  function handleScannerFire() {
    if (busyRef.current) return
    runScanFlow()
  }
  fireRef.current = handleScannerFire

  // ── โมดัล แก้ไข ──────────────────────────────────────────────────────────
  function openEdit(row) {
    setEditId(row.ID)
    setForm({
      item: row.Item || '',
      dateAssembly: toDateInput(row.DateAssembly),
      machineNo: row.MachineNo || '',
      itControllerNo: row.ITControllerNo || '',
      country: row.Country || '',
      checkDate: toDateInput(row.CheckDate),
      status: row.Status || '',
    })
    setModalOpen(true)
  }

  function closeModal() {
    if (saving) return
    setModalOpen(false)
  }

  function setField(key, value) {
    setForm((f) => ({ ...f, [key]: value }))
  }

  async function save() {
    if (!form.machineNo.trim()) {
      toastError('กรุณากรอก Machine No')
      return
    }
    setSaving(true)
    try {
      await updateMFGAssembly(editId, form)
      toastSuccess('แก้ไขรายการแล้ว')
      setModalOpen(false)
      await loadRows()
    } catch (err) {
      toastError(friendlyError(err, 'บันทึกไม่สำเร็จ'))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(row) {
    const label = row.MachineNo || row.ITControllerNo || '#' + row.ID
    const ok = await confirmDelete({ text: `ลบรายการ ${label}? กู้คืนไม่ได้` })
    if (!ok) return
    try {
      await deleteMFGAssembly(row.ID)
      toastSuccess(`ลบรายการ ${label} แล้ว`)
      await loadRows()
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    return rows.filter((r) => {
      // กรองตามสถานะที่เลือก ('' = ทั้งหมด)
      if (statusFilter && (r.Status || '') !== statusFilter) return false
      if (!term) return true
      return (
        (r.Item || '').toLowerCase().includes(term) ||
        (r.MachineNo || '').toLowerCase().includes(term) ||
        (r.ITControllerNo || '').toLowerCase().includes(term) ||
        (r.Country || '').toLowerCase().includes(term) ||
        (r.Status || '').toLowerCase().includes(term) ||
        (r.WHLicenseNo || '').toLowerCase().includes(term) ||
        (r.WHInvoiceNo || '').toLowerCase().includes(term)
      )
    })
  }, [rows, search, statusFilter])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  return (
    <AppShell navItems={MFG_NAV_ITEMS} roleLabel="MFG">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">MFG</h2>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      {/* ── การ์ดบาร์โค้ด: คลิกเพื่อยิง/กรอก Machine No (เหมือนหน้า WH) ── */}
      <div className="pc-barcode-grid pc-barcode-grid--single">
        <div
          className="pc-barcode-card"
          role="button"
          tabIndex={0}
          onClick={() => runScanFlow()}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') runScanFlow()
          }}
        >
          <div className="pc-barcode-title">Machine Part Confirmation</div>
          <div className="pc-barcode-box">
            <img className="pc-barcode-img" src={bcMachine} alt="บาร์โค้ด Machine No." />
          </div>
        </div>
      </div>

      {/* ── ตาราง: รายการที่ส่งแล้ว ──────────────────────────────────────── */}
      <div className="wh-heading-row" style={{ marginTop: 8 }}>
        <div>
          <h3 className="wh-title" style={{ fontSize: 18 }}>
            รายการที่ส่งแล้ว
          </h3>
        </div>
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
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <div className="wh-pagesize-select" style={{ minWidth: 180 }}>
            <SelectField
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: '', label: 'สถานะทั้งหมด' },
                { value: 'MATCHED', label: 'Matched' },
                { value: 'NOT_MATCHED', label: 'Not Matched' },
                { value: 'DUPLICATE', label: 'Duplicate (ซ้ำ)' },
              ]}
            />
          </div>
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหา Item / Machine No / IT Controller / Country / Status"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Item</th>
              <th>Date Ass'y</th>
              <th>Machine No</th>
              <th>IT Controller No.</th>
              <th>Country</th>
              <th>Check Date</th>
              <th>Status</th>
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
              paged.map((a, idx) => (
                <tr key={a.ID}>
                  <td className="wh-cell-head" data-label="Item">
                    <strong>{(page - 1) * pageSize + idx + 1}</strong>
                  </td>
                  <td data-label="Date Ass'y">{fmtDate(a.DateAssembly)}</td>
                  <td className="il-mono" data-label="Machine No">
                    {a.MachineNo || '—'}
                  </td>
                  <td className="il-mono" data-label="IT Controller No.">
                    {a.ITControllerNo || '—'}
                  </td>
                  <td data-label="Country">{a.Country || '—'}</td>
                  <td data-label="Check Date">{fmtDate(a.CheckDate)}</td>
                  <td data-label="Status">
                    {(() => {
                      const meta = STATUS_META[a.Status] || {
                        label: a.Status || '—',
                        cls: 'il-badge il-badge-muted',
                      }
                      return <span className={meta.cls}>{meta.label}</span>
                    })()}
                  </td>
                  <td className="wh-cell-action">
                    <button
                      className="tsf-action-btn tsf-action-btn-danger"
                      onClick={() => handleDelete(a)}
                    >
                      ลบ
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  {rows.length === 0
                    ? 'ยังไม่มีรายการ — คลิกการ์ดด้านบนแล้วยิง/กรอก Machine No'
                    : 'ไม่พบรายการที่ค้นหา'}
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

      {/* ── แก้ไขรายการ ─────────────────────────────────────────────────── */}
      {modalOpen && (
        <div className="wh-modal-overlay" onClick={closeModal}>
          <div className="wh-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">แก้ไขรายการ</h3>

            <label className="wh-modal-label">Item</label>
            <input
              className="wh-modal-input"
              value={form.item}
              onChange={(e) => setField('item', e.target.value)}
            />

            <label className="wh-modal-label">Date Ass'y</label>
            <input
              className="wh-modal-input"
              type="date"
              value={form.dateAssembly}
              onChange={(e) => setField('dateAssembly', e.target.value)}
            />

            <label className="wh-modal-label">Machine No</label>
            <input
              className="wh-modal-input"
              value={form.machineNo}
              onChange={(e) => setField('machineNo', e.target.value)}
              placeholder="เช่น LX10400690"
            />

            <label className="wh-modal-label">IT Controller No.</label>
            <input
              className="wh-modal-input"
              value={form.itControllerNo}
              onChange={(e) => setField('itControllerNo', e.target.value)}
              placeholder="เช่น 878250022802"
            />

            <label className="wh-modal-label">Country</label>
            <input
              className="wh-modal-input"
              value={form.country}
              onChange={(e) => setField('country', e.target.value)}
            />

            <label className="wh-modal-label">Check Date</label>
            <input
              className="wh-modal-input"
              type="date"
              value={form.checkDate}
              onChange={(e) => setField('checkDate', e.target.value)}
            />

            <label className="wh-modal-label">Status</label>
            <SelectField
              value={form.status}
              onChange={(v) => setField('status', v)}
              options={[{ value: '', label: '— ให้ระบบประเมินให้ —' }, ...STATUS_OPTIONS]}
            />

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={closeModal} disabled={saving}>
                ยกเลิก
              </button>
              <button className="wh-modal-confirm" onClick={save} disabled={saving}>
                {saving ? 'กำลังบันทึก...' : 'บันทึก'}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  )
}
