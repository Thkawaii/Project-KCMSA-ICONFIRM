import { apiFetch } from './client.js'

// สเปกเครื่องรายคัน ประกอบสดจากไฟล์ Planning / Assembly / Engine ที่อัปโหลดจริง
// (เดิมดึงจากตาราง machine_specs ซึ่งไม่เคยมีข้อมูลเข้าไปเลย)
export function getMachineProfile(machineNo) {
  return apiFetch(`/machine-profile/${encodeURIComponent(machineNo)}`)
}
