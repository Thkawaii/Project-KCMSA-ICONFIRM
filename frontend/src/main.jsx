import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
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
import './styles.css'
import './Warehouse.css'
import './AppShell.css'
import './ImportLicense.css'
import './Filedropzone.css'
// theme.css = ชั้น Tailwind + ธีมใหม่ ต้องอยู่ท้ายสุดเสมอ (ทับสไตล์เก่า)
import './theme.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        {/* Warehouse: อัปโหลดบัญชีใบอนุญาตนำเข้า -> สแกนยืนยันเทียบกับบัญชี */}
        <Route path="/warehouse" element={<ImportLicensePage />} />
        <Route path="/warehouse/confirm" element={<WHPartConfirmationPage />} />

        <Route path="/tsf" element={<TSFOperatorPage />} />
        <Route path="/upload" element={<UploadViewPage />} />
        <Route path="/master-data" element={<MasterDataPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />

        <Route path="/qa" element={<QAMachineList />} />
        <Route path="/qa/machine/:machineNo" element={<QAMachineDetail />} />

        {/* หน้ารวมส่วนประกอบ UI ไว้ดู/คัดลอกคลาสตอนทำหน้าใหม่ */}
        <Route path="/ui-kit" element={<UiKitPage />} />

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)
