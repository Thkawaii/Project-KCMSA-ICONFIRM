// ─────────────────────────────────────────────────────────────────────────────
// คลังจำ "การซ่อนแจ้งเตือน" (Dismiss Alert) ของกระดิ่งใบอนุญาตนำเข้า
//
// ทำไมต้องมีไฟล์นี้:
//   ตัวเลขบนกระดิ่งถูกคำนวณสด ๆ จากรายการใบอนุญาตในฐานข้อมูลทุกครั้ง (ไม่เก็บ
//   สถานะ alert ไว้ที่ไหนเลย) เพราะฉะนั้นเดิม "วิธีเดียว" ที่จะทำให้ตัวเลขหาย
//   คือลบใบนั้นทิ้ง ซึ่งไม่ควรทำ — ที่ WH อยากได้คือแค่ "ซ่อน" ออกจากกระดิ่ง
//   โดยที่ข้อมูลใบอนุญาตยังอยู่ครบ
//
//   การซ่อนจึงเป็นเรื่องของ "การรับรู้ของผู้ใช้" ล้วน ๆ เก็บไว้ที่ localStorage
//   ของเครื่องนั้น ไม่ต้องแตะ backend/ฐานข้อมูลเลย
//
// key ของการซ่อน = licenseNo | invoiceNo | status | วันหมดอายุ
//   ผูกกับ "สถานะ + วันหมดอายุ" ตั้งใจให้แจ้งเตือนโผล่กลับมาเองเมื่อสถานการณ์
//   เปลี่ยนไปในทางที่ต้องรีบจัดการมากขึ้น:
//     • ที่ซ่อนตอน "ใกล้หมดอายุ" พอเลยกำหนดจริง (EXPIRING→EXPIRED) key เปลี่ยน
//       -> เด้งกลับมาเตือนใหม่
//     • ต่ออายุ/อัปโหลดใบใหม่ วันหมดอายุเปลี่ยน key เปลี่ยน -> เด้งกลับมาใหม่
//     • ลบใบทิ้งจริง ๆ key นั้นไม่โผล่อีก -> ถูกเก็บกวาดออกจาก localStorage เอง
// ─────────────────────────────────────────────────────────────────────────────

const STORE_KEY = 'iconfirm_license_alert_dismissed'

// วันหมดอายุให้เหลือแค่ส่วนวันที่ (YYYY-MM-DD) เพื่อให้ key คงที่
function expiryDay(item) {
  const raw = item?.ExpiryDate
  if (!raw) return 'no-exp'
  const d = raw instanceof Date ? raw : new Date(raw)
  if (Number.isNaN(d.getTime())) return 'no-exp'
  return d.toISOString().slice(0, 10)
}

// สร้าง key ประจำรายการแจ้งเตือน — ต้องตรงกันทั้งตอนซ่อนและตอนกรองแสดงผล
export function dismissKey(item) {
  const license = item?.LicenseNo || '—'
  const invoice = item?.InvoiceNo || '—'
  const status = item?.Status || '—'
  return `${license}|${invoice}|${status}|${expiryDay(item)}`
}

// อ่านคลังทั้งก้อน -> { [key]: dismissedAtISO }  (พังก็คืนก้อนว่าง ไม่ให้ล้ม)
export function readDismissed() {
  try {
    const raw = localStorage.getItem(STORE_KEY)
    if (!raw) return {}
    const obj = JSON.parse(raw)
    return obj && typeof obj === 'object' ? obj : {}
  } catch {
    return {}
  }
}

function writeDismissed(map) {
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify(map))
  } catch {
    // localStorage เต็ม/ปิดอยู่ — ปล่อยผ่าน ไม่ให้กระดิ่งพังทั้งตัว
  }
}

// ซ่อนรายการหนึ่ง -> คืนคลังชุดใหม่
export function addDismissed(item) {
  const map = readDismissed()
  map[dismissKey(item)] = new Date().toISOString()
  writeDismissed(map)
  return map
}

// เลิกซ่อนรายการหนึ่ง -> คืนคลังชุดใหม่
export function removeDismissed(item) {
  const map = readDismissed()
  delete map[dismissKey(item)]
  writeDismissed(map)
  return map
}

// เลิกซ่อนทั้งหมด
export function clearDismissed() {
  writeDismissed({})
  return {}
}

// เก็บกวาด: เก็บไว้เฉพาะ key ที่ยัง "มีอยู่จริง" ในรายการแจ้งเตือนล่าสุด
//   -> ใบที่ถูกลบ / ต่ออายุแล้ว key เก่าจะหลุดออกไปเอง localStorage ไม่บวม
// คืน map ที่ prune แล้ว (เขียนกลับเฉพาะเมื่อมีการเปลี่ยนแปลงจริง)
export function pruneDismissed(items = []) {
  const map = readDismissed()
  const live = new Set((items || []).map(dismissKey))
  let changed = false
  const next = {}
  for (const [k, v] of Object.entries(map)) {
    if (live.has(k)) next[k] = v
    else changed = true
  }
  if (changed) writeDismissed(next)
  return next
}
