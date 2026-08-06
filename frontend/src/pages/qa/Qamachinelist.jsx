import { useEffect, useMemo, useState } from 'react'
import AppShell from '../../components/AppShell.jsx'
import SelectField from '../../components/Selectfield.jsx'
import {
  ArrowDownTrayIcon,
  CameraIcon,
  CheckCircleIcon,
  Squares2X2Icon,
  XMarkIcon,
} from '../../components/icons.jsx'
import { getQAConfirmedTable } from '../../api/qaConfirmed.js'
import { API_BASE_URL } from '../../api/client.js'

const navItems = [{ to: '/qa', label: 'ตรวจสอบ QA', icon: <CheckCircleIcon className="size-4" /> }]

// ป้ายผลเทียบใบอนุญาต — ตารางสรุปจะมีแต่ MATCH แต่แม็พเผื่อค่าอื่นไว้ด้วย
function licenseMatchMeta(status) {
  switch (status) {
    case 'MATCH':
      return { label: 'ตรงกับใบอนุญาต', cls: 'il-badge il-badge-ok' }
    case 'WRONG_INVOICE':
      return { label: 'คนละอินวอยซ์', cls: 'il-badge il-badge-warn' }
    case 'WRONG_PRODNO':
      return { label: 'IMEI ไม่ตรง', cls: 'il-badge il-badge-warn' }
    case 'DUPLICATE':
      return { label: 'ซ้ำ', cls: 'il-badge il-badge-warn' }
    case 'NOT_FOUND':
      return { label: 'ไม่พบในบัญชี', cls: 'il-badge il-badge-bad' }
    default:
      return { label: status || '—', cls: 'il-badge il-badge-muted' }
  }
}

const dash = (v) => (v && String(v).trim() !== '' ? v : '—')

// escape ค่าที่จะฝังลง HTML ของ Check Sheet กันอักขระพิเศษทำ markup พัง
function escapeHtml(s) {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

// สร้าง HTML ของ "Check Sheet" (แนวนอน A4) จากรายการที่ยืนยันแล้ว
function buildCheckSheetHTML(list) {
  const now = new Date()
  const dateStr = now.toLocaleDateString('th-TH', { year: 'numeric', month: 'long', day: 'numeric' })
  const timeStr = now.toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })

  const rowsHtml = list
    .map((r, i) => {
      const lic = licenseMatchMeta(r.matchStatus)
      const photo = r.photoURL
        ? `<img class="ph" src="${API_BASE_URL}${escapeHtml(r.photoURL)}" alt="" />`
        : '—'
      return `<tr>
        <td class="c">${i + 1}</td>
        <td>${escapeHtml(dash(r.partName))}</td>
        <td>${escapeHtml(dash(r.model))}</td>
        <td>${escapeHtml(dash(r.machineNo))}</td>
        <td>${escapeHtml(dash(r.partNo))}</td>
        <td>${escapeHtml(dash(r.serialNo))}</td>
        <td>${escapeHtml(dash(r.itControllerNo))}</td>
        <td>${escapeHtml(dash(r.imei))}</td>
        <td>${escapeHtml(dash(r.licenseNo))}</td>
        <td>${escapeHtml(dash(r.invoiceNo))}</td>
        <td>${escapeHtml(lic.label)}</td>
        <td class="c">${photo}</td>
        <td class="c">Matched</td>
      </tr>`
    })
    .join('')

  return `<!DOCTYPE html><html lang="th"><head><meta charset="utf-8" />
<title>QA Check Sheet</title>
<style>
  @page { size: A4 landscape; margin: 12mm; }
  * { box-sizing: border-box; }
  body { font-family: 'Sarabun','TH Sarabun New','Segoe UI',Tahoma,sans-serif; color:#0f172a; margin:0; }
  .head { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:10px; }
  h1 { font-size:18px; margin:0 0 4px; }
  .sub { color:#475569; font-size:11px; }
  .meta { text-align:right; font-size:11px; color:#475569; white-space:nowrap; }
  table { width:100%; border-collapse:collapse; }
  th,td { border:1px solid #94a3b8; padding:4px 6px; text-align:left; vertical-align:middle; word-break:break-word; }
  th { background:#f1f5f9; font-size:10px; }
  td { font-size:10px; }
  td.c, th.c { text-align:center; }
  .ph { width:40px; height:40px; object-fit:cover; border-radius:4px; }
  .sign { display:flex; justify-content:space-between; margin-top:30px; gap:40px; }
  .sign > div { flex:1; text-align:center; font-size:11px; }
  .sign .line { margin-top:40px; border-top:1px solid #0f172a; padding-top:4px; }
</style></head>
<body>
  <div class="head">
    <div>
      <h1>QA Check Sheet — ใบตรวจสอบ QA</h1>
      <div class="sub">รายการเครื่องที่ยืนยันแล้ว (WH Part Confirmation ตรงกับใบอนุญาต + MFG Matched)</div>
    </div>
    <div class="meta">วันที่พิมพ์: ${escapeHtml(dateStr)} ${escapeHtml(timeStr)}<br/>จำนวน: ${list.length} เครื่อง</div>
  </div>
  <table>
    <thead><tr>
      <th class="c">ITEM</th><th>Part Name</th><th>Model</th><th>Machine No</th>
      <th>Part No.</th><th>Serial No.</th><th>IT Controller No.</th><th>IMEI</th>
      <th>ใบอนุญาตนำเข้า</th><th>อินวอยซ์</th><th>ผลเทียบใบอนุญาต</th><th class="c">รูปถ่าย</th><th class="c">Status</th>
    </tr></thead>
    <tbody>${rowsHtml}</tbody>
  </table>
  <div class="sign">
    <div><div class="line">ผู้ตรวจสอบ (QA)</div></div>
    <div><div class="line">ผู้อนุมัติ</div></div>
    <div><div class="line">วันที่</div></div>
  </div>
</body></html>`
}

export default function QAMachineList() {
  const [confirmedRows, setConfirmedRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [photoView, setPhotoView] = useState(null) // URL รูปที่กำลังเปิดดู

  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  async function loadConfirmed() {
    setLoading(true)
    setLoadError('')
    try {
      const data = await getQAConfirmedTable()
      setConfirmedRows(data || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดตารางสรุปไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfirmed()
  }, [])

  // รีเฟรชทุกครั้งที่กลับมาโฟกัสหน้านี้ (เผื่อ WH/MFG เพิ่งยืนยันเพิ่ม)
  useEffect(() => {
    function handleFocus() {
      loadConfirmed()
    }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [])

  useEffect(() => {
    setPage(1)
  }, [search, pageSize])

  const stats = useMemo(() => {
    const total = confirmedRows.length
    const withPhoto = confirmedRows.filter((r) => r.photoURL).length
    return { total, withPhoto }
  }, [confirmedRows])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return confirmedRows
    return confirmedRows.filter((r) =>
      [
        r.partName,
        r.model,
        r.machineNo,
        r.partNo,
        r.serialNo,
        r.itControllerNo,
        r.imei,
        r.licenseNo,
        r.invoiceNo,
      ].some((v) => (v || '').toLowerCase().includes(q))
    )
  }, [confirmedRows, search])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const pageRows = filtered.slice((page - 1) * pageSize, page * pageSize)

  // Export เป็น PDF (Check Sheet) — สร้าง iframe ซ่อน เขียน HTML แล้วสั่งพิมพ์
  // (เลี่ยง popup blocker) ผู้ใช้เลือก "Save as PDF" จากหน้าต่างพิมพ์ของเบราว์เซอร์
  // แนะนำให้เลือกกระดาษ "แนวนอน (Landscape)" เพราะตารางกว้าง
  function handleExportPDF() {
    const list = filtered // ส่งออกตามที่กรอง/ค้นหาอยู่ (ทุกหน้า)
    if (!list.length) return

    const iframe = document.createElement('iframe')
    iframe.setAttribute('aria-hidden', 'true')
    Object.assign(iframe.style, {
      position: 'fixed',
      right: '0',
      bottom: '0',
      width: '0',
      height: '0',
      border: '0',
    })
    document.body.appendChild(iframe)

    const doc = iframe.contentWindow.document
    doc.open()
    doc.write(buildCheckSheetHTML(list))
    doc.close()

    let printed = false
    const cleanup = () =>
      setTimeout(() => {
        try {
          document.body.removeChild(iframe)
        } catch {
          /* ignore */
        }
      }, 1000)

    const doPrint = () => {
      if (printed) return
      printed = true
      try {
        iframe.contentWindow.focus()
        iframe.contentWindow.print()
      } catch {
        /* ignore */
      }
      cleanup()
    }

    // รอรูปถ่ายโหลดครบก่อนพิมพ์ (มี timeout กันค้าง)
    const imgs = Array.from(doc.images || [])
    let pending = imgs.length
    if (pending === 0) {
      setTimeout(doPrint, 300)
    } else {
      const settle = () => {
        pending -= 1
        if (pending <= 0) doPrint()
      }
      imgs.forEach((img) => {
        if (img.complete) settle()
        else {
          img.addEventListener('load', settle)
          img.addEventListener('error', settle)
        }
      })
      setTimeout(doPrint, 4000) // fallback ถ้ารูปโหลดช้า/ค้าง
    }
  }

  return (
    <AppShell navItems={navItems} roleLabel="QA">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">QA</h2>
          <p className="wh-subtitle">
            ตารางสรุปเครื่องที่ยืนยันแล้ว — WH สแกน Part Confirmation ตรงกับใบอนุญาต และ MFG สแกนได้ Matched
            (ดึงข้อมูลจาก WH + MFG + Master Data)
          </p>
        </div>
        <button
          className="qa-download-btn"
          onClick={handleExportPDF}
          disabled={loading || filtered.length === 0}
          title="พิมพ์/บันทึกเป็น PDF (Check Sheet)"
        >
          <ArrowDownTrayIcon className="size-4" /> Export PDF (Check Sheet)
        </button>
      </div>

      <div className="dash-stats-row">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>เครื่องที่ยืนยันแล้ว</span>
            <span className="dash-stat-icon dash-icon-green">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.total}</div>
          <div className="qa-stat-sub">ครบเงื่อนไข WH + MFG</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>มีรูปถ่ายยืนยัน</span>
            <span className="dash-stat-icon dash-icon-blue">
              <CameraIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.withPhoto}</div>
          <div className="qa-stat-sub">จาก {stats.total} เครื่อง</div>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="wh-table-card" style={{ overflowX: 'auto' }}>
        <div className="qa-table-toolbar">
          <div className="qa-table-toolbar-left">
            แสดง{' '}
            <div className="wh-pagesize-select">
              <SelectField
                value={pageSize}
                onChange={setPageSize}
                options={[
                  { value: 10, label: '10' },
                  { value: 25, label: '25' },
                  { value: 50, label: '50' },
                ]}
              />
            </div>{' '}
            รายการต่อหน้า
          </div>
          <div className="qa-table-toolbar-right">
            <input
              className="wh-search"
              type="text"
              placeholder="ค้นหา Part / Machine No / IT Controller / ใบอนุญาต / อินวอยซ์..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>

        <table className="wh-table">
          <thead>
            <tr>
              <th>ITEM</th>
              <th>Part Name</th>
              <th>Model</th>
              <th>Machine No</th>
              <th>Part No.</th>
              <th>Serial No.</th>
              <th>IT Controller No.</th>
              <th>IMEI</th>
              <th>ใบอนุญาตนำเข้า</th>
              <th>อินวอยซ์</th>
              <th>ผลเทียบใบอนุญาต</th>
              <th>รูปถ่าย</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={13} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              pageRows.map((r, i) => {
                const lic = licenseMatchMeta(r.matchStatus)
                return (
                  <tr key={r.itControllerNo}>
                    <td className="wh-cell-head" data-label="ITEM">
                      <strong>{(page - 1) * pageSize + i + 1}</strong>
                    </td>
                    <td data-label="Part Name">{dash(r.partName)}</td>
                    <td data-label="Model">{dash(r.model)}</td>
                    <td className="wh-cell-head" data-label="Machine No">
                      <strong>{dash(r.machineNo)}</strong>
                    </td>
                    <td className="il-mono" data-label="Part No.">
                      {dash(r.partNo)}
                    </td>
                    <td className="il-mono" data-label="Serial No.">
                      {dash(r.serialNo)}
                    </td>
                    <td className="il-mono" data-label="IT Controller No.">
                      {dash(r.itControllerNo)}
                    </td>
                    <td className="il-mono" data-label="IMEI">
                      {dash(r.imei)}
                    </td>
                    <td data-label="ใบอนุญาตนำเข้า">{dash(r.licenseNo)}</td>
                    <td data-label="อินวอยซ์">{dash(r.invoiceNo)}</td>
                    <td data-label="ผลเทียบใบอนุญาต">
                      <span className={lic.cls} title={r.matchMessage || ''}>
                        {lic.label}
                      </span>
                    </td>
                    <td data-label="รูปถ่าย">
                      {r.photoURL ? (
                        <button
                          type="button"
                          className="qa-photo-thumb"
                          onClick={() => setPhotoView(r.photoURL)}
                          title="ดูรูปถ่าย"
                          style={{ padding: 0, border: 'none', background: 'transparent' }}
                        >
                          <img
                            src={`${API_BASE_URL}${r.photoURL}`}
                            alt="รูปถ่ายป้าย"
                            loading="lazy"
                            style={{
                              width: 44,
                              height: 44,
                              objectFit: 'cover',
                              borderRadius: 6,
                              display: 'block',
                              cursor: 'pointer',
                            }}
                          />
                        </button>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td data-label="Status">
                      <span className="il-badge il-badge-ok">Matched</span>
                    </td>
                  </tr>
                )
              })}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={13} className="wh-empty-cell">
                  {confirmedRows.length === 0
                    ? 'ยังไม่มีเครื่องที่ครบเงื่อนไข — ต้องให้ WH ยืนยันตรงกับใบอนุญาต และ MFG สแกนได้ Matched ก่อน'
                    : 'ไม่พบรายการที่ค้นหา'}
                </td>
              </tr>
            )}
          </tbody>
        </table>

        {!loading && filtered.length > 0 && (
          <div className="qa-pagination">
            <button disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
              ก่อนหน้า
            </button>
            <span>
              หน้า {page} / {totalPages}
            </span>
            <button
              disabled={page >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            >
              ถัดไป
            </button>
          </div>
        )}
      </div>

      {/* ── Lightbox ดูรูปถ่ายป้าย ───────────────────────────────────────── */}
      {photoView && (
        <div className="wh-modal-overlay" onClick={() => setPhotoView(null)}>
          <div
            className="wh-modal"
            style={{ maxWidth: 520, textAlign: 'center' }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="wh-modal-actions" style={{ justifyContent: 'flex-end', marginTop: 0 }}>
              <button className="wh-modal-cancel" onClick={() => setPhotoView(null)} aria-label="ปิด">
                <XMarkIcon className="size-4" />
              </button>
            </div>
            <img
              src={`${API_BASE_URL}${photoView}`}
              alt="รูปถ่ายป้าย"
              style={{ maxWidth: '100%', borderRadius: 8 }}
            />
          </div>
        </div>
      )}
    </AppShell>
  )
}
