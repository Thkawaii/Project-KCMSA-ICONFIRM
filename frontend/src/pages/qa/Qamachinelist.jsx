import { useEffect, useMemo, useState } from 'react'
import AppShell from '../../components/AppShell.jsx'
import SelectField from '../../components/Selectfield.jsx'
import DatePickerField from '../../components/DatePickerField.jsx'
import {
  ArrowDownTrayIcon,
  CameraIcon,
  CheckCircleIcon,
  Squares2X2Icon,
  XMarkIcon,
} from '../../components/icons.jsx'
import { getQAConfirmedTable } from '../../api/qaConfirmed.js'
import { API_BASE_URL } from '../../api/client.js'
import { jsPDF } from 'jspdf'
import { autoTable } from 'jspdf-autotable'
import { SARABUN_REGULAR_BASE64, SARABUN_BOLD_BASE64 } from '../../lib/sarabunFont.js'
import { sheetToXlsxBlob, downloadBlob } from '../../lib/xlsx.js'

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

const pad2 = (n) => String(n).padStart(2, '0')

// ลงทะเบียนฟอนต์ Sarabun (รองรับภาษาไทย) ให้ jsPDF — ฟอนต์ default ของ jsPDF ไม่มีตัวไทย
function registerThaiFont(doc) {
  doc.addFileToVFS('Sarabun-Regular.ttf', SARABUN_REGULAR_BASE64)
  doc.addFont('Sarabun-Regular.ttf', 'Sarabun', 'normal')
  doc.addFileToVFS('Sarabun-Bold.ttf', SARABUN_BOLD_BASE64)
  doc.addFont('Sarabun-Bold.ttf', 'Sarabun', 'bold')
  doc.setFont('Sarabun', 'normal')
}

// แปลง URL รูปเป็น data URL เพื่อฝังลง PDF (ล้มเหลวได้ ไม่ทำให้การสร้าง PDF ทั้งฉบับพัง)
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

// หา format รูปจาก data URL (jsPDF ต้องระบุ format ให้ตรงตอน addImage)
// jsPDF ฝังได้เฉพาะ JPEG/PNG — ถ้าเป็นชนิดอื่น (webp, gif ฯลฯ) คืน null เพื่อให้ข้ามไป
// จะได้ไม่เสี่ยงทำให้ไฟล์ PDF พัง
function imageFormatFromDataURL(dataUrl) {
  const m = /^data:image\/(\w+);base64,/.exec(dataUrl || '')
  if (!m) return null
  const ext = m[1].toLowerCase()
  if (ext === 'jpg' || ext === 'jpeg') return 'JPEG'
  if (ext === 'png') return 'PNG'
  return null // ชนิดที่ jsPDF ฝังไม่ได้ — ข้าม
}

// วันที่วันนี้ในรูปแบบ YYYY-MM-DD (ตามเวลาเครื่องผู้ใช้) — ใช้กับ <input type="date">
function toYMD(d) {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
}

// ป้ายวันที่ภาษาไทยจากสตริง YYYY-MM-DD (เช่น "6 สิงหาคม 2569")
function thaiDateLabel(ymd) {
  const [y, m, d] = ymd.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('th-TH', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

// ดาวน์โหลด PDF แบบทนทาน — เดสก์ท็อปใช้ลิงก์ดาวน์โหลดปกติ,
// มือถือ/เว็บวิวบางตัวสั่งดาวน์โหลด blob แล้วได้ไฟล์ว่าง จึงเปิดในแท็บใหม่ให้แทน
// (ผู้ใช้กดบันทึก/แชร์/พิมพ์จากตัวเปิด PDF ได้เอง)
function savePdf(doc, filename) {
  const blob = doc.output('blob')
  const url = URL.createObjectURL(blob)
  const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent || '')

  if (isMobile) {
    // เปิดในแท็บ/ตัวอ่าน PDF ของเครื่อง — เชื่อถือได้กว่าการสั่งดาวน์โหลดตรงๆ บนมือถือ
    const win = window.open(url, '_blank')
    if (!win) {
      // ป็อปอัปโดนบล็อก → ถอยไปใช้ลิงก์ดาวน์โหลด
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

  // เผื่อตัวอ่าน PDF ยังต้องใช้ URL อยู่ ค่อยคืนหน่วยความจำหลังผ่านไปสักครู่
  setTimeout(() => URL.revokeObjectURL(url), 60_000)
}

export default function QAMachineList() {
  const [confirmedRows, setConfirmedRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [photoView, setPhotoView] = useState(null) // URL รูปที่กำลังเปิดดู

  const [search, setSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)
  const [exportingPDF, setExportingPDF] = useState(false)
  const [exportingExcel, setExportingExcel] = useState(false)

  // ตัวกรองวันที่ (วันที่ WH ยืนยัน) — เลือกได้ตามใจ ไม่บังคับ
  // ว่าง = ออก Check Sheet ของทุกวันรวมกัน / เลือกวัน = เฉพาะวันนั้น
  const [selectedDate, setSelectedDate] = useState('') // 'YYYY-MM-DD' หรือ ''

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
  }, [search, pageSize, selectedDate])

  // ขอบเขตวันที่ที่มีข้อมูลจริง — ใช้กำหนด min/max ให้ปฏิทิน และปุ่ม "ล่าสุด"
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
    setSelectedDate('')
  }

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

      if (selectedDate) {
        if (!r.confirmedAt) return false
        if (toYMD(new Date(r.confirmedAt)) !== selectedDate) return false
      }

      return true
    })
  }, [confirmedRows, search, selectedDate])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const pageRows = filtered.slice((page - 1) * pageSize, page * pageSize)

  // Export เป็น PDF (Check Sheet) — สร้าง PDF จริงด้วย jsPDF + autoTable (ข้อความเป็น vector
  // ไม่ใช่รูปถ่ายหน้าจอ) แล้วสั่งดาวน์โหลดไฟล์ทันที ไม่ต้องผ่านหน้าต่างพิมพ์ของเบราว์เซอร์
  async function handleExportPDF() {
    const list = filtered // ส่งออกตามที่กรอง/ค้นหา/วันที่เลือกอยู่ (ทุกหน้า)
    if (!list.length || exportingPDF) return

    setExportingPDF(true)
    try {
      // ดึงรูปถ่ายทั้งหมดมาแปลงเป็น data URL ก่อน (ถ้าโหลดไม่สำเร็จ ข้ามรูปนั้นไปเฉยๆ ไม่ทำให้ทั้งไฟล์พัง)
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

      const checkDateLabel = selectedDate ? thaiDateLabel(selectedDate) : 'ทั้งหมด'

      const PHOTO_COL_INDEX = 12
      const head = [
        [
          'ITEM',
          'Part Name',
          'Model',
          'Machine No',
          'Part No.',
          'Serial No.',
          'IT Controller No.',
          'IMEI',
          'ใบอนุญาตนำเข้า',
          'อินวอยซ์',
          'ส่งออกไปประเทศ',
          'ผลเทียบใบอนุญาต',
          'รูปถ่าย',
          'Status',
        ],
      ]
      const body = list.map((r, i) => [
        String(i + 1),
        dash(r.partName),
        dash(r.model),
        dash(r.machineNo),
        dash(r.partNo),
        dash(r.serialNo),
        dash(r.itControllerNo),
        dash(r.imei),
        dash(r.licenseNo),
        dash(r.invoiceNo),
        dash(r.exportCountry),
        licenseMatchMeta(r.matchStatus).label,
        '', // รูปถ่ายวาดเองใน didDrawCell
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
          13: { halign: 'center' },
        },
        didDrawPage: drawHeader,
        didDrawCell: (data) => {
          if (data.section !== 'body' || data.column.index !== PHOTO_COL_INDEX) return
          const dataUrl = photoDataUrls[data.row.index]
          if (!dataUrl) return
          const fmt = imageFormatFromDataURL(dataUrl)
          if (!fmt) return // ชนิดรูปที่ฝังไม่ได้ — ข้าม ไม่ให้ไฟล์พัง
          try {
            const pad = 1
            const size = Math.min(data.cell.height, data.cell.width) - pad * 2
            const x = data.cell.x + (data.cell.width - size) / 2
            const y = data.cell.y + (data.cell.height - size) / 2
            doc.addImage(dataUrl, fmt, x, y, size, size)
          } catch {
            // ข้ามรูปที่ฝังไม่สำเร็จ ไม่ทำให้ PDF ทั้งฉบับพัง
          }
        },
      })

      // เส้นเซ็นชื่อท้ายเอกสาร — วางในหน้าสุดท้าย ถ้าที่ไม่พอให้ขึ้นหน้าใหม่
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

      const fileDate = selectedDate || 'ทั้งหมด'
      savePdf(doc, `QA-CheckSheet-${fileDate}.pdf`)
    } catch (err) {
      console.error(err)
      alert('สร้าง PDF ไม่สำเร็จ กรุณาลองใหม่')
    } finally {
      setExportingPDF(false)
    }
  }

  // Export เป็น Excel (.xlsx) — ตารางข้อมูลล้วน (ไม่มีรูป) เปิดใน Excel/Sheets แก้ต่อได้
  // ใช้ตัวสร้าง .xlsx แบบไม่มี dependency (ดู lib/xlsx.js)
  function handleExportExcel() {
    const list = filtered // ส่งออกตามที่กรอง/ค้นหา/วันที่เลือกอยู่ (ทุกหน้า)
    if (!list.length || exportingExcel) return

    setExportingExcel(true)
    try {
      const header = [
        'ITEM',
        'Part Name',
        'Model',
        'Machine No',
        'Part No.',
        'Serial No.',
        'IT Controller No.',
        'IMEI',
        'ใบอนุญาตนำเข้า',
        'อินวอยซ์',
        'ส่งออกไปประเทศ',
        'ผลเทียบใบอนุญาต',
        'วันที่ยืนยัน',
        'Status',
      ]
      const body = list.map((r, i) => [
        i + 1,
        dash(r.partName),
        dash(r.model),
        dash(r.machineNo),
        dash(r.partNo),
        dash(r.serialNo),
        dash(r.itControllerNo),
        dash(r.imei),
        dash(r.licenseNo),
        dash(r.invoiceNo),
        dash(r.exportCountry),
        licenseMatchMeta(r.matchStatus).label,
        r.confirmedAt ? thaiDateLabel(toYMD(new Date(r.confirmedAt))) : '—',
        'Matched',
      ])

      const blob = sheetToXlsxBlob('QA Check Sheet', [header, ...body])
      const fileDate = selectedDate || 'ทั้งหมด'
      downloadBlob(blob, `QA-CheckSheet-${fileDate}.xlsx`)
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

      <div className="qa-date-filter-row">
        <div className="qa-date-field">
          <label>วันที่ยืนยัน (ไม่บังคับ)</label>
          <DatePickerField
            value={selectedDate}
            onChange={setSelectedDate}
            min={dateBounds.min}
            max={dateBounds.max}
            placeholder="— เลือกวันที่ —"
          />
        </div>
        {selectedDate && (
          <button type="button" className="qa-download-btn" onClick={clearDateFilter}>
            ล้างวันที่
          </button>
        )}
        <button
          className="qa-download-btn qa-export-btn"
          onClick={handleExportPDF}
          disabled={loading || filtered.length === 0 || exportingPDF}
          title={
            filtered.length === 0
              ? 'ไม่มีรายการให้ออก Check Sheet'
              : selectedDate
                ? `ดาวน์โหลด Check Sheet ของวันที่ ${thaiDateLabel(selectedDate)}`
                : 'ดาวน์โหลด Check Sheet ของรายการทั้งหมด'
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
              : selectedDate
                ? `ดาวน์โหลด Excel ของวันที่ ${thaiDateLabel(selectedDate)}`
                : 'ดาวน์โหลด Excel ของรายการทั้งหมด'
          }
        >
          <ArrowDownTrayIcon className="size-4" />
          {exportingExcel ? 'กำลังสร้าง Excel...' : 'Export Excel'}
        </button>
      </div>
      <p className="qa-stat-sub" style={{ marginTop: -8, marginBottom: 16 }}>
        {selectedDate
          ? `กำลังกรองเฉพาะวันที่ ${thaiDateLabel(selectedDate)} — พบ ${filtered.length} เครื่อง`
          : `ยังไม่ได้เลือกวัน — Export จะได้ทั้งหมด ${filtered.length} เครื่อง (เลือกวันเพื่อออกเฉพาะวันนั้น)`}
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
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={14} className="wh-empty-cell">
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
                  </tr>
                )
              })}
            {!loading && filtered.length === 0 && (
              <tr>
                <td colSpan={14} className="wh-empty-cell">
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
