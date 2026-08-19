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
