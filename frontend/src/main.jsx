import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import WarehousePage from './pages/WarehousePage.jsx'
import DashboardPage from './pages/DashboardPage.jsx'
import TSFOperatorPage from './pages/TSFOperatorPage.jsx'
import TSFReceivePage from './pages/TSFReceivePage.jsx'
import UploadViewPage from './pages/UploadViewPage.jsx'
import QALayout from './pages/qa/QALayout.jsx'
import QADashboard from './pages/qa/QADashboard.jsx'
import QAValidationResults from './pages/qa/QAValidationResults.jsx'
import './styles.css'
import './Warehouse.css'
import './AppShell.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/warehouse" element={<WarehousePage />} />
        <Route path="/tsf" element={<TSFOperatorPage />} />
        <Route path="/tsf/receive" element={<TSFReceivePage />} />
        <Route path="/upload" element={<UploadViewPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />

        <Route path="/qa" element={<QALayout />}>
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<QADashboard />} />
          <Route path="validation-results" element={<QAValidationResults />} />
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)