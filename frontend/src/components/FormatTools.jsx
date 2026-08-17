import { useEffect, useState } from 'react'
import SelectField from './Selectfield.jsx'
import './FormatTools.css'
import { confirmDelete, toastError, toastSuccess } from '../lib/toast.js'
import {
  getColumnAliases,
  createColumnAlias,
  deleteColumnAlias,
  getCodeAliases,
  createCodeAlias,
  deleteCodeAlias,
  uploadCodeAliases,
} from '../api/formatConfig.js'
import { updateMasterData } from '../api/masterData.js'
import { buildStyledXlsxBlob, downloadBlob } from '../lib/xlsx.js'

// ─────────────────────────────────────────────────────────────────────────────
// เครื่องมือ "รองรับการเปลี่ยน format" ที่ใช้ร่วมกันหลายหน้า
//   • ColumnAliasPanel — จับคู่หัวคอลัมน์ในไฟล์ที่เปลี่ยน → คอลัมน์มาตรฐาน (ตาม scope)
//   • CodeAliasPanel   — จับคู่ค่า P/N / S/N / Machine No. รูปแบบใหม่ → เลขมาตรฐาน
//   • MasterDataEditModal — แก้ไขทะเบียนกลางรายรายการ (PATCH)
//   • PreviewResult    — แสดงผล dry-run (matched / missing / extra) ก่อนอัปโหลดจริง
// ─────────────────────────────────────────────────────────────────────────────

const panelStyle = {
  border: '1px solid #e2e8f0',
  borderRadius: 12,
  background: '#fff',
  marginTop: 16,
  overflow: 'hidden',
}
const panelHeadStyle = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 10,
  padding: '12px 16px',
  cursor: 'pointer',
  background: '#f8fafc',
  fontWeight: 600,
  fontSize: 14,
}
const panelBodyStyle = { padding: '14px 16px' }
const codeStyle = { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }

function Collapsible({ title, hint, children, defaultOpen = false }) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div style={panelStyle}>
      <div style={panelHeadStyle} onClick={() => setOpen((v) => !v)}>
        <span>
          {title}
          {hint && <span style={{ fontWeight: 400, color: '#94a3b8', marginLeft: 8 }}>{hint}</span>}
        </span>
        <span style={{ color: '#64748b' }}>{open ? '▲' : '▼'}</span>
      </div>
      {open && <div style={panelBodyStyle}>{children}</div>}
    </div>
  )
}

// ── A) Column Alias ────────────────────────────────────────────────────────
// scope = ชื่อ dataset (planning | wh1 | wh2 | engine) หรือ import_license | export_license
//
// targetOptions = รายชื่อ "คอลัมน์มาตรฐาน" ที่ถูกต้องของ scope นี้ (ให้เลือกจาก dropdown
// แทนการพิมพ์เอง) — แก้ปัญหา "ตั้งแล้วไม่เปลี่ยน" ที่เกิดจากพิมพ์ชื่อคอลัมน์ไม่ตรงเป๊ะ
export function ColumnAliasPanel({ scope, targetOptions = [], embedded = false }) {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(false)
  const [source, setSource] = useState('')
  const [target, setTarget] = useState('')
  const [note, setNote] = useState('')
  const [changeKind, setChangeKind] = useState('rename') // rename | add | reorder
  const [saving, setSaving] = useState(false)
  const hasTargets = Array.isArray(targetOptions) && targetOptions.length > 0

  async function load() {
    setLoading(true)
    try {
      const data = await getColumnAliases(scope)
      setRows(Array.isArray(data) ? data : [])
    } catch (err) {
      toastError(err.message || 'โหลดรายการจับคู่คอลัมน์ไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    setSource('')
    setTarget('')
    setNote('')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scope])

  async function handleAdd() {
    if (!source.trim() || !target.trim()) {
      toastError('กรอก "ข้อมูลใหม่ที่จะเปลี่ยน" และเลือก "ข้อมูลเดิม"')
      return
    }
    setSaving(true)
    try {
      await createColumnAlias({
        scope,
        source: source.trim(),
        target: target.trim(),
        note: note.trim(),
        kind: changeKind,
      })
      setSource('')
      setTarget('')
      setNote('')
      toastSuccess('เพิ่มการจับคู่คอลัมน์แล้ว')
      await load()
    } catch (err) {
      toastError(err.message || 'เพิ่มไม่สำเร็จ')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id) {
    const ok = await confirmDelete({ text: 'ลบการจับคู่คอลัมน์นี้?' })
    if (!ok) return
    try {
      await deleteColumnAlias(id)
      toastSuccess('ลบแล้ว')
      await load()
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  const CHANGE_KIND_LABEL = { rename: 'เปลี่ยนชื่อ', add: 'เพิ่มใหม่', reorder: 'สลับตำแหน่ง' }
  // ป้ายช่อง "ข้อมูลใหม่ที่จะเปลี่ยน" ปรับตามชนิดการเปลี่ยนให้เข้าใจง่าย
  const sourceLabel = changeKind === 'add' ? 'ชื่อหัวคอลัมน์ที่เพิ่มเข้ามา' : 'ชื่อหัวคอลัมน์ใหม่ (ที่เปลี่ยนมา)'

  const body = (
    <>
      <div className="fmt-field" style={{ maxWidth: 260, marginBottom: 12 }}>
        <label className="fmt-label">ชนิดการเปลี่ยน</label>
        <SelectField
          value={changeKind}
          onChange={setChangeKind}
          options={[
            { value: 'rename', label: 'เปลี่ยนชื่อหัวคอลัมน์' },
            { value: 'add', label: 'เพิ่มหัวคอลัมน์ใหม่' },
            { value: 'reorder', label: 'สลับตำแหน่งคอลัมน์' },
          ]}
        />
      </div>

      {changeKind === 'reorder' ? (
        <div
          style={{
            background: '#f0fdfa',
            border: '1px solid #99f6e4',
            borderRadius: 10,
            padding: '12px 14px',
            fontSize: 13,
            color: '#0f766e',
            lineHeight: 1.6,
          }}
        >
          การ <b>สลับตำแหน่งคอลัมน์</b> ระบบรองรับให้อัตโนมัติอยู่แล้ว — เพราะจับคู่ด้วย
          <b> ชื่อหัวคอลัมน์</b> ไม่ใช่ตำแหน่ง ดังนั้นย้ายคอลัมน์ไปไว้ตรงไหนก็อ่านถูก
          <b> ไม่ต้องตั้งค่าเพิ่ม</b>
        </div>
      ) : (
        <>
          <div className="fmt-form">
            <div className="fmt-field">
              <label className="fmt-label">{sourceLabel}</label>
              <input
                className="fmt-input"
                value={source}
                onChange={(e) => setSource(e.target.value)}
                placeholder="เช่น หมายเลขเครื่อง (ใหม่)"
              />
            </div>
            <div className="fmt-field">
              <label className="fmt-label">ข้อมูลเดิม</label>
              {hasTargets ? (
                <SelectField
                  value={target}
                  onChange={setTarget}
                  options={targetOptions.map((t) =>
                    typeof t === 'string' ? { value: t, label: t } : { value: t.value, label: t.label },
                  )}
                  placeholder="— เลือกคอลัมน์ —"
                />
              ) : (
                <input
                  className="fmt-input"
                  value={target}
                  onChange={(e) => setTarget(e.target.value)}
                  placeholder="เช่น Machine"
                />
              )}
            </div>
            <div className="fmt-field">
              <label className="fmt-label">หมายเหตุ (ไม่บังคับ)</label>
              <input className="fmt-input" value={note} onChange={(e) => setNote(e.target.value)} />
            </div>
          </div>

          <div className="fmt-actions">
            <button className="wh-issue-btn fmt-add-btn" onClick={handleAdd} disabled={saving}>
              {saving ? 'กำลังเพิ่ม...' : 'เพิ่ม'}
            </button>
          </div>
        </>
      )}

      <div className="fmt-table-wrap">
        <table className="wh-table" style={{ width: '100%' }}>
          <thead>
            <tr>
              <th>ชนิด</th>
              <th>ข้อมูลใหม่ที่จะเปลี่ยน</th>
              <th>→ ข้อมูลเดิม</th>
              <th>หมายเหตุ</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">กำลังโหลด...</td>
              </tr>
            )}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">ยังไม่มีการจับคู่ — ไฟล์ปกติไม่ต้องตั้งค่าอะไร</td>
              </tr>
            )}
            {!loading &&
              rows.map((r) => (
                <tr key={r.id}>
                  <td>{CHANGE_KIND_LABEL[r.kind] || 'เปลี่ยนชื่อ'}</td>
                  <td style={codeStyle}>{r.source}</td>
                  <td style={codeStyle}>{r.target}</td>
                  <td>{r.note || '—'}</td>
                  <td className="wh-cell-action">
                    <button className="qa-fail-btn" onClick={() => handleDelete(r.id)}>ลบ</button>
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>
    </>
  )

  // embedded = ฝังตรง ๆ ในการ์ด (ไม่ต้องมีหัวพับ) — ใช้ในหน้า Setting ที่จัด layout เอง
  if (embedded) return body

  return (
    <Collapsible title="จับคู่หัวคอลัมน์ (เมื่อไฟล์เปลี่ยนชื่อหัวคอลัมน์)" defaultOpen>
      {body}
    </Collapsible>
  )
}

// ── B) Code Alias ──────────────────────────────────────────────────────────
// ป้ายกำกับชนิดรหัส — ใช้ต่อท้ายชื่อช่อง เช่น "ข้อมูลใหม่ที่จะเปลี่ยน(Machine No.)"
const KIND_LABEL = { machine: 'Machine No.', pn: 'P/N', sn: 'S/N' }

export function CodeAliasPanel({ componentType = 'it_controller', embedded = false }) {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(false)
  const [fromCode, setFromCode] = useState('')
  const [toSerial, setToSerial] = useState('')
  const [kind, setKind] = useState('machine')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const kindText = KIND_LABEL[kind] || ''

  async function load() {
    setLoading(true)
    try {
      const data = await getCodeAliases({ componentType })
      setRows(Array.isArray(data) ? data : [])
    } catch (err) {
      toastError(err.message || 'โหลดรายการจับคู่ค่ารหัสไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [componentType])

  async function handleAdd() {
    if (!fromCode.trim() || !toSerial.trim()) {
      toastError('กรอกทั้ง "ข้อมูลใหม่" และ "ข้อมูลเก่า"')
      return
    }
    setSaving(true)
    try {
      await createCodeAlias({
        from_code: fromCode.trim(),
        to_serial_no: toSerial.trim(),
        to_part_no: '',
        component_type: componentType,
        kind,
        note: note.trim(),
      })
      setFromCode('')
      setToSerial('')
      setNote('')
      toastSuccess('เพิ่มการจับคู่ค่ารหัสแล้ว')
      await load()
    } catch (err) {
      toastError(err.message || 'เพิ่มไม่สำเร็จ')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id) {
    const ok = await confirmDelete({ text: 'ลบการจับคู่ค่ารหัสนี้?' })
    if (!ok) return
    try {
      await deleteCodeAlias(id)
      toastSuccess('ลบแล้ว')
      await load()
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  async function handleUpload(e) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setUploading(true)
    try {
      const res = await uploadCodeAliases(file)
      toastSuccess(`นำเข้าแล้ว ${res.imported ?? 0} รายการ`)
      await load()
    } catch (err) {
      toastError(err.message || 'อัปโหลดไม่สำเร็จ')
    } finally {
      setUploading(false)
    }
  }

  // ดาวน์โหลดไฟล์ตัวอย่าง (.xlsx) — หัวคอลัมน์ + ตัวอย่างข้อมูลให้กรอกตาม
  function handleDownloadSample() {
    const columns = [
      { key: 'from_code', header: 'from_code', type: 'text' },
      { key: 'to_serial_no', header: 'to_serial_no', type: 'text' },
      { key: 'kind', header: 'kind', type: 'text' },
      { key: 'note', header: 'note', type: 'text' },
    ]
    const rows = [
      { from_code: 'TNN-YN23993', to_serial_no: 'YN23993', kind: 'machine', note: 'ตัวอย่าง Machine No.' },
      { from_code: 'KQ-3000/NEW', to_serial_no: 'KQ3000045093', kind: 'sn', note: 'ตัวอย่าง S/N' },
      { from_code: 'YN22-E00849', to_serial_no: 'YN22E00849FA', kind: 'pn', note: 'ตัวอย่าง P/N' },
    ]
    const blob = buildStyledXlsxBlob({ sheetName: 'Change Format Part', columns, rows })
    downloadBlob(blob, 'change-format-part-ตัวอย่าง.xlsx')
  }

  const body = (
    <>
      <div className="fmt-form">
        <div className="fmt-field">
          <label className="fmt-label">ชนิดรหัส</label>
          <SelectField
            value={kind}
            onChange={setKind}
            options={[
              { value: 'machine', label: 'Machine No.' },
              { value: 'sn', label: 'S/N' },
              { value: 'pn', label: 'P/N' },
            ]}
          />
        </div>
        <div className="fmt-field">
          <label className="fmt-label">ข้อมูลใหม่ที่จะเปลี่ยน({kindText})</label>
          <input
            className="fmt-input"
            value={fromCode}
            onChange={(e) => setFromCode(e.target.value)}
            placeholder="เช่น TNN-YN23993 / KQ-3000/NEW"
          />
        </div>
        <div className="fmt-field">
          <label className="fmt-label">ข้อมูลเดิม({kindText})</label>
          <input
            className="fmt-input"
            value={toSerial}
            onChange={(e) => setToSerial(e.target.value)}
            placeholder="เช่น KQ3000045093 / YN23993"
          />
        </div>
        <div className="fmt-field">
          <label className="fmt-label">หมายเหตุ (ไม่บังคับ)</label>
          <input className="fmt-input" value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
      </div>

      <div className="fmt-actions">
        <button className="wh-issue-btn fmt-action-btn" type="button" onClick={handleDownloadSample}>
          ดาวน์โหลดตัวอย่าง
        </button>
        <label className="wh-issue-btn fmt-action-btn" style={{ cursor: 'pointer' }}>
          <input type="file" accept=".xlsx,.xls,.csv" onChange={handleUpload} style={{ display: 'none' }} disabled={uploading} />
          {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลดหลายรายการจากไฟล์'}
        </label>
        <button className="wh-issue-btn fmt-add-btn" onClick={handleAdd} disabled={saving}>
          {saving ? 'กำลังเพิ่ม...' : 'เพิ่ม'}
        </button>
      </div>

      <div className="fmt-table-wrap">
        <table className="wh-table" style={{ width: '100%' }}>
          <thead>
            <tr>
              <th>ชนิด</th>
              <th>ข้อมูลใหม่</th>
              <th>→ ข้อมูลเก่า</th>
              <th>หมายเหตุ</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">กำลังโหลด...</td>
              </tr>
            )}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">ยังไม่มีการจับคู่ค่ารหัส</td>
              </tr>
            )}
            {!loading &&
              rows.map((r) => (
                <tr key={r.id}>
                  <td>{KIND_LABEL[r.kind] || '—'}</td>
                  <td style={codeStyle}>{r.from_code}</td>
                  <td style={codeStyle}>{r.to_serial_no}</td>
                  <td>{r.note || '—'}</td>
                  <td className="wh-cell-action">
                    <button className="qa-fail-btn" onClick={() => handleDelete(r.id)}>ลบ</button>
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>
    </>
  )

  if (embedded) return body

  return (
    <Collapsible title="Change Format Part" defaultOpen>
      {body}
    </Collapsible>
  )
}

// ── Preview result (dry-run) ────────────────────────────────────────────────
export function PreviewResult({ result }) {
  if (!result) return null
  if (result.headerFound === false) {
    return (
      <div style={{ marginTop: 10, padding: '10px 12px', background: '#fef2f2', borderRadius: 10, fontSize: 13, color: '#b42318' }}>
        {result.message || 'หาหัวตารางไม่เจอในไฟล์นี้'}
      </div>
    )
  }
  const matched = result.matched || []
  const missing = result.missing || []
  const extra = result.extra || []
  const chip = (text, bg, color) => (
    <span key={text} style={{ background: bg, color, borderRadius: 999, padding: '2px 10px', fontSize: 12, ...codeStyle }}>
      {text}
    </span>
  )
  return (
    <div style={{ marginTop: 10, padding: '12px 14px', background: '#f8fafc', borderRadius: 10, fontSize: 13 }}>
      <div style={{ marginBottom: 6 }}>
        พบหัวตารางแถวที่ {result.headerRow ?? '—'} — ไฟล์: <strong>{result.file}</strong>
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center', marginBottom: missing.length || extra.length ? 8 : 0 }}>
        <span style={{ color: '#16a34a', fontWeight: 600 }}>แม็ปได้ {matched.length}:</span>
        {matched.length ? matched.map((m) => chip(typeof m === 'string' ? m : m.label, '#dcfce7', '#166534')) : <span style={{ color: '#94a3b8' }}>—</span>}
      </div>
      {missing.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center', marginBottom: extra.length ? 8 : 0 }}>
          <span style={{ color: '#b45309', fontWeight: 600 }}>ไม่พบในไฟล์ {missing.length}:</span>
          {missing.map((m) => chip(m, '#fef3c7', '#92400e'))}
        </div>
      )}
      {extra.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
          <span style={{ color: '#2563eb', fontWeight: 600 }}>คอลัมน์ใหม่ (จะถูกเก็บไว้) {extra.length}:</span>
          {extra.map((m) => chip(m, '#dbeafe', '#1e40af'))}
        </div>
      )}
    </div>
  )
}

// ── แสดงคอลัมน์ใหม่ที่ระบบเก็บไว้ (extra_json) เป็น chip เล็ก ๆ ในตาราง ─────────
export function ExtraColumnsCell({ json }) {
  let obj = null
  try {
    obj = json ? JSON.parse(json) : null
  } catch {
    obj = null
  }
  // คีย์ที่ตอนนี้ระบบรู้จักเป็นคอลัมน์จริงแล้ว (เช่น Country) — ไม่ต้องโชว์เป็น "คอลัมน์เพิ่ม"
  // ซ้ำอีก (รองรับข้อมูลเก่าที่อัปโหลดไว้ก่อนจะรู้จักคอลัมน์นี้ ยังค้างใน extra_json)
  const HIDDEN_EXTRA = new Set([
    'country',
    'countryname',
    'exportcountry',
    'ประเทศ',
    'ปลายทาง',
    'ส่งออกไปประเทศ',
  ])
  const normKey = (k) =>
    String(k)
      .replace(/^\[\+\]\s*/, '')
      .toLowerCase()
      .replace(/[\s_./-]/g, '')
  const entries = obj
    ? Object.entries(obj).filter(([k]) => !HIDDEN_EXTRA.has(normKey(k)))
    : []
  if (entries.length === 0) return <span style={{ color: '#cbd5e1' }}>—</span>
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 5, minWidth: 160 }}>
      {entries.map(([k, v]) => {
        const label = k.replace(/^\[\+\]\s*/, '')
        return (
          <div
            key={k}
            title={`${label}: ${v}`}
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 1,
              padding: '5px 9px',
              background: '#f8fafc',
              border: '1px solid #e5e9f0',
              borderLeft: '3px solid #60a5fa',
              borderRadius: 7,
            }}
          >
            <span
              style={{
                fontSize: 10,
                fontWeight: 700,
                letterSpacing: 0.3,
                color: '#64748b',
                textTransform: 'uppercase',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {label}
            </span>
            <span style={{ fontSize: 12.5, color: '#0f172a', fontWeight: 500, ...codeStyle }}>
              {String(v) || '—'}
            </span>
          </div>
        )
      })}
    </div>
  )
}

// ── Change-detection preview (Master Data) ──────────────────────────────────
// แสดงสรุป NEW/UPDATED/CHANGED/UNCHANGED + ตารางค่า old→new ของแถวที่เปลี่ยน
export function ChangePreview({ result }) {
  if (!result) return null
  if (result.headerFound === false) {
    return (
      <div style={{ marginTop: 10, padding: '10px 12px', background: '#fef2f2', borderRadius: 10, fontSize: 13, color: '#b42318' }}>
        {result.message || 'หาหัวตารางไม่เจอในไฟล์นี้'}
      </div>
    )
  }
  const s = result.summary || {}
  const rows = result.rows || []
  const extra = result.extra || []

  const stat = (label, value, bg, color) => (
    <div style={{ background: bg, color, borderRadius: 10, padding: '8px 12px', minWidth: 92, textAlign: 'center' }}>
      <div style={{ fontSize: 20, fontWeight: 700, lineHeight: 1 }}>{value ?? 0}</div>
      <div style={{ fontSize: 12, marginTop: 2 }}>{label}</div>
    </div>
  )
  const badge = (status) => {
    const map = {
      NEW: ['#dcfce7', '#166534'],
      UPDATED: ['#dbeafe', '#1e40af'],
      CHANGED: ['#fef3c7', '#92400e'],
    }
    const [bg, color] = map[status] || ['#f1f5f9', '#475569']
    return <span style={{ background: bg, color, borderRadius: 999, padding: '2px 9px', fontSize: 12, fontWeight: 600 }}>{status}</span>
  }

  return (
    <div style={{ marginTop: 10, padding: '12px 14px', background: '#f8fafc', borderRadius: 10, fontSize: 13 }}>
      <div style={{ marginBottom: 8 }}>
        ตรวจไฟล์ <strong>{result.file}</strong> (หัวตารางแถวที่ {result.headerRow ?? '—'}) — ยังไม่บันทึก กดอัปโหลดเพื่อยืนยัน
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 10 }}>
        {stat('ทั้งหมด', s.total, '#eef2ff', '#3730a3')}
        {stat('เพิ่มใหม่', s.new, '#dcfce7', '#166534')}
        {stat('อัปเดต', s.updated, '#dbeafe', '#1e40af')}
        {stat('ค่าเปลี่ยน', s.changed, '#fef3c7', '#92400e')}
        {stat('เหมือนเดิม', s.unchanged, '#f1f5f9', '#475569')}
      </div>

      {extra.length > 0 && (
        <div style={{ marginBottom: 8, color: '#2563eb' }}>
          คอลัมน์ใหม่ที่ระบบไม่รู้จัก: {extra.join(', ')}
        </div>
      )}

      {s.changed > 0 && (
        <div style={{ marginBottom: 8, color: '#92400e' }}>
          ⚠ มี {s.changed} แถวที่ค่าหลัก (P/N · S/N · IT Controller · IMEI) เปลี่ยน — ตรวจก่อนยืนยัน
        </div>
      )}

      {rows.length > 0 ? (
        <div style={{ maxHeight: 320, overflow: 'auto', border: '1px solid #e2e8f0', borderRadius: 8 }}>
          <table className="wh-table" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th>สถานะ</th>
                <th>Serial No.</th>
                <th>ฟิลด์ที่เปลี่ยน (เดิม → ใหม่)</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => (
                <tr key={`${r.serial}-${i}`}>
                  <td>{badge(r.status)}</td>
                  <td style={codeStyle}>{r.serial}</td>
                  <td>
                    {(r.diffs || []).length === 0 ? (
                      <span style={{ color: '#94a3b8' }}>{r.status === 'NEW' ? 'แถวใหม่' : '—'}</span>
                    ) : (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                        {r.diffs.map((d, j) => (
                          <div key={j} style={{ fontSize: 12 }}>
                            <span style={{ color: '#64748b' }}>{d.field}: </span>
                            <span style={{ ...codeStyle, color: '#b91c1c' }}>{d.old || '(ว่าง)'}</span>
                            <span style={{ color: '#94a3b8' }}> → </span>
                            <span style={{ ...codeStyle, color: '#15803d' }}>{d.new || '(ว่าง)'}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div style={{ color: '#64748b' }}>ทุกแถวเหมือนเดิม — อัปโหลดจะไม่เปลี่ยนแปลงข้อมูล</div>
      )}

      {result.problems?.length > 0 && (
        <ul style={{ margin: '8px 0 0', paddingLeft: 18, color: '#b45309', fontSize: 12 }}>
          {result.problems.map((p, i) => (
            <li key={i}>{p}</li>
          ))}
        </ul>
      )}
    </div>
  )
}

// ── Master Data edit modal (PATCH) ──────────────────────────────────────────
export function MasterDataEditModal({ row, componentOptions = [], itcLabel = 'IT Controller no.', onClose, onSaved }) {
  const [form, setForm] = useState({
    Name: row.Name || '',
    Model: row.Model || '',
    ComponentType: row.ComponentType || '',
    PartNo: row.PartNo || '',
    SerialNo: row.SerialNo || '',
    ITControllerNo: row.ITControllerNo || '',
    IMEI: row.IMEI || '',
  })
  const [saving, setSaving] = useState(false)

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }))

  async function handleSave() {
    if (!form.SerialNo.trim() && !form.Name.trim()) {
      toastError('อย่างน้อยต้องมี Serial No. หรือ Part Name')
      return
    }
    setSaving(true)
    try {
      // ไม่ส่ง ItemNo / SpecCode — backend เป็น PATCH ค่าที่ไม่ส่งจะคงเดิมไว้
      const patch = {
        Name: form.Name.trim(),
        Model: form.Model.trim(),
        ComponentType: form.ComponentType.trim(),
        PartNo: form.PartNo.trim(),
        SerialNo: form.SerialNo.trim(),
        ITControllerNo: form.ITControllerNo.trim(),
        IMEI: form.IMEI.trim(),
      }
      await updateMasterData(row.ID, patch)
      toastSuccess('บันทึกการแก้ไขแล้ว')
      onSaved && onSaved()
      onClose && onClose()
    } catch (err) {
      toastError(err.message || 'บันทึกไม่สำเร็จ')
    } finally {
      setSaving(false)
    }
  }

  const field = (label, key, mono = false) => (
    <div className="fmt-field">
      <label className="fmt-label">{label}</label>
      <input className={'fmt-input' + (mono ? ' fmt-input-mono' : '')} value={form[key]} onChange={set(key)} />
    </div>
  )

  return (
    <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal" style={{ maxWidth: 560 }} onClick={(e) => e.stopPropagation()}>
        <h3 className="wh-modal-title">แก้ไขทะเบียน Master Data</h3>

        <div className="fmt-form fmt-form-compact" style={{ marginTop: 12 }}>
          {field('Part Name', 'Name')}
          {field('Model', 'Model')}
          <div className="fmt-field">
            <label className="fmt-label">ชนิดอะไหล่</label>
            {componentOptions.length > 0 ? (
              <SelectField
                value={form.ComponentType}
                onChange={(v) => setForm((f) => ({ ...f, ComponentType: v }))}
                options={componentOptions}
              />
            ) : (
              <input className="fmt-input" value={form.ComponentType} onChange={set('ComponentType')} />
            )}
          </div>
          {field('Part No.', 'PartNo', true)}
          {field('Serial No.', 'SerialNo', true)}
          {field(itcLabel, 'ITControllerNo', true)}
          {field('IMEI', 'IMEI', true)}
        </div>

        <p style={{ fontSize: 12, color: '#94a3b8', marginTop: 10 }}>
          แก้เพื่อให้ตรงกับ format ใหม่ที่หน้างานใช้ — เว้นว่างได้ในช่องที่อะไหล่ชนิดนั้นไม่มี
        </p>

        <div className="wh-modal-actions">
          <button className="wh-modal-cancel" onClick={onClose} disabled={saving}>ยกเลิก</button>
          <button className="wh-issue-btn" onClick={handleSave} disabled={saving}>
            {saving ? 'กำลังบันทึก...' : 'บันทึก'}
          </button>
        </div>
      </div>
    </div>
  )
}
