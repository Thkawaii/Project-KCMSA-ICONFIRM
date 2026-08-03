import { apiFetch } from './client.js'

// รายการ Matching Assembly ทั้งหมด (ใหม่สุดอยู่บน)
export function getMatchingAssemblies() {
  return apiFetch('/matching-assembly')
}

// เพิ่มแถวเอง (นอกเหนือจากที่ระบบสร้างอัตโนมัติตอนสแกน IT Controller สำเร็จ)
export function createMatchingAssembly(payload) {
  return apiFetch('/matching-assembly', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

// แก้ไขข้อมูล 1 แถว — ส่งทุกฟิลด์ (backend ตั้งค่าตามที่ส่งมา รวมถึงตั้งเป็นค่าว่างได้)
export function updateMatchingAssembly(id, payload) {
  return apiFetch(`/matching-assembly/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

// ลบ 1 แถว
export function deleteMatchingAssembly(id) {
  return apiFetch(`/matching-assembly/${id}`, { method: 'DELETE' })
}
