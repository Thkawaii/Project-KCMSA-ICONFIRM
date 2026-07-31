import { apiFetch, API_BASE_URL, getToken } from './client.js'

// ── ตารางอ้างอิงเพิ่มเติมของ Warehouse: ชีต MC (สต๊อกเครื่อง) + Inv (อินวอยซ์) ──
// อัปโหลดจากไฟล์ Excel เล่มเดียวกับบัญชีใบอนุญาต ระบบเลือกชีตให้อัตโนมัติตามชื่อ
// (ชีต "MC" / "Inv") ถ้าไฟล์มีชีตเดียวก็ใช้ชีตแรกได้เลย

// multipart helper — ใช้ fetch ตรง (apiFetch ใส่ Content-Type: application/json
// ให้อัตโนมัติ ซึ่งจะทำให้ boundary หายและ backend parse ไฟล์ไม่ได้)
async function uploadFile(path, file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}${path}`, {
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

// ── MC — สต๊อกเครื่อง/ออเดอร์ ──────────────────────────────────────────────
export function getWHMachineStock(q = '') {
  const qs = q ? `?q=${encodeURIComponent(q)}` : ''
  return apiFetch(`/wh-stock/mc${qs}`)
}
export function uploadWHMachineStock(file) {
  return uploadFile('/wh-stock/mc/upload', file)
}
export function deleteWHMachineStock(id) {
  return apiFetch(`/wh-stock/mc/${id}`, { method: 'DELETE' })
}
export function clearWHMachineStock() {
  return apiFetch('/wh-stock/mc', { method: 'DELETE' })
}

// ── Inv — รายการอินวอยซ์ + ตำแหน่งจัดเก็บ ──────────────────────────────────
export function getWHInvoice(q = '') {
  const qs = q ? `?q=${encodeURIComponent(q)}` : ''
  return apiFetch(`/wh-stock/inv${qs}`)
}
export function uploadWHInvoice(file) {
  return uploadFile('/wh-stock/inv/upload', file)
}
export function deleteWHInvoice(id) {
  return apiFetch(`/wh-stock/inv/${id}`, { method: 'DELETE' })
}
export function clearWHInvoice() {
  return apiFetch('/wh-stock/inv', { method: 'DELETE' })
}
