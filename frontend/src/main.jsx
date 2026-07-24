import React from 'react'
import ReactDOM from 'react-dom/client'
import LoginPage from './pages/LoginPage.jsx'
import ImportLicensePage from './pages/Importlicensepage.jsx'
import WHPartConfirmationPage from './pages/Whpartconfirmationpage.jsx'
import DashboardPage from './pages/Dashboardpage.jsx'
import TSFOperatorPage from './pages/Tsfoperatorpage.jsx'
import UploadViewPage from './pages/UploadViewpage.jsx'
import MasterDataPage from './pages/MasterDataPage.jsx'
import QAMachineList from './pages/qa/Qamachinelist.jsx'
import QAMachineDetail from './pages/qa/Qamachinedetail.jsx'
import UiKitPage from './pages/UiKitPage.jsx'
import { NavProvider, useAppNavigate, useAppView } from './lib/nav.jsx'
import { getToken } from './api/client.js'
import './styles.css'
import './Warehouse.css'
import './AppShell.css'
import './ImportLicense.css'
import './Filedropzone.css'
// theme.css = ชั้น Tailwind + ธีมใหม่ ต้องอยู่ท้ายสุดเสมอ (ทับสไตล์เก่า)
import './theme.css'

// role_name ที่ backend ส่งมา (ดู seed.go): QA / WH / TSF / UPLOAD
// role อื่น ๆ ที่ไม่ตรงกับ 4 ตัวนี้ (เช่น LOG, Coding) จะถูกส่งไป /dashboard เป็น fallback
function resolveHomeRoute(role) {
  const normalized = (role || '').toUpperCase()

  if (normalized === 'WH') return '/warehouse'
  if (normalized === 'QA') return '/qa'
  if (normalized === 'TSF') return '/tsf'
  if (normalized === 'UPLOAD') return '/master-data'

  return '/dashboard'
}

// ตารางหน้าทั้งหมดของระบบ + role ที่อนุญาต (roles: null = แค่ login ก็เข้าได้ ไม่จำกัด role)
// หมายเหตุ: ค่า key พวกนี้ (เช่น '/warehouse') เป็นแค่ "ชื่อหน้า" ภายใน state เท่านั้น
// ไม่ใช่ path จริงของ browser แล้ว — address bar จะไม่เปลี่ยนตามค่านี้อีกต่อไป
const ROUTE_CONFIG = {
  '/login': { component: LoginPage, public: true },
  '/warehouse': { component: ImportLicensePage, roles: ['WH'] },
  '/warehouse/confirm': { component: WHPartConfirmationPage, roles: ['WH'] },
  '/tsf': { component: TSFOperatorPage, roles: ['TSF'] },
  '/upload': { component: UploadViewPage, roles: ['UPLOAD'] },
  '/master-data': { component: MasterDataPage, roles: ['UPLOAD'] },
  '/dashboard': { component: DashboardPage, roles: null },
  '/qa': { component: QAMachineList, roles: ['QA'] },
  '/qa/machine': { component: QAMachineDetail, roles: ['QA'] },
  '/ui-kit': { component: UiKitPage, roles: null },
}

// เช็คสิทธิ์จริงทุกครั้งที่มีการ "เปลี่ยนหน้า" (ไม่ว่าจะเปลี่ยนโดยคลิกเมนู หรือโดย state ใด ๆ)
// ไม่มี token -> เด้งไป login เสมอ, role ไม่ตรงกับหน้าที่ขอ -> เด้งไป home ของ role ตัวเอง
function resolveEffectiveView(requestedView) {
  const token = getToken()
  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase()

  if (requestedView === '/login') {
    return token ? resolveHomeRoute(role) : '/login'
  }

  const entry = ROUTE_CONFIG[requestedView]
  if (!entry) {
    // ไม่รู้จักหน้านี้ -> ถ้า login แล้วพากลับ home ของตัวเอง ถ้ายัง -> login
    return token ? resolveHomeRoute(role) : '/login'
  }

  if (!token) return '/login'

  if (entry.roles && !entry.roles.includes(role)) return resolveHomeRoute(role)

  return requestedView
}

function AppScreen() {
  const requestedView = useAppView()
  const navigate = useAppNavigate()
  const effectiveView = resolveEffectiveView(requestedView)

  // ถ้าหน้าที่ขอไม่ตรงกับสิทธิ์จริง ให้ sync state view ให้ตรงกับหน้าที่ถูกเด้งไปจริง ๆ
  // (แค่ sync state ภายใน ไม่แตะ URL ใด ๆ ทั้งสิ้น)
  React.useEffect(() => {
    if (effectiveView !== requestedView) {
      navigate(effectiveView)
    }
  }, [effectiveView, requestedView, navigate])

  const entry = ROUTE_CONFIG[effectiveView] || ROUTE_CONFIG['/login']
  const Component = entry.component
  return <Component />
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <NavProvider initialView={getToken() ? resolveHomeRoute(localStorage.getItem('iconfirm_role')) : '/login'}>
      <AppScreen />
    </NavProvider>
  </React.StrictMode>,
)
