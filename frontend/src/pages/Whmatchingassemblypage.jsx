import { useEffect, useMemo, useState } from 'react'
import {
  getMatchingAssemblies,
  createMatchingAssembly,
  updateMatchingAssembly,
  deleteMatchingAssembly,
} from '../api/matchingAssembly.js'
import { confirmDelete, toastSuccess, toastError } from '../lib/toast.js'
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from '../components/icons.jsx'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { WH_NAV_ITEMS } from './Importlicensepage.jsx'

const EMPTY_FORM = {
  item: '',
  dateAssy: '',
  machineNo: '',
  itControllerSN: '',
  country: '',
  classification: '',
  assemblyPartsNo: '',
  assemblyPartsName: '',
}

// ISO datetime -> "YYYY-MM-DD" สำหรับ <input type="date">
function toDateInput(v) {
  if (!v) return ''
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return ''
  const off = d.getTimezoneOffset()
  const local = new Date(d.getTime() - off * 60000)
  return local.toISOString().slice(0, 10)
}

export default function WHMatchingAssemblyPage() {
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

  // แปลง error ให้ผู้ใช้เข้าใจง่าย — 404/405 = backend ยังไม่มี endpoint นี้
  // (มักเกิดเมื่อยังไม่ได้ rebuild/restart backend หลังเพิ่มฟีเจอร์ Matching Assembly)
  function friendlyError(err, fallback) {
    if (err?.status === 404 || err?.status === 405) {
      return 'ยังไม่พบ API /matching-assembly ที่ฝั่งเซิร์ฟเวอร์ — ต้อง rebuild แล้ว restart backend ก่อน'
    }
    return err?.message || fallback
  }

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const list = await getMatchingAssemblies()
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

  function openAdd() {
    setEditId(null)
    setForm(EMPTY_FORM)
    setModalOpen(true)
  }

  function openEdit(row) {
    setEditId(row.ID)
    setForm({
      item: row.Item || '',
      dateAssy: toDateInput(row.DateAssy),
      machineNo: row.MachineNo || '',
      itControllerSN: row.ITControllerSN || '',
      country: row.Country || '',
      classification: row.Classification || '',
      assemblyPartsNo: row.AssemblyPartsNo || '',
      assemblyPartsName: row.AssemblyPartsName || '',
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
    setSaving(true)
    try {
      if (editId) {
        await updateMatchingAssembly(editId, form)
        toastSuccess('แก้ไข Matching Assembly แล้ว')
      } else {
        await createMatchingAssembly(form)
        toastSuccess('เพิ่ม Matching Assembly แล้ว')
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
    const label = row.MachineNo || row.AssemblyPartsNo || '#' + row.ID
    const ok = await confirmDelete({ text: `ลบรายการ Matching Assembly ${label}? กู้คืนไม่ได้` })
    if (!ok) return
    try {
      await deleteMatchingAssembly(row.ID)
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
        (r.ITControllerSN || '').toLowerCase().includes(term) ||
        (r.Country || '').toLowerCase().includes(term) ||
        (r.Classification || '').toLowerCase().includes(term) ||
        (r.AssemblyPartsNo || '').toLowerCase().includes(term) ||
        (r.AssemblyPartsName || '').toLowerCase().includes(term)
    )
  }, [rows, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  return (
    <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Matching Assembly</h2>
          <p className="wh-subtitle">
            เมื่อสแกน IT Controller สำเร็จบนหน้า Part Confirmation ระบบใช้ P/N (เช่น YN22E00849FA)
            เป็นตัวเชื่อม ดึงข้อมูลลงตารางนี้อัตโนมัติ — แก้ไข/ลบ/เพิ่มเองได้
          </p>
        </div>
        <button className="wh-issue-btn" onClick={openAdd}>
          + เพิ่มแถว
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
        <input
          className="wh-search"
          type="text"
          placeholder="ค้นหา Item / Machine No. / S/N / Country / Parts"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Item</th>
              <th>Date Assy</th>
              <th>Machine No.</th>
              <th>IT Controller Serial No.</th>
              <th>Country</th>
              <th>Classification</th>
              <th>Assembly Parts Number</th>
              <th>Assembly Parts Name</th>
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
              paged.map((a) => (
                <tr key={a.ID}>
                  <td className="wh-cell-head" data-label="Item">
                    <strong>{a.Item || '—'}</strong>
                  </td>
                  <td data-label="Date Assy">
                    {a.DateAssy ? new Date(a.DateAssy).toLocaleDateString('th-TH') : '—'}
                  </td>
                  <td className="il-mono" data-label="Machine No.">
                    {a.MachineNo || '—'}
                  </td>
                  <td className="il-mono" data-label="IT Controller Serial No.">
                    {a.ITControllerSN || '—'}
                  </td>
                  <td data-label="Country">{a.Country || '—'}</td>
                  <td data-label="Classification">{a.Classification || '—'}</td>
                  <td className="il-mono" data-label="Assembly Parts Number">
                    {a.AssemblyPartsNo || '—'}
                  </td>
                  <td data-label="Assembly Parts Name">{a.AssemblyPartsName || '—'}</td>
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
              ))}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={9} className="wh-empty-cell">
                  {rows.length === 0
                    ? 'ยังไม่มีรายการ — สแกน IT Controller สำเร็จแล้วข้อมูลจะขึ้นที่นี่'
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

      {/* ── แก้ไข / เพิ่ม Matching Assembly ─────────────────────────────── */}
      {modalOpen && (
        <div className="wh-modal-overlay" onClick={closeModal}>
          <div className="wh-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">
              {editId ? 'แก้ไข Matching Assembly' : 'เพิ่ม Matching Assembly'}
            </h3>

            <label className="wh-modal-label">Item</label>
            <input
              className="wh-modal-input"
              value={form.item}
              onChange={(e) => setField('item', e.target.value)}
              placeholder="ลำดับ/รหัสรายการ"
            />

            <label className="wh-modal-label">Date Assy</label>
            <input
              className="wh-modal-input"
              type="date"
              value={form.dateAssy}
              onChange={(e) => setField('dateAssy', e.target.value)}
            />

            <label className="wh-modal-label">Machine No.</label>
            <input
              className="wh-modal-input"
              value={form.machineNo}
              onChange={(e) => setField('machineNo', e.target.value)}
              placeholder="หมายเลขเครื่อง (IT Controller No.)"
            />

            <label className="wh-modal-label">IT Controller Serial No.</label>
            <input
              className="wh-modal-input"
              value={form.itControllerSN}
              onChange={(e) => setField('itControllerSN', e.target.value)}
            />

            <label className="wh-modal-label">Country</label>
            <input
              className="wh-modal-input"
              value={form.country}
              onChange={(e) => setField('country', e.target.value)}
            />

            <label className="wh-modal-label">Classification</label>
            <input
              className="wh-modal-input"
              value={form.classification}
              onChange={(e) => setField('classification', e.target.value)}
            />

            <label className="wh-modal-label">Assembly Parts Number</label>
            <input
              className="wh-modal-input"
              value={form.assemblyPartsNo}
              onChange={(e) => setField('assemblyPartsNo', e.target.value)}
              placeholder="เช่น YN22E00849FA"
            />

            <label className="wh-modal-label">Assembly Parts Name</label>
            <input
              className="wh-modal-input"
              value={form.assemblyPartsName}
              onChange={(e) => setField('assemblyPartsName', e.target.value)}
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
