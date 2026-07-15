// Base URL ของ backend — เปลี่ยนผ่าน env ได้ตอน build/deploy จริง
// (ตั้ง VITE_API_BASE_URL ใน .env ถ้า backend ไม่ได้รันที่ localhost:8080)
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

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