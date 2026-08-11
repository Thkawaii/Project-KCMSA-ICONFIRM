import { apiFetch, API_BASE_URL, getToken } from './client.js'

// ─────────────────────────────────────────────────────────────────────────────
// Format Config API — เรียกใช้ endpoint สำหรับ "รองรับการเปลี่ยน format" ตอนรัน
//
//  A) Column Alias : ไฟล์อัปโหลดเปลี่ยนชื่อ/เพิ่มหัวคอลัมน์ → แม็ปไปคอลัมน์มาตรฐาน
//  B) Code Alias   : ค่า P/N / S/N / Machine No. เปลี่ยน format → แม็ปกลับทะเบียนกลาง
// ─────────────────────────────────────────────────────────────────────────────

// ---------- A) Column Alias ----------

// รายการจับคู่หัวคอลัมน์ทั้งหมด (กรองด้วย scope เช่น 'planning' | 'wh1' | 'wh2' | 'engine')
export function getColumnAliases(scope) {
  const qs = scope ? `?scope=${encodeURIComponent(scope)}` : ''
  return apiFetch(`/format-config/column-alias${qs}`)
}

// เพิ่มการจับคู่ 1 รายการ: { scope, source, target, note? }
//   scope  = ชื่อ dataset ของหน้า Upload Data
//   source = หัวคอลัมน์ที่ไฟล์เขียนมาจริง (เช่น 'หมายเลขเครื่อง (ใหม่)')
//   target = ชื่อคอลัมน์มาตรฐานที่จะให้ค่าไหลไปลง (เช่น 'Machine')
export function createColumnAlias({ scope, source, target, note }) {
  return apiFetch('/format-config/column-alias', {
    method: 'POST',
    body: JSON.stringify({ scope, source, target, note: note || '' }),
  })
}

export function deleteColumnAlias(id) {
  return apiFetch(`/format-config/column-alias/${id}`, { method: 'DELETE' })
}

// ---------- B) Code Alias ----------

// รายการจับคู่ค่ารหัสทั้งหมด (กรองด้วย componentType / kind ได้)
export function getCodeAliases({ componentType, kind } = {}) {
  const params = new URLSearchParams()
  if (componentType) params.set('component_type', componentType)
  if (kind) params.set('kind', kind)
  const qs = params.toString()
  return apiFetch(`/format-config/code-alias${qs ? `?${qs}` : ''}`)
}

// เพิ่มการจับคู่ค่ารหัส 1 รายการ:
//   { from_code, to_serial_no, to_part_no?, component_type?, kind?, note? }
//   from_code    = ค่าที่หน้างานยิงมา (รูปแบบใหม่ที่ยังไม่มีในทะเบียน)
//   to_serial_no = S/N มาตรฐานในทะเบียนกลางที่ต้องการชี้ไปหา
export function createCodeAlias(payload) {
  return apiFetch('/format-config/code-alias', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function deleteCodeAlias(id) {
  return apiFetch(`/format-config/code-alias/${id}`, { method: 'DELETE' })
}

// อัปโหลด code alias จำนวนมากจากไฟล์ Excel/CSV ทีเดียว
// (หัวคอลัมน์: from_code, to_serial_no, [to_part_no], [component_type], [kind], [note])
export async function uploadCodeAliases(file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/format-config/code-alias/upload`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  })

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(data?.message || `Upload failed (${res.status})`)
  }
  return data
}
