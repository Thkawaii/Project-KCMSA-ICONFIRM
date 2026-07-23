import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import WarehousePage from './pages/Warehousepage.jsx'
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

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/warehouse" element={<WarehousePage />} />
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