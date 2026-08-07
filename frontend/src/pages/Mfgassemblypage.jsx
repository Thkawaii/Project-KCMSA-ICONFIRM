import { useEffect, useMemo, useState } from 'react'
import {
  getMFGAssemblies,
  scanMFGAssembly,
  createMFGAssembly,
  updateMFGAssembly,
  deleteMFGAssembly,
} from '../api/mfgAssembly.js'
import { confirmDelete, toastSuccess, toastError } from '../lib/toast.js'
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  QrCodeIcon,
} from '../components/icons.jsx'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import BarcodeScannerModal from '../components/Barcodescannermodal.jsx'
import { MFG_NAV_ITEMS } from './Tsfoperatorpage.jsx'

// ป้ายสถานะ — ใช้ชุดคลาส .il-badge เดิม
const STATUS_META = {
  OK: { label: 'ตรงกัน', cls: 'il-badge il-badge-ok' },
  UNKNOWN: { label: 'ไม่พบในทะเบียน', cls: 'il-badge il-badge-warn' },
  REUSED: { label: 'ผูกกับเครื่องอื่น', cls: 'il-badge il-badge-bad' },
  DUPLICATE: { label: 'ซ้ำ', cls: 'il-badge il-badge-warn' },
}

const STATUS_OPTIONS = [
  { value: 'OK', label: 'OK — ตรงกัน' },
  { value: 'UNKNOWN', label: 'UNKNOWN — ไม่พบในทะเบียน' },
  { value: 'REUSED', label: 'REUSED — ผูกกับเครื่องอื่น' },
  { value: 'DUPLICATE', label: 'DUPLICATE — ซ้ำ' },
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

// แยกค่าที่สแกนได้จาก QR ตอนประกอบเสร็จ (บรรจุ Machine No + IT Controller No.)
// heuristic: IT Controller No. = โทเคนตัวเลขล้วน 10–15 หลัก, ที่เหลือ = Machine No
function parseAssemblyCode(raw) {
  const s = (raw || '').trim()
  if (!s) return { machineNo: '', itControllerNo: '' }
  const tokens = s.split(/[\s,;|]+/).map((t) => t.trim()).filter(Boolean)
  const itc = tokens.find((t) => /^\d{10,15}$/.test(t)) || ''
  const mc = tokens.find((t) => t !== itc) || ''
  if (itc || mc) return { machineNo: mc, itControllerNo: itc }
  return { machineNo: tokens[0] || '', itControllerNo: tokens[1] || '' }
}

export default function MFGAssemblyPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  // ── โมดัลแก้ไข/เพิ่ม ───────────────────────────────────────────────────
  const [modalOpen, setModalOpen] = useState(false)
  const [editId, setEditId] = useState(null) // null = เพิ่มใหม่
  const [form, setForm] = useState(EMPTY_FORM)
  const [saving, setSaving] = useState(false)

  // ── สแกนเนอร์ ──────────────────────────────────────────────────────────
  // scanTarget: 'combined' = สแกน QR ประกอบเสร็จ (ได้ทั้งสองค่า)
  //             'machineNo' / 'itControllerNo' = สแกนเติมทีละช่องในโมดัล
  const [showScanner, setShowScanner] = useState(false)
  const [scanTarget, setScanTarget] = useState('combined')
  const [scanBusy, setScanBusy] = useState(false)

  // 404/405 = backend ยังไม่มี endpoint (มักเพราะยังไม่ได้ rebuild/restart)
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
  }, [search, pageSize])

  // ── สแกน ───────────────────────────────────────────────────────────────
  function openCombinedScan() {
    setScanTarget('combined')
    setShowScanner(true)
  }

  function openFieldScan(field) {
    setScanTarget(field)
    setShowScanner(true)
  }

  async function submitScan(machineNo, itControllerNo) {
    setScanBusy(true)
    try {
      const res = await scanMFGAssembly({ machineNo, itControllerNo })
      if (res?.matched) {
        toastSuccess(res?.message || 'บันทึกสำเร็จ')
      } else {
        // ยังบันทึกแล้ว แต่ต้อง flag ให้เห็น
        toastError(res?.message || 'บันทึกแล้ว — มีข้อควรตรวจสอบ')
      }
      await loadRows()
    } catch (err) {
      toastError(friendlyError(err, 'บันทึกไม่สำเร็จ'))
    } finally {
      setScanBusy(false)
    }
  }

  async function handleScanDetected(text) {
    const target = scanTarget
    setShowScanner(false)

    if (target === 'machineNo' || target === 'itControllerNo') {
      const parsed = parseAssemblyCode(text)
      // ถ้าช่องนั้นเป็น IT Controller ใช้ค่าเลขล้วน ถ้าเป็น Machine ใช้ตัวที่เหลือ
      const val =
        target === 'itControllerNo'
          ? parsed.itControllerNo || text.trim()
          : parsed.machineNo || text.trim()
      setForm((f) => ({ ...f, [target]: val }))
      return
    }

    // combined
    const { machineNo, itControllerNo } = parseAssemblyCode(text)
    if (machineNo && itControllerNo) {
      await submitScan(machineNo, itControllerNo)
    } else {
      // แยกไม่ครบ — เปิดโมดัลให้เติม/แก้เอง
      setEditId(null)
      setForm({ ...EMPTY_FORM, machineNo, itControllerNo })
      setModalOpen(true)
      toastError('อ่านได้ไม่ครบทั้งสองค่า — กรุณาตรวจ/เติมข้อมูลก่อนบันทึก')
    }
  }

  // ── โมดัล เพิ่ม/แก้ไข ────────────────────────────────────────────────────
  function openAdd() {
    setEditId(null)
    setForm(EMPTY_FORM)
    setModalOpen(true)
  }

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
    if (!form.machineNo.trim() || !form.itControllerNo.trim()) {
      toastError('กรุณากรอก Machine No และ IT Controller No.')
      return
    }
    setSaving(true)
    try {
      if (editId) {
        await updateMFGAssembly(editId, form)
        toastSuccess('แก้ไขรายการแล้ว')
      } else {
        await createMFGAssembly(form)
        toastSuccess('เพิ่มรายการแล้ว')
      }
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
    if (!term) return rows
    return rows.filter(
      (r) =>
        (r.Item || '').toLowerCase().includes(term) ||
        (r.MachineNo || '').toLowerCase().includes(term) ||
        (r.ITControllerNo || '').toLowerCase().includes(term) ||
        (r.Country || '').toLowerCase().includes(term) ||
        (r.Status || '').toLowerCase().includes(term)
    )
  }, [rows, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  const scannerTitle =
    scanTarget === 'machineNo'
      ? 'สแกน Machine No'
      : scanTarget === 'itControllerNo'
        ? 'สแกน IT Controller No.'
        : 'สแกน QR เครื่องที่ประกอบเสร็จ'

  return (
    <AppShell navItems={MFG_NAV_ITEMS} roleLabel="MFG">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Matching Assembly</h2>
          <p className="wh-subtitle">
            สแกน QR ตอนประกอบเสร็จ — ระบบบันทึก Machine No + IT Controller No. แล้วตรวจสถานะให้
          </p>
        </div>
        <button className="wh-issue-btn" onClick={openCombinedScan} disabled={scanBusy}>
          <QrCodeIcon className="size-4" />
          {scanBusy ? 'กำลังบันทึก...' : 'สแกน QR ประกอบเสร็จ'}
        </button>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

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
        <div className="mfg-search-actions">
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหา Item / Machine No / IT Controller / Country / Status"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <button className="tsf-action-btn" onClick={openAdd}>
            + เพิ่มรายการ
          </button>
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
              paged.map((a) => {
                const meta = STATUS_META[a.Status] || {
                  label: a.Status || '—',
                  cls: 'il-badge il-badge-muted',
                }
                return (
                  <tr key={a.ID}>
                    <td className="wh-cell-head" data-label="Item">
                      <strong>{a.Item || '—'}</strong>
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
                      <span className={meta.cls}>{meta.label}</span>
                    </td>
                    <td className="wh-cell-action">
                      <button className="tsf-action-btn" onClick={() => openEdit(a)}>
                        แก้ไข
                      </button>
                      <button
                        className="tsf-action-btn tsf-action-btn-danger"
                        onClick={() => handleDelete(a)}
                      >
                        ลบ
                      </button>
                    </td>
                  </tr>
                )
              })}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  {rows.length === 0
                    ? 'ยังไม่มีรายการ — สแกน QR เครื่องที่ประกอบเสร็จแล้วข้อมูลจะขึ้นที่นี่'
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

      {/* ── แก้ไข / เพิ่มรายการ ─────────────────────────────────────────── */}
      {modalOpen && (
        <div className="wh-modal-overlay" onClick={closeModal}>
          <div className="wh-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">{editId ? 'แก้ไขรายการ' : 'เพิ่มรายการ'}</h3>

            <label className="wh-modal-label">Item</label>
            <input
              className="wh-modal-input"
              value={form.item}
              onChange={(e) => setField('item', e.target.value)}
              placeholder="ลำดับ/รหัสรายการ (เว้นว่างให้ระบบใส่ลำดับถัดไป)"
            />

            <label className="wh-modal-label">Date Ass'y</label>
            <input
              className="wh-modal-input"
              type="date"
              value={form.dateAssembly}
              onChange={(e) => setField('dateAssembly', e.target.value)}
            />

            <label className="wh-modal-label">Machine No</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input
                className="wh-modal-input"
                style={{ flex: 1 }}
                value={form.machineNo}
                onChange={(e) => setField('machineNo', e.target.value)}
                placeholder="เช่น LX10400690"
              />
              <button
                type="button"
                className="tsf-action-btn"
                onClick={() => openFieldScan('machineNo')}
              >
                <QrCodeIcon className="size-4" /> สแกน
              </button>
            </div>

            <label className="wh-modal-label">IT Controller No.</label>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <input
                className="wh-modal-input"
                style={{ flex: 1 }}
                value={form.itControllerNo}
                onChange={(e) => setField('itControllerNo', e.target.value)}
                placeholder="เช่น 878250022802"
              />
              <button
                type="button"
                className="tsf-action-btn"
                onClick={() => openFieldScan('itControllerNo')}
              >
                <QrCodeIcon className="size-4" /> สแกน
              </button>
            </div>

            <label className="wh-modal-label">Country</label>
            <input
              className="wh-modal-input"
              value={form.country}
              onChange={(e) => setField('country', e.target.value)}
              placeholder="เว้นว่างให้ระบบดึงจากบัญชีใบอนุญาตนำเข้า (ถ้ามี)"
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

      {/* ── สแกนเนอร์ ───────────────────────────────────────────────────── */}
      {showScanner && (
        <BarcodeScannerModal
          title={scannerTitle}
          onDetected={handleScanDetected}
          onClose={() => setShowScanner(false)}
        />
      )}
    </AppShell>
  )
}
