import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage.jsx'
import WarehousePage from './pages/WarehousePage.jsx'
import DashboardPage from './pages/DashboardPage.jsx'
import TSFOperatorPage from './pages/TSFOperatorPage.jsx'
import QALayout from './pages/qa/QALayout.jsx'
import QADashboard from './pages/qa/QADashboard.jsx'
import QAValidationResults from './pages/qa/QAValidationResults.jsx'
import QAPlaceholder from './pages/qa/QAPlaceholder.jsx'
import './styles.css'
import './warehouse.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/warehouse" element={<WarehousePage />} />
        <Route path="/tsf" element={<TSFOperatorPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />

        <Route path="/qa" element={<QALayout />}>
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<QADashboard />} />
          <Route path="production-orders" element={<QAPlaceholder title="Production Orders" />} />
          <Route path="scan-validate" element={<QAPlaceholder title="Scan & Validate" />} />
          <Route path="validation-results" element={<QAValidationResults />} />
          <Route path="master-data" element={<QAPlaceholder title="Master Data" />} />
          <Route path="traceability" element={<QAPlaceholder title="Traceability" />} />
          <Route path="quality-alerts" element={<QAPlaceholder title="Quality Alerts" />} />
          <Route path="part-comparison" element={<QAPlaceholder title="Part Comparison" />} />
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)