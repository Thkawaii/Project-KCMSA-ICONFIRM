import { apiFetch } from './client.js'

// รายการ MFG Assembly ทั้งหมด (ใหม่สุดอยู่บน)
export function getMFGAssemblies() {
  return apiFetch('/mfg-assembly')
}

// สแกน QR ตอนประกอบเสร็จ — ส่ง Machine No + IT Controller No.
// backend จะคำนวณ Status (OK/UNKNOWN/REUSED/DUPLICATE) + Country ให้ แล้วตอบกลับ
//   { row, status, matched, message }
export function scanMFGAssembly({ machineNo, itControllerNo }) {
  return apiFetch('/mfg-assembly/scan', {
    method: 'POST',
    body: JSON.stringify({ machineNo, itControllerNo }),
  })
}

// เพิ่มแถวเอง (นอกเหนือจากสแกน)
export function createMFGAssembly(payload) {
  return apiFetch('/mfg-assembly', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

// แก้ไข 1 แถว (ทุกฟิลด์แก้ได้)
export function updateMFGAssembly(id, payload) {
  return apiFetch(`/mfg-assembly/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

// ลบ 1 แถว
export function deleteMFGAssembly(id) {
  return apiFetch(`/mfg-assembly/${id}`, { method: 'DELETE' })
}
