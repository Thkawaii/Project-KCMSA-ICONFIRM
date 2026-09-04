const envBase = import.meta.env.VITE_API_BASE_URL;
export const API_BASE_URL = envBase === undefined || envBase === null ? 'http://localhost:8080' : envBase;
export function getToken() {
  return localStorage.getItem('iconfirm_token');
}
export async function apiFetch(path, options = {}) {
  const token = getToken();
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {})
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });
  let data = null;
  try {
    data = await res.json();
  } catch {}
  if (!res.ok) {
    // แสดงเฉพาะข้อความหลักสั้น ๆ ไม่ต่อท้ายด้วย detail
    // (เช่น กรณีสแกนรหัสรูปแบบเก่า ให้ขึ้นแค่ "รูปแบบเดิมถูกยกเลิกแล้ว")
    // ส่วน detail ยังเก็บไว้ใน error.detail เผื่อหน้าไหนอยากเอาไปใช้เอง
    const base = data?.message || `Request failed (${res.status})`;
    const error = new Error(base);
    error.detail = data?.detail || '';
    error.status = res.status;
    error.data = data;
    throw error;
  }
  return data;
}