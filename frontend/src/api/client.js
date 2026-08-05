// Base URL ของ backend — เปลี่ยนผ่าน env ได้ตอน build/deploy จริง
// กติกาการตั้งค่า VITE_API_BASE_URL ใน .env:
//   - ไม่ตั้งเลย (ไม่มีบรรทัดนี้)  → default 'http://localhost:8080' (ตอน dev บนเครื่อง)
//   - ตั้งเป็นค่าว่าง VITE_API_BASE_URL=  → ยิง API แบบ same-origin (path ตรงๆ เช่น /login)
//       ใช้ตอน build frontend แล้วให้ Go เสิร์ฟจากพอร์ตเดียวกัน — ไม่ต้องผูก IP,
//       ย้ายเครื่อง/เปลี่ยน wifi/ใช้ tunnel ได้โดยไม่ต้องแก้อะไร และไม่ต้องยุ่ง CORS
//   - ตั้งเป็น URL เต็ม เช่น http://192.168.2.62:8080 → ยิงไป host นั้นตรงๆ (โหมดแยกพอร์ต Vite/Go)
const envBase = import.meta.env.VITE_API_BASE_URL
export const API_BASE_URL =
  envBase === undefined || envBase === null ? 'http://localhost:8080' : envBase

export function getToken() {
  return localStorage.getItem('iconfirm_token')
}

export async function apiFetch(path, options = {}) {
  const token = getToken()

  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }

  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  })

  let data = null
  try {
    data = await res.json()
  } catch {
    // response ไม่มี body (เช่น 204) — ปล่อยเป็น null ได้
  }

  if (!res.ok) {
    const message = data?.message || `Request failed (${res.status})`
    const error = new Error(message)
    error.status = res.status
    error.data = data
    throw error
  }

  return data
}