// ─────────────────────────────────────────────────────────────────────────────
// คลังจำ "ป๊อปอัปแจ้งเตือนอายุใบอนุญาตประจำสัปดาห์" — ให้เด้ง "สัปดาห์ละครั้ง"
//
// โจทย์: อยากให้ป๊อปอัปสรุปสถานะอายุใบอนุญาตเด้งขึ้น "แค่ครั้งเดียวต่อสัปดาห์"
//   ผู้ใช้เข้าระบบวันไหนก็ได้ในสัปดาห์นั้น จะเห็นแค่ครั้งแรกครั้งเดียว
//   พอข้ามไปสัปดาห์ใหม่ (ISO week) ก็เด้งใหม่อีกครั้งเอง
//
// วิธีจำ: เก็บ "รหัสสัปดาห์ ISO" (เช่น 2026-W34) ที่เคยแสดงไปแล้วไว้ที่ localStorage
//   ของเครื่องนั้น ไม่ต้องแตะ backend เลย — เป็นเรื่อง "การรับรู้ของผู้ใช้" ล้วน ๆ
//   ISO week เริ่มวันจันทร์ จึงตรงกับความรู้สึก "สัปดาห์เดียวกัน" ของคนทั่วไป
// ─────────────────────────────────────────────────────────────────────────────

const STORE_KEY = 'iconfirm_license_weekly_popup_shown'

// รหัสสัปดาห์แบบ ISO-8601: ปีของสัปดาห์ + เลขสัปดาห์ (จันทร์เป็นวันแรก)
//   คืนค่าเช่น "2026-W34" — ทุกวันจันทร์–อาทิตย์ในสัปดาห์เดียวกันได้ค่าเท่ากัน
export function isoWeekKey(input) {
  const src = input ? new Date(input) : new Date()
  // ตัดเวลาออก ทำงานบน UTC เพื่อไม่ให้ timezone ทำให้เลขสัปดาห์เพี้ยนช่วงเที่ยงคืน
  const d = new Date(Date.UTC(src.getFullYear(), src.getMonth(), src.getDate()))
  const dayNum = d.getUTCDay() || 7 // จันทร์=1 ... อาทิตย์=7
  // เลื่อนไปหา "วันพฤหัสของสัปดาห์นี้" — ตัวกำหนดว่าสัปดาห์นี้เป็นของปีไหน (กติกา ISO)
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  const weekNo = Math.ceil(((d - yearStart) / 86400000 + 1) / 7)
  return `${d.getUTCFullYear()}-W${String(weekNo).padStart(2, '0')}`
}

// อ่านรหัสสัปดาห์ที่เคยแสดงป๊อปอัปไปแล้ว (พังก็คืนค่าว่าง ไม่ให้ล้ม)
function readShownWeek() {
  try {
    return localStorage.getItem(STORE_KEY) || ''
  } catch {
    return ''
  }
}

// สัปดาห์นี้เคยแสดงป๊อปอัปไปแล้วหรือยัง
export function wasShownThisWeek() {
  return readShownWeek() === isoWeekKey()
}

// ปั๊มว่า "แสดงสัปดาห์นี้แล้ว" — เรียกตอนที่ป๊อปอัปโผล่ขึ้นจริง ๆ
export function markShownThisWeek() {
  try {
    localStorage.setItem(STORE_KEY, isoWeekKey())
  } catch {
    // localStorage เต็ม/ปิดอยู่ — ปล่อยผ่าน ไม่ให้ทั้งหน้าล้ม
  }
}

// รีเซ็ต (เผื่อใช้ตอนทดสอบ/ปุ่ม dev) — ลบความจำเพื่อให้ป๊อปอัปเด้งใหม่ทันที
export function resetWeeklyPopup() {
  try {
    localStorage.removeItem(STORE_KEY)
  } catch {
    /* ปล่อยผ่าน */
  }
}
