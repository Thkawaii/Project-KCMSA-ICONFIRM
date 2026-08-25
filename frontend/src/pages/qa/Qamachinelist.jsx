import { useEffect, useMemo, useState } from 'react'
import AppShell from '../../components/AppShell.jsx'
import SelectField from '../../components/Selectfield.jsx'
import PeriodRangePicker from '../../components/PeriodRangePicker.jsx'
import { inPeriod, periodRangeLabel, periodFileTag } from '../../lib/dateRange.js'
import {
  ArrowDownTrayIcon,
  CameraIcon,
  CheckCircleIcon,
  ClockIcon,
  DocumentTextIcon,
  QrCodeIcon,
  Squares2X2Icon,
  TagIcon,
  WrenchScrewdriverIcon,
  XMarkIcon,
} from '../../components/icons.jsx'
import { getQAConfirmedTable } from '../../api/qaConfirmed.js'
import { API_BASE_URL } from '../../api/client.js'
import { jsPDF } from 'jspdf'
import { autoTable } from 'jspdf-autotable'
import { SARABUN_REGULAR_BASE64, SARABUN_BOLD_BASE64 } from '../../lib/sarabunFont.js'
import { buildStyledXlsxBlob, downloadBlob } from '../../lib/xlsx.js'

const navItems = [{ to: '/qa', label: 'ตรวจสอบ QA', icon: <CheckCircleIcon className="size-4" /> }]

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

const pad2 = (n) => String(n).padStart(2, '0')

function registerThaiFont(doc) {
  doc.addFileToVFS('Sarabun-Regular.ttf', SARABUN_REGULAR_BASE64)
  doc.addFont('Sarabun-Regular.ttf', 'Sarabun', 'normal')
  doc.addFileToVFS('Sarabun-Bold.ttf', SARABUN_BOLD_BASE64)
  doc.addFont('Sarabun-Bold.ttf', 'Sarabun', 'bold')
  doc.setFont('Sarabun', 'normal')
}

async function fetchImageAsDataURL(url) {
  try {
    const res = await fetch(url)
    if (!res.ok) return null
    const blob = await res.blob()
    return await new Promise((resolve) => {
      const reader = new FileReader()
      reader.onloadend = () => resolve(reader.result)
      reader.onerror = () => resolve(null)
      reader.readAsDataURL(blob)
    })
  } catch {
    return null
  }
}

async function fetchImageForXlsx(url) {
  try {
    const res = await fetch(url)
    if (!res.ok) return null
    const blob = await res.blob()
    let ext = ''
    const t = (blob.type || '').toLowerCase()
    if (t.includes('png')) ext = 'png'
    else if (t.includes('jpeg') || t.includes('jpg')) ext = 'jpeg'
    if (!ext) {
      if (/\.png(\?|$)/i.test(url)) ext = 'png'
      else if (/\.jpe?g(\?|$)/i.test(url)) ext = 'jpeg'
    }
    if (ext !== 'png' && ext !== 'jpeg') return null
    const buf = await blob.arrayBuffer()
    return { bytes: new Uint8Array(buf), ext }
  } catch {
    return null
  }
}

function imageFormatFromDataURL(dataUrl) {
  const m = /^data:image\/(\w+);base64,/.exec(dataUrl || '')
  if (!m) return null
  const ext = m[1].toLowerCase()
  if (ext === 'jpg' || ext === 'jpeg') return 'JPEG'
  if (ext === 'png') return 'PNG'
  return null
}

function toYMD(d) {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
}

function thaiDateLabel(ymd) {
  const [y, m, d] = ymd.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('th-TH', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

function savePdf(doc, filename) {
  const blob = doc.output('blob')
  const url = URL.createObjectURL(blob)
  const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent || '')

  if (isMobile) {
    const win = window.open(url, '_blank')
    if (!win) {
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      a.remove()
    }
  } else {
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
  }

  setTimeout(() => URL.revokeObjectURL(url), 60_000)
}

export default function QAMachineList() {
  const [confirmedRows, setConfirmedRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [photoView, setPhotoView] = useState(null)
  const [detailRow, setDetailRow] = useState(null)

  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)
  const [exportingPDF, setExportingPDF] = useState(false)
  const [exportingExcel, setExportingExcel] = useState(false)

  const [periodMode, setPeriodMode] = useState('all')
  const [periodAnchor, setPeriodAnchor] = useState('')

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

  useEffect(() => {
    function handleFocus() {
      loadConfirmed()
    }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [])

  useEffect(() => {
    setPage(1)
  }, [search, pageSize, periodMode, periodAnchor])

  const dateBounds = useMemo(() => {
    let min = null
    let max = null
    confirmedRows.forEach((r) => {
      if (!r.confirmedAt) return
      const ymd = toYMD(new Date(r.confirmedAt))
      if (min === null || ymd < min) min = ymd
      if (max === null || ymd > max) max = ymd
    })
    return { min, max }
  }, [confirmedRows])

  function clearDateFilter() {
    setPeriodMode('all')
    setPeriodAnchor('')
  }

  function handlePeriodModeChange(next) {
    setPeriodMode(next)
    if (next !== 'all' && !periodAnchor) {
      setPeriodAnchor(dateBounds.max || toYMD(new Date()))
    }
  }

  const periodLabel = periodMode === 'all' ? 'ทั้งหมด' : periodRangeLabel(periodMode, periodAnchor)
  const periodTag = periodFileTag(periodMode, periodAnchor)

  const stats = useMemo(() => {
    const total = confirmedRows.length
    const withPhoto = confirmedRows.filter((r) => r.photoURL).length
    return { total, withPhoto }
  }, [confirmedRows])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return confirmedRows.filter((r) => {
      if (q) {
        const matchesSearch = [
          r.partName,
          r.model,
          r.asmModel,
          r.specCode,
          r.specDetail,
          r.itDevice,
          r.machineNo,
          r.partNo,
          r.serialNo,
          r.itControllerNo,
          r.imei,
          r.licenseNo,
          r.invoiceNo,
        ].some((v) => (v || '').toLowerCase().includes(q))
        if (!matchesSearch) return false
      }

      if (periodMode !== 'all') {
        if (!r.confirmedAt) return false
        if (!inPeriod(r.confirmedAt, periodMode, periodAnchor)) return false
      }

      return true
    })
  }, [confirmedRows, search, periodMode, periodAnchor])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const pageRows = filtered.slice((page - 1) * pageSize, page * pageSize)

  async function handleExportPDF() {
    const list = filtered
    if (!list.length || exportingPDF) return

    setExportingPDF(true)
    try {
      const photoDataUrls = await Promise.all(
        list.map((r) => (r.photoURL ? fetchImageAsDataURL(`${API_BASE_URL}${r.photoURL}`) : Promise.resolve(null)))
      )

      const doc = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4' })
      registerThaiFont(doc)

      const now = new Date()
      const printedStr = `${now.toLocaleDateString('th-TH', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })} ${now.toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })}`

      const checkDateLabel = periodLabel

      const PHOTO_COL_INDEX = 11
      const head = [
        [
          'ITEM',
          'Part Name',
          'Model (Assembly)',
          'Spec Code',
          'IT device',
          'Machine No',
          'Part No.',
          'Serial No.',
          'IT Controller No.',
          'ส่งออกไปประเทศ',
          'ผลเทียบใบอนุญาต',
          'รูปถ่าย',
          'Status',
        ],
      ]
      const body = list.map((r, i) => [
        String(i + 1),
        dash(r.partName),
        dash(r.asmModel),
        dash(r.specCode),
        dash(r.itDevice),
        dash(r.machineNo),
        dash(r.partNo),
        dash(r.serialNo),
        dash(r.itControllerNo),
        dash(r.exportCountry),
        licenseMatchMeta(r.matchStatus).label,
        '',
        'Matched',
      ])

      const drawHeader = () => {
        doc.setFont('Sarabun', 'bold')
        doc.setFontSize(16)
        doc.setTextColor(15, 23, 42)
        doc.text('QA Check Sheet — ใบตรวจสอบ QA', 10, 14)

        doc.setFont('Sarabun', 'normal')
        doc.setFontSize(9)
        doc.setTextColor(71, 85, 105)

        const pageWidth = doc.internal.pageSize.getWidth()
        doc.text(`วันที่ตรวจสอบ (Check Sheet): ${checkDateLabel}`, pageWidth - 10, 12, { align: 'right' })
        doc.text(`วันที่พิมพ์: ${printedStr}`, pageWidth - 10, 17, { align: 'right' })
        doc.text(`จำนวน: ${list.length} เครื่อง`, pageWidth - 10, 22, { align: 'right' })
      }

      autoTable(doc, {
        head,
        body,
        startY: 26,
        margin: { top: 26, left: 8, right: 8, bottom: 12 },
        styles: {
          font: 'Sarabun',
          fontSize: 8,
          cellPadding: 1.5,
          lineColor: [148, 163, 184],
          lineWidth: 0.1,
          valign: 'middle',
        },
        headStyles: {
          font: 'Sarabun',
          fontStyle: 'bold',
          fillColor: [241, 245, 249],
          textColor: [15, 23, 42],
          halign: 'center',
        },
        columnStyles: {
          0: { halign: 'center', cellWidth: 10 },
          [PHOTO_COL_INDEX]: { halign: 'center', cellWidth: 16, minCellHeight: 14 },
          12: { halign: 'center' },
        },
        didDrawPage: drawHeader,
        didDrawCell: (data) => {
          if (data.section !== 'body' || data.column.index !== PHOTO_COL_INDEX) return
          const dataUrl = photoDataUrls[data.row.index]
          if (!dataUrl) return
          const fmt = imageFormatFromDataURL(dataUrl)
          if (!fmt) return
          try {
            const pad = 1
            const size = Math.min(data.cell.height, data.cell.width) - pad * 2
            const x = data.cell.x + (data.cell.width - size) / 2
            const y = data.cell.y + (data.cell.height - size) / 2
            doc.addImage(dataUrl, fmt, x, y, size, size)
          } catch {
          }
        },
      })

      const pageWidth = doc.internal.pageSize.getWidth()
      const pageHeight = doc.internal.pageSize.getHeight()
      let signY = doc.lastAutoTable.finalY + 18
      if (signY + 14 > pageHeight - 10) {
        doc.addPage()
        drawHeader()
        signY = 40
      }

      const cols = [
        { label: 'ผู้ตรวจสอบ (QA)', cx: pageWidth * 0.2 },
        { label: 'ผู้อนุมัติ', cx: pageWidth * 0.5 },
        { label: 'วันที่', cx: pageWidth * 0.8 },
      ]
      doc.setFont('Sarabun', 'normal')
      doc.setFontSize(9)
      doc.setTextColor(15, 23, 42)
      cols.forEach((c) => {
        doc.line(c.cx - 25, signY, c.cx + 25, signY)
        doc.text(c.label, c.cx, signY + 5, { align: 'center' })
      })

      savePdf(doc, `QA-CheckSheet-${periodTag}.pdf`)
    } catch (err) {
      console.error(err)
      alert('สร้าง PDF ไม่สำเร็จ กรุณาลองใหม่')
    } finally {
      setExportingPDF(false)
    }
  }

  async function handleExportExcel() {
    const list = filtered
    if (!list.length || exportingExcel) return

    setExportingExcel(true)
    try {
      const photos = await Promise.all(
        list.map((r) => (r.photoURL ? fetchImageForXlsx(`${API_BASE_URL}${r.photoURL}`) : Promise.resolve(null)))
      )

      const columns = [
        { key: 'item', header: 'ITEM', type: 'number', width: 8 },
        { key: 'partName', header: 'Part Name', type: 'text' },
        { key: 'model', header: 'Model', type: 'center', width: 12 },
        { key: 'asmModel', header: 'Model (Assembly)', type: 'center', width: 14 },
        { key: 'specCode', header: 'Spec Code', type: 'center', width: 14 },
        { key: 'specDetail', header: 'Specification Detail', type: 'text', width: 20 },
        { key: 'itDevice', header: 'IT device', type: 'center', width: 14 },
        { key: 'machineNo', header: 'Machine No', type: 'text' },
        { key: 'partNo', header: 'Part No.', type: 'text' },
        { key: 'serialNo', header: 'Serial No.', type: 'text' },
        { key: 'itControllerNo', header: 'IT Controller No.', type: 'text' },
        { key: 'imei', header: 'IMEI', type: 'text' },
        { key: 'licenseNo', header: 'ใบอนุญาตนำเข้า', type: 'text' },
        { key: 'invoiceNo', header: 'อินวอยซ์', type: 'text' },
        { key: 'exportCountry', header: 'ส่งออกไปประเทศ', type: 'center', width: 16 },
        { key: 'matchStatus', header: 'ผลเทียบใบอนุญาต', type: 'center', width: 16 },
        { key: 'confirmedAt', header: 'วันที่ยืนยัน', type: 'center', width: 14 },
        { key: 'photo', header: 'รูปถ่าย', type: 'image', width: 14 },
        { key: 'status', header: 'Status', type: 'center', width: 12 },
      ]
      const rows = list.map((r, i) => ({
        item: i + 1,
        partName: dash(r.partName),
        model: dash(r.model),
        asmModel: dash(r.asmModel),
        specCode: dash(r.specCode),
        specDetail: dash(r.specDetail),
        itDevice: dash(r.itDevice),
        machineNo: dash(r.machineNo),
        partNo: dash(r.partNo),
        serialNo: dash(r.serialNo),
        itControllerNo: dash(r.itControllerNo),
        imei: dash(r.imei),
        licenseNo: dash(r.licenseNo),
        invoiceNo: dash(r.invoiceNo),
        exportCountry: dash(r.exportCountry),
        matchStatus: licenseMatchMeta(r.matchStatus).label,
        confirmedAt: r.confirmedAt ? thaiDateLabel(toYMD(new Date(r.confirmedAt))) : '—',
        photo: photos[i] || null,
        status: 'Matched',
      }))

      const blob = buildStyledXlsxBlob({ sheetName: 'QA Check Sheet', columns, rows })
      downloadBlob(blob, `QA-CheckSheet-${periodTag}.xlsx`)
    } catch (err) {
      console.error(err)
      alert('สร้าง Excel ไม่สำเร็จ กรุณาลองใหม่')
    } finally {
      setExportingExcel(false)
    }
  }

  return (
    <AppShell navItems={navItems} roleLabel="QA">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">QA</h2>
        </div>
      </div>

      <div className="qa-filter-card">
        <div className="qa-filter-top">
          <PeriodRangePicker
            mode={periodMode}
            onModeChange={handlePeriodModeChange}
            anchor={periodAnchor}
            onAnchorChange={setPeriodAnchor}
            min={dateBounds.min}
            max={dateBounds.max}
            label="ช่วงวันที่ยืนยัน (สำหรับ Check Sheet)"
            countLabel={`${filtered.length} เครื่อง`}
          />
          {periodMode !== 'all' && (
            <button type="button" className="qa-clear-btn" onClick={clearDateFilter}>
              <XMarkIcon className="size-4" />
              ล้างช่วง
            </button>
          )}
        </div>

        <div className="qa-export-actions">
          <button
            className="qa-download-btn qa-export-btn"
            onClick={handleExportPDF}
            disabled={loading || filtered.length === 0 || exportingPDF}
            title={
              filtered.length === 0
                ? 'ไม่มีรายการให้ออก Check Sheet'
                : `ดาวน์โหลด Check Sheet — ช่วง ${periodLabel}`
            }
          >
            <ArrowDownTrayIcon className="size-4" />
            {exportingPDF ? 'กำลังสร้าง PDF...' : 'Export PDF (Check Sheet)'}
          </button>
          <button
            className="qa-download-btn qa-export-btn qa-export-btn-excel"
            onClick={handleExportExcel}
            disabled={loading || filtered.length === 0 || exportingExcel}
            title={
              filtered.length === 0
                ? 'ไม่มีรายการให้ออก Excel'
                : `ดาวน์โหลด Excel — ช่วง ${periodLabel}`
            }
          >
            <ArrowDownTrayIcon className="size-4" />
            {exportingExcel ? 'กำลังสร้าง Excel...' : 'Export Excel'}
          </button>
        </div>
      </div>
      <p className="qa-stat-sub" style={{ marginTop: 10, marginBottom: 16 }}>
        {periodMode === 'all'
          ? `ยังไม่ได้เลือกช่วง — Export จะได้ทั้งหมด ${filtered.length} เครื่อง (เลือกช่วงเพื่อออกเฉพาะรายวัน/สัปดาห์/เดือน/ปี)`
          : `กำลังกรองช่วง ${periodLabel} — พบ ${filtered.length} เครื่อง (Export PDF/Excel จะได้เฉพาะช่วงนี้)`}
      </p>

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
              <th>Model (Assembly)</th>
              <th>Spec Code</th>
              <th>Specification Detail</th>
              <th>IT device</th>
              <th>Machine No</th>
              <th>Part No.</th>
              <th>Serial No.</th>
              <th>IT Controller No.</th>
              <th>IMEI</th>
              <th>ใบอนุญาตนำเข้า</th>
              <th>อินวอยซ์</th>
              <th>ส่งออกไปประเทศ</th>
              <th>ผลเทียบใบอนุญาต</th>
              <th>รูปถ่าย</th>
              <th>Status</th>
              <th>รายละเอียด</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={19} className="wh-empty-cell">
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
                    <td data-label="Model (Assembly)">{dash(r.asmModel)}</td>
                    <td data-label="Spec Code">{dash(r.specCode)}</td>
                    <td data-label="Specification Detail">{dash(r.specDetail)}</td>
                    <td data-label="IT device">{dash(r.itDevice)}</td>
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
                    <td data-label="ส่งออกไปประเทศ">{dash(r.exportCountry)}</td>
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
                    <td className="wh-cell-action" data-label="รายละเอียด">
                      <button className="tsf-action-btn" onClick={() => setDetailRow(r)}>
                        รายละเอียด
                      </button>
                    </td>
                  </tr>
                )
              })}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={19} className="wh-empty-cell">
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

      {detailRow && (
        <div className="wh-modal-overlay" onClick={() => setDetailRow(null)}>
          <div className="wh-modal wh-detail-modal" onClick={(e) => e.stopPropagation()}>
            <button
              type="button"
              className="wh-detail-close"
              onClick={() => setDetailRow(null)}
              aria-label="ปิด"
            >
              <XMarkIcon className="size-4" />
            </button>

            <div className="wh-detail-header">
              <span className="wh-detail-header-icon">
                <CheckCircleIcon className="size-5" />
              </span>
              <div>
                <h3 className="wh-modal-title">รายละเอียดการยืนยัน</h3>
                <span className="wh-detail-header-sub">{dash(detailRow.partName)}</span>
              </div>
            </div>

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <DocumentTextIcon className="size-4" /> ข้อมูลชิ้นงาน
              </span>
              <div className="wh-detail-grid">
                <div className="wh-detail-item">
                  <span className="wh-detail-label">Machine No</span>
                  <span className="wh-detail-value mono">{dash(detailRow.machineNo)}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">IT Controller No.</span>
                  <span className="wh-detail-value mono">{dash(detailRow.itControllerNo)}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">Part No.</span>
                  <span className="wh-detail-value mono">{dash(detailRow.partNo)}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">Serial No.</span>
                  <span className="wh-detail-value mono">{dash(detailRow.serialNo)}</span>
                </div>
              </div>
            </div>

            <div className="wh-detail-divider" />

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <QrCodeIcon className="size-4" /> ตอนสแกน (ยืนยันคลัง — WH)
              </span>
              <div className="wh-detail-grid">
                <div className="wh-detail-item">
                  <span className="wh-detail-label">สแกนโดย</span>
                  <span className="wh-detail-value">{dash(detailRow.checkedByWH)}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">เมื่อ</span>
                  <span className="wh-detail-value">
                    {detailRow.checkedAtWH
                      ? new Date(detailRow.checkedAtWH).toLocaleString('th-TH')
                      : '—'}
                  </span>
                </div>
              </div>
            </div>

            <div className="wh-detail-divider" />

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <WrenchScrewdriverIcon className="size-4" /> ตอนประกอบ (ผลิต — MFG)
              </span>
              <div className="wh-detail-grid">
                <div className="wh-detail-item">
                  <span className="wh-detail-label">ประกอบโดย</span>
                  <span className="wh-detail-value">{dash(detailRow.assembledBy)}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">เมื่อ</span>
                  <span className="wh-detail-value">
                    {detailRow.assembledAt
                      ? new Date(detailRow.assembledAt).toLocaleString('th-TH')
                      : '—'}
                  </span>
                </div>
              </div>
            </div>

            <div className="wh-detail-meta">
              <span>
                <TagIcon className="size-3.5" /> ใบอนุญาต {dash(detailRow.licenseNo)}
              </span>
              <span>
                <ClockIcon className="size-3.5" /> อินวอยซ์ {dash(detailRow.invoiceNo)}
              </span>
            </div>

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={() => setDetailRow(null)}>
                ปิด
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  )
}
