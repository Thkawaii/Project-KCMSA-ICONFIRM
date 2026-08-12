// homeRouteForRole — หน้าแรกของแต่ละ role (แหล่งความจริงเดียว)
//
// ใช้ร่วมกันทั้งตอน login (LoginPage) และตอนเช็คสิทธิ์/เด้งหน้า (main.jsx) เพื่อกัน
// ไม่ให้สองที่นี้ให้ผลต่างกัน (เคยเป็นบั๊ก: WH Manager login แล้วตกไป /dashboard
// เพราะ LoginPage ไม่รู้จัก role WH_MANAGER แต่ main.jsx รู้จัก — ต้อง refresh ถึงเข้าถูก)
export function homeRouteForRole(role) {
  const normalized = (role || '').toUpperCase()

  if (normalized === 'ADMIN') return '/admin'
  if (normalized === 'LOG') return '/warehouse'
  if (normalized === 'WH') return '/warehouse/confirm'
  if (normalized === 'MFG') return '/mfg-assembly'
  if (normalized === 'QA') return '/qa'
  if (normalized === 'TSF') return '/tsf'
  if (normalized === 'UPLOAD') return '/master-data'

  return '/dashboard'
}
