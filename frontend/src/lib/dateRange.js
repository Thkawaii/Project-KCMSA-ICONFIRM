// ─────────────────────────────────────────────────────────────────────────────
// ตัวช่วยกรองช่วงเวลา "ทั้งหมด / รายวัน / รายสัปดาห์ / รายเดือน" ให้ใช้ร่วมกันหลายหน้า
// (หน้า WH Part Confirmation + หน้า MFG Matching Assembly) เพื่อให้พฤติกรรมตรงกันเป๊ะ
//
// สำคัญ: ใช้ "ปฏิทินจริง" ไม่ใช่หน้าต่างเลื่อน (rolling window) —
//   • รายวัน   = วันนี้ (ตั้งแต่เที่ยงคืนของวันนี้)
//   • รายสัปดาห์ = สัปดาห์นี้ เริ่มวันจันทร์ (ISO) ถึงก่อนจันทร์หน้า
//   • รายเดือน = เดือนนี้ (เดือน/ปี เดียวกับวันนี้)
//
// ของเดิมฝั่ง WH ใช้ diffDays <= 1/7/31 (นับถอยหลังเป็นชั่วโมง) ทำให้ "รายวัน" กินของ
// เมื่อวานบ่าย ๆ มาด้วย ไม่ตรงกับความหมาย "วันนี้" — เปลี่ยนมาใช้ตัวนี้ให้ถูกต้อง
// ─────────────────────────────────────────────────────────────────────────────

// startOfDay คืน Date ที่เวลา 00:00:00.000 ของวันเดียวกับ ref (อิงเวลาเครื่องผู้ใช้)
function startOfDay(ref) {
  return new Date(ref.getFullYear(), ref.getMonth(), ref.getDate())
}

// startOfWeek คืนเที่ยงคืนของ "วันจันทร์" ในสัปดาห์เดียวกับ ref (ISO: จันทร์เป็นวันแรก)
function startOfWeek(ref) {
  const d = startOfDay(ref)
  const dow = d.getDay() // 0=อาทิตย์ .. 6=เสาร์
  const backToMonday = (dow + 6) % 7 // จำนวนวันที่ต้องถอยกลับไปถึงวันจันทร์
  d.setDate(d.getDate() - backToMonday)
  return d
}

// inDateTab ตรวจว่า value (วันที่/เวลา — string, number หรือ Date) อยู่ในช่วง tab หรือไม่
//
//   tab: 'all' | 'day' | 'week' | 'month'   (ค่าอื่น/ว่าง = ไม่กรอง = true)
//
// วันที่อ่านไม่ได้ (Invalid Date) จะถือว่า "ไม่อยู่ในช่วงเฉพาะเจาะจงใด ๆ" คืน false
// เมื่อเลือก day/week/month (แต่จะยังโชว์ตอนเลือก 'ทั้งหมด') — กันแถวเสียมากลบทั้งตาราง
export function inDateTab(value, tab, now = new Date()) {
  if (!tab || tab === 'all') return true

  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return false

  if (tab === 'day') {
    return (
      d.getFullYear() === now.getFullYear() &&
      d.getMonth() === now.getMonth() &&
      d.getDate() === now.getDate()
    )
  }

  if (tab === 'month') {
    return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
  }

  if (tab === 'week') {
    const start = startOfWeek(now)
    const end = new Date(start)
    end.setDate(start.getDate() + 7) // จันทร์หน้า 00:00
    return d >= start && d < end
  }

  return true
}

// ตัวเลือกแท็บช่วงเวลามาตรฐาน — ใช้ให้เหมือนกันทั้ง WH และ MFG
export const DATE_TAB_OPTIONS = [
  { key: 'all', label: 'ทั้งหมด' },
  { key: 'day', label: 'รายวัน' },
  { key: 'week', label: 'รายสัปดาห์' },
  { key: 'month', label: 'รายเดือน' },
]

// ─────────────────────────────────────────────────────────────────────────────
// ตัวช่วยช่วงเวลาแบบ "เลือกได้" (มี anchor) — รองรับ รายวัน/รายสัปดาห์/รายเดือน/รายปี
//
// ต่างจาก inDateTab (ที่ยึด "วันนี้" เสมอ) — ชุดนี้รับ "วันอ้างอิง" (anchor) เข้ามา
// เพื่อให้ผู้ใช้เลือกได้ว่าจะดู/Export ของ วัน/สัปดาห์/เดือน/ปี ไหน เช่น
//   • รายเดือน + anchor = 15 ก.ค. 2569  → ทั้งเดือน ก.ค. 2569
//   • รายปี   + anchor = วันใดก็ได้ปี 2569 → ทั้งปี 2569
//
// ใช้ร่วมกันหน้า LOG (Export ใบอนุญาตส่งออกแยกประเทศ) และหน้า QA (Check Sheet)
// ─────────────────────────────────────────────────────────────────────────────

const THAI_MONTHS_FULL = [
  'มกราคม', 'กุมภาพันธ์', 'มีนาคม', 'เมษายน', 'พฤษภาคม', 'มิถุนายน',
  'กรกฎาคม', 'สิงหาคม', 'กันยายน', 'ตุลาคม', 'พฤศจิกายน', 'ธันวาคม',
]
const THAI_MONTHS_SHORT = [
  'ม.ค.', 'ก.พ.', 'มี.ค.', 'เม.ย.', 'พ.ค.', 'มิ.ย.',
  'ก.ค.', 'ส.ค.', 'ก.ย.', 'ต.ค.', 'พ.ย.', 'ธ.ค.',
]
const pad2 = (n) => String(n).padStart(2, '0')
const beYear = (d) => d.getFullYear() + 543 // ค.ศ. → พ.ศ.

// startOfMonth / startOfYear — คู่กับ startOfDay / startOfWeek ด้านบน
function startOfMonth(ref) {
  return new Date(ref.getFullYear(), ref.getMonth(), 1)
}
function startOfYear(ref) {
  return new Date(ref.getFullYear(), 0, 1)
}

// แปลงค่า anchor เป็น Date "ตามเวลาเครื่อง" — สตริง 'YYYY-MM-DD' ต้อง parse เป็น local
// (new Date('2569-07-15') จะถูกตีเป็น UTC ทำให้เพี้ยนวันในบางโซนเวลา)
function toLocalDate(v) {
  if (v instanceof Date) return v
  if (typeof v === 'string') {
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(v.trim())
    if (m) return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
  }
  return new Date(v)
}

// resolvePeriodRange — คืนช่วง [start, end) (end แบบเปิด = ก่อนวันแรกของช่วงถัดไป)
//   mode  : 'all' | 'day' | 'week' | 'month' | 'year'
//   anchor: วันอ้างอิง (Date หรือ 'YYYY-MM-DD') — ไม่ใส่ = ใช้วันนี้
// คืน null เมื่อ mode = 'all' หรือ anchor เสีย (หมายถึง "ไม่กรอง")
export function resolvePeriodRange(mode, anchor, now = new Date()) {
  if (!mode || mode === 'all') return null
  const ref = anchor ? toLocalDate(anchor) : now
  if (Number.isNaN(ref.getTime())) return null

  if (mode === 'day') {
    const start = startOfDay(ref)
    const end = new Date(start)
    end.setDate(start.getDate() + 1)
    return { start, end }
  }
  if (mode === 'week') {
    const start = startOfWeek(ref)
    const end = new Date(start)
    end.setDate(start.getDate() + 7)
    return { start, end }
  }
  if (mode === 'month') {
    const start = startOfMonth(ref)
    const end = new Date(ref.getFullYear(), ref.getMonth() + 1, 1)
    return { start, end }
  }
  if (mode === 'year') {
    const start = startOfYear(ref)
    const end = new Date(ref.getFullYear() + 1, 0, 1)
    return { start, end }
  }
  return null
}

// inPeriod — value อยู่ในช่วงที่เลือก (mode + anchor) หรือไม่
// วันที่อ่านไม่ได้จะถือว่า "ไม่อยู่ในช่วง" (คืน false) เมื่อมีการกรองจริง
export function inPeriod(value, mode, anchor, now = new Date()) {
  const range = resolvePeriodRange(mode, anchor, now)
  if (!range) return true // 'all' หรือ anchor เสีย = ไม่กรอง
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return false
  return d >= range.start && d < range.end
}

// periodRangeLabel — ป้ายภาษาไทยของช่วงที่เลือก (โชว์ให้ผู้ใช้เห็นว่าจะ Export ช่วงไหน)
//   day   → "15 กรกฎาคม 2569"
//   week  → "14–20 กรกฎาคม 2569"  (ข้ามเดือน/ปีจะกางให้ครบ)
//   month → "กรกฎาคม 2569"
//   year  → "ปี 2569"
export function periodRangeLabel(mode, anchor, now = new Date()) {
  const range = resolvePeriodRange(mode, anchor, now)
  if (!range) return 'ทั้งหมด'
  const { start, end } = range
  const last = new Date(end)
  last.setDate(last.getDate() - 1) // วันสุดท้ายที่รวมอยู่ (inclusive)

  if (mode === 'day') {
    return `${start.getDate()} ${THAI_MONTHS_FULL[start.getMonth()]} ${beYear(start)}`
  }
  if (mode === 'month') {
    return `${THAI_MONTHS_FULL[start.getMonth()]} ${beYear(start)}`
  }
  if (mode === 'year') {
    return `ปี ${beYear(start)}`
  }
  // week — จัดรูปแบบให้อ่านง่ายตามว่าข้ามเดือน/ปีหรือไม่
  const sameMonth = start.getMonth() === last.getMonth() && start.getFullYear() === last.getFullYear()
  const sameYear = start.getFullYear() === last.getFullYear()
  if (sameMonth) {
    return `${start.getDate()}–${last.getDate()} ${THAI_MONTHS_FULL[start.getMonth()]} ${beYear(start)}`
  }
  if (sameYear) {
    return `${start.getDate()} ${THAI_MONTHS_SHORT[start.getMonth()]} – ${last.getDate()} ${THAI_MONTHS_SHORT[last.getMonth()]} ${beYear(start)}`
  }
  return `${start.getDate()} ${THAI_MONTHS_SHORT[start.getMonth()]} ${beYear(start)} – ${last.getDate()} ${THAI_MONTHS_SHORT[last.getMonth()]} ${beYear(last)}`
}

// periodFileTag — ชิ้นส่วนชื่อไฟล์จากช่วงที่เลือก (ค.ศ. เพื่อเรียงไฟล์ง่าย)
//   day → "รายวัน-2026-07-15", week → "รายสัปดาห์-2026-07-14",
//   month → "รายเดือน-2026-07", year → "รายปี-2026", all → "ทั้งหมด"
export function periodFileTag(mode, anchor, now = new Date()) {
  const range = resolvePeriodRange(mode, anchor, now)
  if (!range) return 'ทั้งหมด'
  const s = range.start
  const ymd = `${s.getFullYear()}-${pad2(s.getMonth() + 1)}-${pad2(s.getDate())}`
  if (mode === 'day') return `รายวัน-${ymd}`
  if (mode === 'week') return `รายสัปดาห์-${ymd}`
  if (mode === 'month') return `รายเดือน-${s.getFullYear()}-${pad2(s.getMonth() + 1)}`
  if (mode === 'year') return `รายปี-${s.getFullYear()}`
  return 'ทั้งหมด'
}

// ตัวเลือกโหมดช่วงเวลาแบบเต็ม (มีรายปี) — ใช้กับ PeriodRangePicker
export const PERIOD_MODE_OPTIONS = [
  { key: 'all', label: 'ทั้งหมด' },
  { key: 'day', label: 'รายวัน' },
  { key: 'week', label: 'รายสัปดาห์' },
  { key: 'month', label: 'รายเดือน' },
  { key: 'year', label: 'รายปี' },
]

// ข้อความช่วยเหนือปฏิทิน anchor — บอกผู้ใช้ว่าจะเลือกวันไหนก็ได้ในช่วงนั้น
export const PERIOD_ANCHOR_HINT = {
  day: 'เลือกวันที่ต้องการ',
  week: 'เลือกวันใดก็ได้ในสัปดาห์ที่ต้องการ',
  month: 'เลือกวันใดก็ได้ในเดือนที่ต้องการ',
  year: 'เลือกวันใดก็ได้ในปีที่ต้องการ',
}

const fmtYMD = (d) => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`

// shiftPeriodAnchor — เลื่อน anchor ไปข้างหน้า/หลัง 1 หน่วยตามโหมด
//   (รายวัน→วัน, รายสัปดาห์→7 วัน, รายเดือน→เดือน, รายปี→ปี)
// คืนค่าเป็นวัน "ต้นช่วง" ใหม่เสมอ เพื่อให้เลื่อนซ้ำ ๆ ไม่เพี้ยน (เช่น 31 → เดือนถัดไปไม่กลายเป็นเลื่อน 2 เดือน)
export function shiftPeriodAnchor(mode, anchor, delta, now = new Date()) {
  const range = resolvePeriodRange(mode, anchor, now)
  const ref = range ? range.start : anchor ? toLocalDate(anchor) : now
  let d
  if (mode === 'day') d = new Date(ref.getFullYear(), ref.getMonth(), ref.getDate() + delta)
  else if (mode === 'week') d = new Date(ref.getFullYear(), ref.getMonth(), ref.getDate() + delta * 7)
  else if (mode === 'month') d = new Date(ref.getFullYear(), ref.getMonth() + delta, 1)
  else if (mode === 'year') d = new Date(ref.getFullYear() + delta, 0, 1)
  else return anchor
  return fmtYMD(d)
}

// periodStepLabel — ป้ายสั้นบนปุ่มลูกศร (‹ ›) ใช้เป็น title/aria
export const PERIOD_STEP_LABEL = {
  day: { prev: 'วันก่อนหน้า', next: 'วันถัดไป' },
  week: { prev: 'สัปดาห์ก่อนหน้า', next: 'สัปดาห์ถัดไป' },
  month: { prev: 'เดือนก่อนหน้า', next: 'เดือนถัดไป' },
  year: { prev: 'ปีก่อนหน้า', next: 'ปีถัดไป' },
}
