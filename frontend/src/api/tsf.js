import { apiFetch } from './client.js'

export function getMasterData() {
  return apiFetch('/master-data')
}

export function getTsfScans() {
  return apiFetch('/tsf')
}

// หมายเหตุ: backend ตอนนี้เก็บแค่ชื่อไฟล์รูป (FileName) ไม่มี endpoint
// สำหรับอัปโหลดตัวไฟล์รูปจริง — ถ้าต้องการเก็บรูปจริงต้องเพิ่ม
// multipart upload endpoint ฝั่ง backend ก่อน
export function createTsfScan(payload) {
  return apiFetch('/tsf', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}