// ─────────────────────────────────────────────────────────────────────────────
// คลังจำ "การซ่อนแจ้งเตือน" (Dismiss Alert) ของกระดิ่งใบอนุญาตส่งออก
//
// เหมือน licenseDismiss.js ทุกประการ ต่างกันแค่:
//   • เก็บคนละช่อง localStorage (ซ่อนฝั่งส่งออกไม่กระทบฝั่งนำเข้า และกลับกัน)
//   • key ผูกกับ Serial Number (ฝั่งส่งออกเป็นรายเครื่อง ไม่ได้จัดกลุ่มรายใบ)
//
// key = serialNumber | status | วันหมดอายุ
//   ผูกกับ "สถานะ + วันหมดอายุ" เพื่อให้แจ้งเตือนเด้งกลับเองเมื่อสถานการณ์เปลี่ยน
//   (EXPIRING→EXPIRED หรือต่ออายุแล้ววันหมดอายุขยับ) — ตรรกะเดียวกับฝั่งนำเข้า
// ─────────────────────────────────────────────────────────────────────────────

const STORE_KEY = 'iconfirm_export_license_alert_dismissed'

// วันหมดอายุให้เหลือแค่ส่วนวันที่ (YYYY-MM-DD) เพื่อให้ key คงที่
function expiryDay(item) {
  const raw = item?.ExpiryDate
  if (!raw) return 'no-exp'
  const d = raw instanceof Date ? raw : new Date(raw)
  if (Number.isNaN(d.getTime())) return 'no-exp'
  return d.toISOString().slice(0, 10)
}

// สร้าง key ประจำรายการแจ้งเตือน — ต้องตรงกันทั้งตอนซ่อนและตอนกรองแสดงผล
export function exportDismissKey(item) {
  const serial = item?.SerialNumber || '—'
  const status = item?.Status || '—'
  return `${serial}|${status}|${expiryDay(item)}`
}

// อ่านคลังทั้งก้อน -> { [key]: dismissedAtISO }  (พังก็คืนก้อนว่าง ไม่ให้ล้ม)
export function readExportDismissed() {
  try {
    const raw = localStorage.getItem(STORE_KEY)
    if (!raw) return {}
    const obj = JSON.parse(raw)
    return obj && typeof obj === 'object' ? obj : {}
  } catch {
    return {}
  }
}

function writeExportDismissed(map) {
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify(map))
  } catch {
    // localStorage เต็ม/ปิดอยู่ — ปล่อยผ่าน ไม่ให้กระดิ่งพังทั้งตัว
  }
}

// ซ่อนรายการหนึ่ง -> คืนคลังชุดใหม่
export function addExportDismissed(item) {
  const map = readExportDismissed()
  map[exportDismissKey(item)] = new Date().toISOString()
  writeExportDismissed(map)
  return map
}

// เลิกซ่อนรายการหนึ่ง -> คืนคลังชุดใหม่
export function removeExportDismissed(item) {
  const map = readExportDismissed()
  delete map[exportDismissKey(item)]
  writeExportDismissed(map)
  return map
}

// เลิกซ่อนทั้งหมด
export function clearExportDismissed() {
  writeExportDismissed({})
  return {}
}

// เก็บกวาด: เก็บไว้เฉพาะ key ที่ยัง "มีอยู่จริง" ในรายการแจ้งเตือนล่าสุด
export function pruneExportDismissed(items = []) {
  const map = readExportDismissed()
  const live = new Set((items || []).map(exportDismissKey))
  let changed = false
  const next = {}
  for (const [k, v] of Object.entries(map)) {
    if (live.has(k)) next[k] = v
    else changed = true
  }
  if (changed) writeExportDismissed(next)
  return next
}
