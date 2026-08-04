import { useEffect, useMemo, useRef, useState } from 'react'
import { getPartChecks, scanPartCheck, deletePartCheck } from '../api/partcheck.js'
import { getImportLicenseItems } from '../api/importLicense.js'
import { scanStep, scanSelect, scanLoading, scanSuccessToast, scanErrorAlert, scanClose } from '../lib/scanPopup.js'
import { confirmDelete, toastSuccess, toastError } from '../lib/toast.js'
import {
  CheckIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  MinusIcon,
  XMarkIcon,
} from '../components/icons.jsx'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { WH_NAV_ITEMS } from './Importlicensepage.jsx'

// รูปบาร์โค้ดอ้างอิงของแต่ละพาร์ท (Vite จะ bundle ให้อัตโนมัติ)
// IT Controller ยุบรวมป้าย P/N + S/N เหลือใบเดียว ใช้รูปบาร์โค้ดตามที่ส่งมา
import bcItc from '../assets/barcodes/IT_Controller.gif'
import bcSwingSn from '../assets/barcodes/Swing_Motor__SN_.gif'
import bcPumpSn from '../assets/barcodes/Pump_Assy_HYD__SN_.gif'
import bcMotorSn from '../assets/barcodes/Motor_Propel__SN_.gif'
import bcValveSn from '../assets/barcodes/Control_Valve__SN_.gif'

const TAG_TYPES = [
  { code: 'MC', label: 'Machine', needsPN: false },
  { code: 'ITC', label: 'IT Controller', needsPN: true },
  { code: 'CV', label: 'Control Valve', needsPN: false },
  { code: 'SM', label: 'Swing Motor', needsPN: false },
  { code: 'MP', label: 'Motor Propel', needsPN: false },
  { code: 'PH', label: 'Pump Assy HYD', needsPN: false },
]

// ชนิดพาร์ทที่เลือกได้ในฟอร์ม (ไม่รวม Machine เพราะ Machine คือ tag ที่ใช้ระบุตัวเครื่อง)
// IT Controller ต้องสแกนทั้ง P/N และหมายเลขเครื่อง ส่วนพาร์ทอื่นสแกนเฉพาะ S/N
const PART_TYPES = TAG_TYPES.filter((t) => t.code !== 'MC')

function tagLabel(code) {
  return TAG_TYPES.find((t) => t.code === code)?.label || code || '—'
}

// firstToken เอาเฉพาะ "ส่วนแรก" ของค่าที่สแกนมา ก่อนช่องว่างชุดแรก
//
// บาร์โค้ด P/N และ S/N บางป้ายพ่วงรหัสชุดที่สองต่อท้าย คั่นด้วยช่องว่าง เช่น
//   ยิง P/N ได้  "YN22E00849FA      878250023501"  -> ต้องการ  "YN22E00849FA"
//   ยิง S/N ได้  "KQ3000045363      300234031527950" -> ต้องการ  "KQ3000045363"
// P/N / S/N จริงไม่มีช่องว่างในตัวเอง จึงตัดตั้งแต่ช่องว่างแรกได้อย่างปลอดภัย
// (กรอกมือแบบไม่มีช่องว่างก็คืนค่าเดิมไม่เปลี่ยน)
function firstToken(v) {
  if (!v) return ''
  return String(v).trim().split(/\s+/)[0] || ''
}

// ป้ายผลการเทียบกับบัญชีใบอนุญาตนำเข้า (ค่าตรงกับค่าคงที่ฝั่ง backend)
const MATCH_LABELS = {
  MATCH: { Icon: CheckIcon, text: 'ตรงกับใบอนุญาต', cls: 'il-badge-ok' },
  NOT_FOUND: { Icon: XMarkIcon, text: 'ไม่พบในใบอนุญาต', cls: 'il-badge-bad' },
  WRONG_INVOICE: { Icon: ExclamationTriangleIcon, text: 'คนละอินวอยซ์', cls: 'il-badge-warn' },
  WRONG_PRODNO: { Icon: ExclamationTriangleIcon, text: 'หมายเลขการผลิตไม่ตรง', cls: 'il-badge-warn' },
  DUPLICATE: { Icon: ExclamationTriangleIcon, text: 'ยืนยันซ้ำ', cls: 'il-badge-warn' },
  NOT_REQUIRED: { Icon: MinusIcon, text: 'ไม่ต้องเทียบ', cls: 'il-badge-muted' },
}

function matchBadge(status) {
  const m = MATCH_LABELS[status] || MATCH_LABELS.NOT_REQUIRED
  return (
    <span className={'il-badge ' + m.cls}>
      <m.Icon className="inline size-3.5 align-text-bottom" /> {m.text}
    </span>
  )
}

// การ์ดบาร์โค้ดที่โชว์บนหน้า Part Confirmation (ตามรูป label จริง)
const BARCODE_CARDS = [
  { partType: 'ITC', title: 'IT Controller', caption: 'IT Controller', img: bcItc, kind: 'P/N + S/N' },
  { partType: 'SM', title: 'Swing Motor', caption: 'Swing Motor (S/N)', img: bcSwingSn, kind: 'S/N' },
  { partType: 'PH', title: 'Pump Assy HYD', caption: 'Pump Assy HYD (S/N)', img: bcPumpSn, kind: 'S/N' },
  { partType: 'MP', title: 'Motor Propel', caption: 'Motor Propel (S/N)', img: bcMotorSn, kind: 'S/N' },
  { partType: 'CV', title: 'Control Valve', caption: 'Control Valve (S/N)', img: bcValveSn, kind: 'S/N' },
]

export default function WHPartConfirmationPage() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  // ── ตารางอ้างอิง: บัญชีใบอนุญาตนำเข้า ────────────────────────────────────
  const [licenseItems, setLicenseItems] = useState([])
  const [licenseTab, setLicenseTab] = useState('all')
  const [licenseModel, setLicenseModel] = useState('all') // ตัวกรอง แบบ/รุ่น
  const [licensePageSize, setLicensePageSize] = useState(10) // จำนวนต่อหน้าของตารางเทียบ
  const [licensePage, setLicensePage] = useState(1)
  const [highlightId, setHighlightId] = useState(null)

  // ผลสแกนล่าสุด (ไว้โชว์แถบสรุปบนหน้า)
  const [lastScan, setLastScan] = useState(null)

  const [dateTab, setDateTab] = useState('all')
  const [search, setSearch] = useState('')
  const [matchFilter, setMatchFilter] = useState('all') // ตัวกรอง ผลเทียบใบอนุญาต
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  const [detailRow, setDetailRow] = useState(null)

  // busyRef = true ระหว่างที่ flow สแกนกำลังทำงาน (กันตัวดักสแกนเนอร์ยิงซ้อน)
  const busyRef = useRef(false)
  // เก็บฟังก์ชันจัดการเมื่อสแกนเนอร์ยิง (อัปเดตทุก render กัน closure ค้าง)
  const fireRef = useRef(() => {})

  async function loadRows() {
    setLoading(true)
    setLoadError('')
    try {
      const [checks, items] = await Promise.all([getPartChecks(), getImportLicenseItems()])
      setRows(checks || [])
      setLicenseItems(items || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRows()
  }, [])

  useEffect(() => {
    setPage(1)
  }, [dateTab, search, matchFilter, pageSize])

  useEffect(() => {
    setLicensePage(1)
  }, [licenseTab, licenseModel, licensePageSize])

  // ลบรายการประวัติการสแกน — กดได้เฉพาะแถวที่ผลเทียบเป็น "ไม่พบในใบอนุญาต" (NOT_FOUND)
  async function handleDeleteCheck(row) {
    const label = `${tagLabel(row.PartType)} — ${row.SN || row.PN || '#' + row.ID}`
    const ok = await confirmDelete({ text: `ลบรายการสแกน ${label} ออกจากประวัติ? กู้คืนไม่ได้` })
    if (!ok) return

    try {
      await deletePartCheck(row.ID)
      toastSuccess(`ลบรายการ ${label} แล้ว`)
      await loadRows()
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ')
    }
  }

  // ── SCAN FLOW (SweetAlert) ───────────────────────────────────────────────
  // WH ไม่มี TAG เครื่อง — สแกน "หรือกรอก" แค่ P/N / S/N ของพาร์ทเท่านั้น
  // ITC: P/N + S/N -> ระบบเทียบกับ master data เพื่อดึงหมายเลขเครื่อง
  //      (IT Controller No.) -> ลิงก์อินวอยซ์ + เทียบบัญชีใบอนุญาตนำเข้า -> บันทึก
  // พาร์ทอื่น: S/N -> บันทึก (ไม่ต้องเทียบบัญชี)
  //
  // ไม่ต้องเลือก Invoice/ล็อตก่อนสแกนแล้ว — ระบบเทียบกับ "ทุกใบอนุญาต" ในบัญชี
  // เสมอ โดยหาเครื่องจากหมายเลขเครื่องที่ดึงมาได้โดยตรง (ดู matchImportLicense
  // ฝั่ง backend) จึงไม่มีแนวคิดล็อตที่ต้องเลือกไว้ล่วงหน้าอีกต่อไป
  async function runScanFlow(partTypeCode) {
    if (!partTypeCode || busyRef.current) return
    const part = PART_TYPES.find((t) => t.code === partTypeCode)
    if (!part) return

    const partLabel = part.label
    const isITC = part.code === 'ITC'
    const needsPN = Boolean(part.needsPN)

    busyRef.current = true
    // ข้อความ toast แจ้ง "สำเร็จ" (ถ้ามี) — เก็บไว้เด้ง "หลัง" ปลด busy เสมอ
    // เพราะ toast มี timer 3 วินาที ถ้า await ใต้ busy จะกันตัวดักสแกนไว้ตลอด 3 วิ
    // ทำให้บาร์โค้ดพาร์ทถัดไปที่ยิงระหว่างนั้นถูกกลืน/ตัดครึ่ง แล้วเดาชนิดพาร์ทผิดเป็น ITC
    let successToast = null
    try {
      // 1) สแกน "หรือกรอก" P/N (เฉพาะพาร์ทที่ต้องมี P/N เช่น IT Controller)
      let pn = ''
      if (needsPN) {
        pn = firstToken(
          await scanStep({
            title: `${partLabel}( P/N)`,
            placeholder: 'ยิงบาร์โค้ด หรือพิมพ์ P/N แล้วกดปุ่ม',
            html: '',
          }),
        )
        if (!pn) return
      }

      // 2) สแกน "หรือกรอก" S/N -> ขั้นสุดท้าย บันทึกเลย
      //    ITC: ระบบจะเอา P/N + S/N ไปเทียบ master data เพื่อดึงหมายเลขเครื่อง
      //         (IT Controller) แล้วลิงก์อินวอยซ์ + เทียบบัญชีใบอนุญาตนำเข้าให้เอง
      const sn = firstToken(
        await scanStep({
          title: `${partLabel}( S/N)`,
          placeholder: 'ยิงบาร์โค้ด หรือพิมพ์ S/N แล้วกดปุ่ม',
          html: needsPN
            ? `<div class="scan-popup-hint">P/N: <b>${pn}</b></div>`
            : '',
          confirmText: 'บันทึก',
        }),
      )
      if (!sn) return

      // 3) ส่งขึ้น API — backend เทียบกับบัญชีแล้วตอบผลกลับมาในทีเดียว
      scanLoading('กำลังตรวจสอบกับบัญชีใบอนุญาต...')
      try {
        const res = await scanPartCheck({
          machineTag: '', // WH ไม่มี TAG เครื่อง
          partType: partTypeCode,
          pn: needsPN ? pn : '',
          sn,
          productionNo: '',
          invoiceNo: '', // ไม่มีล็อตให้เลือกแล้ว — เทียบกับทุกใบอนุญาตในบัญชีเสมอ
        })

        const check = res.check || res

        setLastScan({
          machineTag: check.Tag || '',
          partType: check.PartType || partTypeCode,
          pn: needsPN ? pn : '',
          sn,
          machineNo: check.MachineNo || '',
          productionNo: check.ProductionNo || '',
          matchStatus: check.MatchStatus,
          message: check.MatchMessage || res.message,
          at: check.CheckedDatetime || new Date().toISOString(),
        })

        // ไฮไลต์แถวในตารางที่เพิ่งจับคู่ได้ ให้เห็นด้วยตาว่าไปโดนแถวไหน
        if (res.item?.ID) {
          setHighlightId(res.item.ID)
          setTimeout(() => setHighlightId(null), 6000)
        }

        if (res.matched) {
          // toast แจ้งสำเร็จ — เก็บไว้เด้งหลังปลด busy (ห้าม await ใต้ busy)
          successToast = `ตรงกับบัญชี: ${sn}`
        } else if (isITC) {
          // ITC ไม่ตรงบัญชี = ต้องให้ผู้ใช้กดรับทราบ จึง await ทั้งที่ยัง busy อยู่
          // (กันไม่ให้บาร์โค้ดถัดไปเปิด flow ใหม่ทับกล่อง error ที่กำลังโชว์)
          await scanErrorAlert(check.MatchMessage || res.message || 'ไม่ตรงกับบัญชีใบอนุญาตนำเข้า')
        } else {
          successToast = `บันทึกแล้ว: ${tagLabel(check.PartType)} — ${sn}`
        }

        await loadRows()
      } catch (err) {
        await scanErrorAlert(err.message || 'บันทึกไม่สำเร็จ')
      }
    } finally {
      busyRef.current = false
      scanClose() // ปิด popup loading ที่อาจค้างอยู่ ก่อนเด้ง toast/รับสแกนตัวถัดไป
    }

    // เด้ง toast แจ้งสำเร็จ "หลัง" ปลด busy แล้ว — ตอนนี้ตัวดักสแกนพร้อมรับบาร์โค้ด
    // พาร์ทถัดไปแบบเต็มตั้งแต่ตัวอักษรแรก ไม่โดน 3 วินาทีของ toast กันไว้อีก
    if (successToast) scanSuccessToast(successToast)
  }

  // ระบุชนิดพาร์ทจากข้อความบาร์โค้ดที่ยิงมา — คืน code พาร์ท ถ้าดูออก, หรือ null ถ้าดูไม่ออก
  // ป้ายบาร์โค้ดอ้างอิงบนการ์ด (และป้ายของจริงส่วนใหญ่) มีคำกำกับ เช่น
  //   SWING / PROPEL / PUMP / HYD / VALVE / CONTROLLER อยู่ในเนื้อบาร์โค้ด -> จับด้วยคีย์เวิร์ดได้
  // ⚠️ ห้าม default เป็น ITC เด็ดขาด: ถ้าดูไม่ออก (เช่น ยิงสติกเกอร์ S/N จริงที่เป็นเลขล้วน
  //    ไม่มีคำกำกับ หรือบาร์โค้ดถูกอ่านมาไม่ครบ) การเดาเป็น ITC จะทำให้เด้ง flow "IT Controller"
  //    ผิดพาร์ท (นี่คือบั๊กที่เจอ: ยิง Control Valve แล้วขึ้น popup IT Controller( P/N))
  //    -> คืน null แล้วให้ผู้ใช้เลือกชนิดพาร์ทเองแทน
  function detectPartType(raw) {
    const s = (raw || '').toUpperCase()
    if (s.includes('SWING')) return 'SM'
    if (s.includes('PROPEL')) return 'MP'
    if (s.includes('PUMP') || s.includes('HYD')) return 'PH'
    if (s.includes('CONTROL VALVE') || s.includes('VALVE')) return 'CV'
    if (s.includes('CONTROLLER')) return 'ITC'
    return null
  }

  // เมื่อสแกนเนอร์ยิง 1 ครั้ง (ไม่มี popup เปิดอยู่):
  //  - ถ้าระบุชนิดพาร์ทจากบาร์โค้ดได้ -> เข้า flow ของพาร์ทนั้นทันที
  //  - ถ้าระบุไม่ได้ -> เด้งตัวเลือกให้ผู้ใช้ยืนยันชนิดพาร์ทก่อน (ไม่เดามั่วเป็น ITC อีก)
  async function handleScannerFire(code) {
    if (busyRef.current) return

    let partType = detectPartType(code)

    if (!partType) {
      // กัน flow/สแกนซ้อนระหว่างเปิดตัวเลือก
      busyRef.current = true
      let picked = null
      try {
        picked = await scanSelect({
          title: 'เลือกชนิดพาร์ทที่จะยืนยัน',
          html: `<div class="scan-popup-hint">บาร์โค้ดที่ยิง: <b>${code}</b></div>`,
          options: PART_TYPES.map((p) => ({ value: p.code, label: p.label })),
        })
      } finally {
        busyRef.current = false
      }
      if (!picked) return // ผู้ใช้ยกเลิก
      partType = picked
    }

    runScanFlow(partType)
  }

  fireRef.current = handleScannerFire

  // ตัวดักสัญญาณเครื่องสแกนเนอร์ระดับหน้าเว็บ:
  // สแกนเนอร์ = คีย์บอร์ดที่พิมพ์เร็วมาก (เว้นแต่ละตัว < ~50ms)
  // เดิมจะ return ทิ้งถ้าโฟกัสอยู่ในช่อง input ทำให้ถ้าเคอร์เซอร์ค้างในช่องค้นหา
  // บาร์โค้ดจะไหลลงช่องนั้นแทนที่จะเด้ง popup — เวอร์ชันนี้จับจาก "ความเร็วการยิง" แทน
  // จึงเด้ง popup ได้ไม่ว่าโฟกัสอยู่ตรงไหน + flush ด้วย timeout เผื่อเครื่องไม่ได้ตั้ง Enter suffix
  useEffect(() => {
    let buffer = ''
    let lastTime = 0
    let flushTimer = null
    // startedClean = true ก็ต่อเมื่อ buffer เริ่มนับจาก "จังหวะว่างจริง ๆ" (เว้นเกิน 50ms)
    // ไม่ใช่เศษท้ายของบาร์โค้ดที่ผู้ใช้ยิงคร่อมช่วงที่ flow ยัง busy อยู่ (popup เปิดค้าง)
    // แล้ว busy เพิ่งปิดกลางคัน — เศษแบบนั้นจะสั้นและไม่มีคีย์เวิร์ด เลยถูกเดาผิดเป็น ITC
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
      // ยิงเฉพาะโค้ดที่เริ่มนับจากจังหวะว่าง — กันเศษท้ายบาร์โค้ดที่คร่อมช่วง busy หลุดมา
      if (clean && code.length >= 2) fireRef.current(code)
    }

    function onKeydown(e) {
      // ระหว่าง flow กำลังทำงาน (popup เปิด): ล้าง buffer ทิ้ง + อัปเดตเวลาไว้ตลอด
      // เพื่อว่าถ้า flow ปิดกลางคันตอนบาร์โค้ดถัดไปยังยิงไม่จบ ตัวอักษรที่เหลือจะยัง
      // ต่อเนื่อง (gap < 50ms) ทำให้ startedClean = false -> ไม่ถูกยิงเป็นโค้ดใหม่
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
        // เว้นเกิน 50ms = เริ่มสแกนชุดใหม่จากจังหวะว่าง -> เริ่มนับใหม่แบบ "สะอาด"
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

    // ⭐ Fallback สำหรับสแกนเนอร์บางรุ่น (เช่น WinMax P307) ที่ "ไม่ได้พิมพ์ทีละตัวอักษร"
    // แบบคีย์บอร์ดจริง แต่แทรกข้อความที่อ่านได้ทั้งก้อนเข้าไปในช่องที่โฟกัสอยู่ในทีเดียว
    // (เช่นเครื่องสแกนที่เป็น Android/PDA ยิงผ่าน IME/"paste" แทนการจำลองปุ่มกด)
    // — กรณีนี้ onKeydown ด้านบนจะไม่เห็นอะไรเลย เพราะไม่มี keydown ทีละตัวเกิดขึ้น
    // จึงต้องดัก event 'input' เพิ่ม: ถ้ามีการแทรกข้อความยาว > 1 ตัวอักษรในจังหวะเดียว
    // (คนพิมพ์เองจะได้ทีละตัวอักษรต่อ event เสมอ) ให้ถือว่าเป็นการยิงบาร์โค้ด
    function onGlobalInput(e) {
      if (busyRef.current) return
      const inserted = typeof e.data === 'string' ? e.data : ''
      const code = inserted.trim()
      if (code.length < 2) return // ตัวอักษรเดียว/ไม่มีค่า -> น่าจะเป็นคนพิมพ์เอง ปล่อยผ่าน

      // เอาข้อความที่เพิ่งแทรกออกจากช่องเดิม กันไม่ให้ไปปนกับค่าที่มีอยู่ก่อน
      // (เช่น ช่องค้นหาประวัติการสแกน) เพราะช่องนั้นไม่ได้ตั้งใจรับบาร์โค้ดนี้
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

  // ── ตารางเทียบ: บัญชีใบอนุญาตนำเข้าทั้งหมด ────────────────────────────────
  const licenseRows = useMemo(() => {
    let list = licenseItems
    if (licenseTab === 'pending') list = list.filter((r) => r.ConfirmStatus !== 'CONFIRMED')
    if (licenseTab === 'confirmed') list = list.filter((r) => r.ConfirmStatus === 'CONFIRMED')
    if (licenseModel !== 'all') list = list.filter((r) => (r.Model || '') === licenseModel)
    return list
  }, [licenseItems, licenseTab, licenseModel])

  // แบ่งหน้าตารางเทียบ
  const licenseTotalPages = Math.max(1, Math.ceil(licenseRows.length / licensePageSize))
  const licensePaged = licenseRows.slice(
    (licensePage - 1) * licensePageSize,
    licensePage * licensePageSize
  )
  function goToLicensePage(p) {
    setLicensePage(Math.min(Math.max(1, p), licenseTotalPages))
  }

  // เมื่อสแกนโดนแถวไหน ให้เด้งไปหน้าที่มีแถวนั้น จะได้เห็นไฮไลต์แม้อยู่คนละหน้า
  useEffect(() => {
    if (!highlightId) return
    const idx = licenseRows.findIndex((r) => r.ID === highlightId)
    if (idx >= 0) setLicensePage(Math.floor(idx / licensePageSize) + 1)
  }, [highlightId, licenseRows, licensePageSize])

  // รายชื่อ แบบ/รุ่น ที่มีอยู่จริงในบัญชี (ไว้ทำตัวเลือกใน dropdown กรอง)
  const licenseModelOptions = useMemo(() => {
    const set = new Set()
    licenseItems.forEach((r) => {
      if (r.Model) set.add(r.Model)
    })
    return Array.from(set).sort()
  }, [licenseItems])

  const licenseCounts = useMemo(() => {
    return {
      total: licenseItems.length,
      confirmed: licenseItems.filter((r) => r.ConfirmStatus === 'CONFIRMED').length,
      pending: licenseItems.filter((r) => r.ConfirmStatus !== 'CONFIRMED').length,
    }
  }, [licenseItems])

  // ── ประวัติการสแกน ───────────────────────────────────────────────────────
  const filtered = useMemo(() => {
    const now = new Date()
    let list = rows

    if (dateTab !== 'all') {
      list = list.filter((r) => {
        const diffDays = (now - new Date(r.CheckedDatetime)) / (1000 * 60 * 60 * 24)
        if (dateTab === 'day') return diffDays <= 1
        if (dateTab === 'week') return diffDays <= 7
        if (dateTab === 'month') return diffDays <= 31
        return true
      })
    }

    if (matchFilter !== 'all') {
      list = list.filter((r) => r.MatchStatus === matchFilter)
    }

    const term = search.trim().toLowerCase()
    if (term) {
      list = list.filter(
        (r) =>
          (r.Tag || '').toLowerCase().includes(term) ||
          (r.PN || '').toLowerCase().includes(term) ||
          (r.SN || '').toLowerCase().includes(term) ||
          (r.MachineNo || '').toLowerCase().includes(term) ||
          (r.CheckedBy || '').toLowerCase().includes(term)
      )
    }

    return list
  }, [rows, dateTab, search, matchFilter])

  const mismatchCount = useMemo(
    () => rows.filter((r) => r.PartType === 'ITC' && r.MatchStatus && r.MatchStatus !== 'MATCH').length,
    [rows]
  )

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  return (
    <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Part Confirmation</h2>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="pc-barcode-grid">
        {BARCODE_CARDS.map((card) => (
          <div
            className={'pc-barcode-card pc-card-' + card.partType.toLowerCase()}
            key={card.partType}
            role="button"
            tabIndex={0}
            title={`เริ่มสแกน ${card.title}`}
            onClick={() => runScanFlow(card.partType)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                runScanFlow(card.partType)
              }
            }}
          >
            <span className="pc-barcode-kind">{card.kind}</span>
            <div className="pc-barcode-title">{card.title}</div>
            <div className="pc-barcode-box">
              <img className="pc-barcode-img" src={card.img} alt={`บาร์โค้ด ${card.caption}`} />
            </div>
          </div>
        ))}
      </div>

      {/* ── ผลสแกนล่าสุด ────────────────────────────────────────────────── */}
      {lastScan && (
        <div
          className={
            'il-result-bar' +
            (lastScan.matchStatus === 'MATCH'
              ? ' il-result-ok'
              : lastScan.matchStatus === 'NOT_REQUIRED'
              ? ''
              : ' il-result-bad')
          }
        >
          <div>
            <strong>{tagLabel(lastScan.partType)}</strong>
            {lastScan.pn ? (
              <>
                {' '}
                · P/N <span className="il-mono">{lastScan.pn}</span>
              </>
            ) : null}{' '}
            · S/N <span className="il-mono">{lastScan.sn}</span>
            {lastScan.machineNo ? (
              <>
                {' '}
                · หมายเลขเครื่อง (IT Controller){' '}
                <span className="il-mono">{lastScan.machineNo}</span>
              </>
            ) : null}
            {lastScan.productionNo ? (
              <>
                {' '}
                · หมายเลขการผลิต <span className="il-mono">{lastScan.productionNo}</span>
              </>
            ) : null}
          </div>
          <div className="il-result-msg">
            {matchBadge(lastScan.matchStatus)} {lastScan.message}
          </div>
        </div>
      )}

      {/* ── ตารางเทียบกับบัญชีใบอนุญาต ─────────────────────────────────── */}
      {!loading && licenseItems.length === 0 && (
        <p className="wh-subtitle">
          ยังไม่มีบัญชีใบอนุญาตนำเข้าในระบบ — ไปที่เมนู <strong>Import License</strong>{' '}
          เพื่ออัปโหลดไฟล์ Excel ก่อน แล้วค่อยกลับมาสแกน
        </p>
      )}
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            เทียบกับบัญชีใบอนุญาตนำเข้า ({licenseCounts.confirmed}/{licenseCounts.total})
          </h2>
        </div>
        <div className="vr-tabs">
          {[
            { key: 'all', label: `ทั้งหมด (${licenseCounts.total})` },
            { key: 'pending', label: `รอสแกน (${licenseCounts.pending})` },
            { key: 'confirmed', label: `ยืนยันแล้ว (${licenseCounts.confirmed})` },
          ].map((tab) => (
            <button
              key={tab.key}
              className={'vr-tab' + (licenseTab === tab.key ? ' vr-tab-active' : '')}
              onClick={() => setLicenseTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="tsf-history-toolbar">
        <div className="tsf-history-pagesize">
          <div className="wh-pagesize-select">
            <SelectField
              value={licensePageSize}
              onChange={setLicensePageSize}
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
        <div className="wh-filter-field">
          <span className="wh-filter-label">แบบ/รุ่น</span>
          <SelectField
            value={licenseModel}
            onChange={setLicenseModel}
            options={[
              { value: 'all', label: 'ทั้งหมด' },
              ...licenseModelOptions.map((m) => ({ value: m, label: m })),
            ]}
          />
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              <th>แบบ/รุ่น</th>
              <th>ใบอนุญาตนำเข้า</th>
              <th>อินวอยซ์</th>
              <th>หมายเลขเครื่อง</th>
              <th>หมายเลขการผลิต</th>
              <th>หมายเหตุ</th>
              <th>ส่งออกไปประเทศ</th>
              <th>สถานะ</th>
              <th>TAG ที่สแกนคู่</th>
              <th>ยืนยันเมื่อ</th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={11} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              licensePaged.map((r, idx) => (
                <tr key={r.ID} className={highlightId === r.ID ? 'il-row-hit' : ''}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {(licensePage - 1) * licensePageSize + idx + 1}
                  </td>
                  <td data-label="แบบ/รุ่น">{r.Model || '—'}</td>
                  <td data-label="ใบอนุญาตนำเข้า">{r.LicenseNo || '—'}</td>
                  <td data-label="อินวอยซ์">{r.InvoiceNo || '—'}</td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    <strong>{r.MachineNo}</strong>
                  </td>
                  <td className="il-mono" data-label="หมายเลขการผลิต">
                    {r.ProductionNo || '—'}
                  </td>
                  <td data-label="หมายเหตุ">{r.Remark || '—'}</td>
                  <td data-label="ส่งออกไปประเทศ">{r.ExportCountry || '—'}</td>
                  <td data-label="สถานะ">
                    {r.ConfirmStatus === 'CONFIRMED' ? (
                      <span className="il-badge il-badge-ok">
                        <CheckIcon className="inline size-3.5 align-text-bottom" /> ตรงกัน
                      </span>
                    ) : (
                      <span className="il-badge il-badge-pending">
                        <ClockIcon className="inline size-3.5 align-text-bottom" /> รอสแกน
                      </span>
                    )}
                  </td>
                  <td data-label="TAG ที่สแกนคู่">{r.ConfirmedTag || '—'}</td>
                  <td data-label="ยืนยันเมื่อ">
                    {r.ConfirmedDatetime ? new Date(r.ConfirmedDatetime).toLocaleString('th-TH') : '—'}
                  </td>
                </tr>
              ))}
            {!loading && licenseRows.length === 0 && (
              <tr>
                <td colSpan={11} className="wh-empty-cell">
                  ไม่มีรายการในมุมมองนี้
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {!loading && licenseRows.length > 0 && (
        <div className="tsf-pagination">
          <span className="wh-subtitle" style={{ fontSize: 13 }}>
            Showing {(licensePage - 1) * licensePageSize + 1} to{' '}
            {Math.min(licensePage * licensePageSize, licenseRows.length)} of {licenseRows.length}{' '}
            entries
          </span>
          <div className="tsf-pagination-buttons">
            <button
              className="wh-modal-cancel"
              onClick={() => goToLicensePage(1)}
              disabled={licensePage === 1}
            >
              <ChevronDoubleLeftIcon className="size-4" />
            </button>
            <button
              className="wh-modal-cancel"
              onClick={() => goToLicensePage(licensePage - 1)}
              disabled={licensePage === 1}
            >
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {licensePage} / {licenseTotalPages}
            </span>
            <button
              className="wh-modal-cancel"
              onClick={() => goToLicensePage(licensePage + 1)}
              disabled={licensePage === licenseTotalPages}
            >
              <ChevronRightIcon className="size-4" />
            </button>
            <button
              className="wh-modal-cancel"
              onClick={() => goToLicensePage(licenseTotalPages)}
              disabled={licensePage === licenseTotalPages}
            >
              <ChevronDoubleRightIcon className="size-4" />
            </button>
          </div>
        </div>
      )}

      {/* ── ประวัติการสแกน ─────────────────────────────────────────────── */}
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{ fontSize: 19 }}>
            ประวัติการสแกน ({filtered.length})
          </h2>
          {mismatchCount > 0 && (
            <p className="wh-subtitle" style={{ color: '#b42318', fontWeight: 600 }}>
              มี {mismatchCount} รายการที่สแกนแล้วไม่ตรงกับบัญชีใบอนุญาต
            </p>
          )}
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
        <div className="wh-history-filters">
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
          <div className="wh-filter-field">
            <span className="wh-filter-label">ผลเทียบใบอนุญาต</span>
            <SelectField
              value={matchFilter}
              onChange={setMatchFilter}
              options={[
                { value: 'all', label: 'ทั้งหมด' },
                { value: 'MATCH', label: 'ตรงกับใบอนุญาต' },
                { value: 'NOT_FOUND', label: 'ไม่พบในใบอนุญาต' },
              ]}
            />
          </div>
        </div>
        <input
          className="wh-search"
          type="text"
          placeholder="ค้นหา Tag / P/N / หมายเลขเครื่อง / ผู้ตรวจสอบ"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Part</th>
              <th>P/N</th>
              <th>S/N</th>
              <th>หมายเลขเครื่อง (IT Controller)</th>
              <th>ผลเทียบใบอนุญาต</th>
              <th>Checked By</th>
              <th>วันที่</th>
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
              paged.map((r) => (
                <tr key={r.ID}>
                  <td className="wh-cell-head" data-label="Part">
                    <strong>{tagLabel(r.PartType)}</strong>
                  </td>
                  <td data-label="P/N">{r.PN || '—'}</td>
                  <td className="il-mono" data-label="S/N">
                    {r.SN || '—'}
                  </td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง (IT Controller)">
                    {r.MachineNo || '—'}
                  </td>
                  <td data-label="ผลเทียบใบอนุญาต">{matchBadge(r.MatchStatus)}</td>
                  <td data-label="Checked By">{r.CheckedBy}</td>
                  <td data-label="วันที่">{new Date(r.CheckedDatetime).toLocaleString('th-TH')}</td>
                  <td className="wh-cell-action">
                    <button className="tsf-action-btn" onClick={() => setDetailRow(r)}>
                      รายละเอียด
                    </button>
                    {r.MatchStatus === 'NOT_FOUND' && (
                      <button
                        className="tsf-action-btn tsf-action-btn-danger"
                        onClick={() => handleDeleteCheck(r)}
                      >
                        ลบ
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            {!loading && paged.length === 0 && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  ยังไม่มีรายการตรวจสอบ
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

      {detailRow && (
        <div className="wh-modal-overlay" onClick={() => setDetailRow(null)}>
          <div className="wh-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">รายละเอียดการตรวจสอบ</h3>
            <p className="wh-modal-line">
              ชนิดพาร์ท: <strong>{tagLabel(detailRow.PartType)}</strong>
            </p>
            {detailRow.Tag ? (
              <p className="wh-modal-line">Machine TAG: {detailRow.Tag}</p>
            ) : null}
            <p className="wh-modal-line">P/N: {detailRow.PN || '—'}</p>
            <p className="wh-modal-line">S/N: {detailRow.SN || '—'}</p>
            <p className="wh-modal-line">
              หมายเลขเครื่อง (IT Controller): {detailRow.MachineNo || '—'}
            </p>
            <p className="wh-modal-line">หมายเลขการผลิต (IMEI): {detailRow.ProductionNo || '—'}</p>
            <p className="wh-modal-line">ใบอนุญาตนำเข้า: {detailRow.LicenseNo || '—'}</p>
            <p className="wh-modal-line">อินวอยซ์: {detailRow.InvoiceNo || '—'}</p>
            <p className="wh-modal-line">
              ผลเทียบ: {matchBadge(detailRow.MatchStatus)} {detailRow.MatchMessage || ''}
            </p>
            <p className="wh-modal-line">ตรวจสอบโดย: {detailRow.CheckedBy}</p>
            <p className="wh-modal-line">
              เวลา: {new Date(detailRow.CheckedDatetime).toLocaleString('th-TH')}
            </p>
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