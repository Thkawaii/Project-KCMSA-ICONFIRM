import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import WarehousePage from './pages/WarehousePage.jsx'
import WHPartConfirmationPage from './pages/WHPartConfirmationPage.jsx'
import DashboardPage from './pages/DashboardPage.jsx'
import TSFOperatorPage from './pages/TSFOperatorPage.jsx'
import TSFReceivePage from './pages/TSFReceivePage.jsx'
import UploadViewPage from './pages/UploadViewPage.jsx'
import QAMachineList from './pages/qa/QAMachineList.jsx'
import QAMachineDetail from './pages/qa/QAMachineDetail.jsx'
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
        <Route path="/dashboard" element={<DashboardPage />} />

        <Route path="/qa" element={<QAMachineList />} />
        <Route path="/qa/machine/:machineNo" element={<QAMachineDetail />} />

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)