import React from 'react'
import ReactDOM from 'react-dom/client'
import LoginPage from './pages/LoginPage.jsx'
import ImportLicensePage from './pages/Importlicensepage.jsx'
import ExportLicensePage from './pages/Exportlicensepage.jsx'
import WHPartConfirmationPage from './pages/Whpartconfirmationpage.jsx'
import DashboardPage from './pages/Dashboardpage.jsx'
import UploadViewPage from './pages/UploadViewpage.jsx'
import MasterDataPage from './pages/MasterDataPage.jsx'
import MFGAssemblyPage from './pages/Mfgassemblypage.jsx'
import FormatSettingsPage from './pages/Formatsettingspage.jsx'
import AdminDashboardPage from './pages/AdminDashboardpage.jsx'
import QAMachineList from './pages/qa/Qamachinelist.jsx'
import QAMachineDetail from './pages/qa/Qamachinedetail.jsx'
import UiKitPage from './pages/UiKitPage.jsx'
import LicenseWeeklyPopup from './components/LicenseWeeklyPopup.jsx'
import { NavProvider, useAppNavigate, useAppView } from './lib/nav.jsx'
import { getToken } from './api/client.js'
import { homeRouteForRole } from './lib/roleRoutes.js'
import './styles.css'
import './Warehouse.css'
import './AppShell.css'
import './ImportLicense.css'
import './Filedropzone.css'
import './Selectfield.css'
// theme.css = ชั้น Tailwind + ธีมใหม่ ต้องอยู่ท้ายสุดเสมอ (ทับสไตล์เก่า)
import './theme.css'

// role_name ที่ backend ส่งมา (ดู seed.go): QA / WH / MFG / LOG / UPLOAD / ADMIN
// role อื่น ๆ ที่ไม่ตรงกับ 4 ตัวนี้ (เช่น LOG, Coding) จะถูกส่งไป /dashboard เป็น fallback
// หน้าแรกของแต่ละ role — ใช้ตัวเดียวกับ LoginPage (ดู lib/roleRoutes.js)
const resolveHomeRoute = homeRouteForRole

// ตารางหน้าทั้งหมดของระบบ + role ที่อนุญาต (roles: null = แค่ login ก็เข้าได้ ไม่จำกัด role)
// หมายเหตุ: ค่า key พวกนี้ (เช่น '/warehouse') เป็นแค่ "ชื่อหน้า" ภายใน state เท่านั้น
// ไม่ใช่ path จริงของ browser แล้ว — address bar จะไม่เปลี่ยนตามค่านี้อีกต่อไป
const ROUTE_CONFIG = {
  '/login': { component: LoginPage, public: true },
  '/warehouse': { component: ImportLicensePage, roles: ['LOG'] },
  '/warehouse/export-license': { component: ExportLicensePage, roles: ['LOG'] },
  '/warehouse/confirm': { component: WHPartConfirmationPage, roles: ['WH', 'LOG'] },
  '/mfg-assembly': { component: MFGAssemblyPage, roles: ['MFG'] },
  '/admin': { component: AdminDashboardPage, roles: ['ADMIN'] },
  '/upload': { component: UploadViewPage, roles: ['UPLOAD'] },
  '/master-data': { component: MasterDataPage, roles: ['UPLOAD'] },
  '/format-settings': { component: FormatSettingsPage, roles: ['UPLOAD', 'ADMIN'] },
  '/admin/master-data': { component: MasterDataPage, roles: ['ADMIN'] },
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

  // ── ป๊อปอัปแจ้งเตือนอายุใบอนุญาตประจำสัปดาห์ ──
  // แสดงเฉพาะผู้ใช้ที่ล็อกอินแล้วและเป็น role LOG (คนเดียวกับที่เห็นกระดิ่งเตือน)
  // วางไว้เป็น sibling ของหน้า -> mount ครั้งเดียวตอนเข้าระบบ และอยู่ข้ามหน้า
  // (ไม่ remount ทุกครั้งที่สลับเมนู) ตัวป๊อปอัปเองคุมให้เด้ง "สัปดาห์ละครั้ง"
  const token = getToken()
  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase()
  const showWeeklyPopup = Boolean(token) && role === 'LOG' && effectiveView !== '/login'

  return (
    <>
      <Component />
      {showWeeklyPopup && <LicenseWeeklyPopup />}
    </>
  )
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <NavProvider initialView={getToken() ? resolveHomeRoute(localStorage.getItem('iconfirm_role')) : '/login'}>
      <AppScreen />
    </NavProvider>
  </React.StrictMode>,
)