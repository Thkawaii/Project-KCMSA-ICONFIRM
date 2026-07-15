import { apiFetch } from './client.js'

// เรียก POST /login จริง — backend ตอนนี้ออก JWT token กลับมาด้วยแล้ว
// (แก้ auth.go ฝั่ง backend ให้ sign token ก่อนหน้านี้)
export async function login(username, password) {
  const data = await apiFetch('/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })

  localStorage.setItem('iconfirm_token', data.token)
  localStorage.setItem('iconfirm_role', data.role)
  localStorage.setItem('iconfirm_name', data.name || '')

  return data
}

export function logout() {
  localStorage.removeItem('iconfirm_token')
  localStorage.removeItem('iconfirm_role')
  localStorage.removeItem('iconfirm_name')
}