import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import ImportLicensePage from './pages/Importlicensepage.jsx'
import WHPartConfirmationPage from './pages/Whpartconfirmationpage.jsx'
import DashboardPage from './pages/Dashboardpage.jsx'
import TSFOperatorPage from './pages/Tsfoperatorpage.jsx'
import TSFReceivePage from './pages/Tsfreceivepage.jsx'
import UploadViewPage from './pages/UploadViewpage.jsx'
import MasterDataPage from './pages/MasterDataPage.jsx'
import QAMachineList from './pages/qa/Qamachinelist.jsx'
import QAMachineDetail from './pages/qa/Qamachinedetail.jsx'
import './styles.css'
import './Warehouse.css'
import './AppShell.css'
import './Itcontroller.css'
import './ImportLicense.css'
import './Filedropzone.css'
import './Selectfield.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        {/* Warehouse: อัปโหลดบัญชีใบอนุญาตนำเข้า -> สแกนยืนยันเทียบกับบัญชี
            (หน้า "จ่ายของ FIFO & S/O" และ "IT Controller (กสทช.)" ถูกถอดออกแล้ว) */}
        <Route path="/warehouse" element={<ImportLicensePage />} />
        <Route path="/warehouse/confirm" element={<WHPartConfirmationPage />} />

        <Route path="/tsf" element={<TSFOperatorPage />} />
        <Route path="/tsf/receive" element={<TSFReceivePage />} />
        <Route path="/upload" element={<UploadViewPage />} />
        <Route path="/master-data" element={<MasterDataPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />

        <Route path="/qa" element={<QAMachineList />} />
        <Route path="/qa/machine/:machineNo" element={<QAMachineDetail />} />

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)
