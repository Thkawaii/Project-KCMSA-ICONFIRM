import React from 'react';
import ReactDOM from 'react-dom/client';
import LoginPage from './pages/LoginPage.jsx';
import ImportLicensePage from './pages/Importlicensepage.jsx';
import ExportLicensePage from './pages/Exportlicensepage.jsx';
import WHPartConfirmationPage from './pages/Whpartconfirmationpage.jsx';
import DashboardPage from './pages/Dashboardpage.jsx';
import MasterDataPage from './pages/MasterDataPage.jsx';
import MFGAssemblyPage from './pages/Mfgassemblypage.jsx';
import FormatSettingsPage from './pages/Formatsettingspage.jsx';
import AdminDashboardPage from './pages/AdminDashboardpage.jsx';
import QAMachineList from './pages/qa/Qamachinelist.jsx';
import QAMachineDetail from './pages/qa/Qamachinedetail.jsx';
import UiKitPage from './pages/UiKitPage.jsx';
import LicenseWeeklyPopup from './components/LicenseWeeklyPopup.jsx';
import { NavProvider, useAppNavigate, useAppView } from './lib/nav.jsx';
import { getToken } from './api/client.js';
import { homeRouteForRole } from './lib/roleRoutes.js';
import './styles.css';
import './Warehouse.css';
import './AppShell.css';
import './ImportLicense.css';
import './Filedropzone.css';
import './Selectfield.css';
import './theme.css';
const resolveHomeRoute = homeRouteForRole;
const ROUTE_CONFIG = {
  '/login': {
    component: LoginPage,
    public: true
  },
  '/warehouse': {
    component: ImportLicensePage,
    roles: ['LOG']
  },
  '/warehouse/export-license': {
    component: ExportLicensePage,
    roles: ['LOG']
  },
  '/warehouse/confirm': {
    component: WHPartConfirmationPage,
    roles: ['WH', 'LOG']
  },
  '/mfg-assembly': {
    component: MFGAssemblyPage,
    roles: ['MFG']
  },
  '/admin': {
    component: AdminDashboardPage,
    roles: ['ADMIN']
  },
  '/master-data': {
    component: MasterDataPage,
    roles: ['UPLOAD']
  },
  '/format-settings': {
    component: FormatSettingsPage,
    roles: ['UPLOAD', 'ADMIN']
  },
  '/admin/master-data': {
    component: MasterDataPage,
    roles: ['ADMIN']
  },
  '/dashboard': {
    component: DashboardPage,
    roles: null
  },
  '/qa': {
    component: QAMachineList,
    roles: ['QA']
  },
  '/qa/machine': {
    component: QAMachineDetail,
    roles: ['QA']
  },
  '/ui-kit': {
    component: UiKitPage,
    roles: null
  }
};
function resolveEffectiveView(requestedView) {
  const token = getToken();
  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase();
  if (requestedView === '/login') {
    return token ? resolveHomeRoute(role) : '/login';
  }
  const entry = ROUTE_CONFIG[requestedView];
  if (!entry) {
    return token ? resolveHomeRoute(role) : '/login';
  }
  if (!token) return '/login';
  if (entry.roles && !entry.roles.includes(role)) return resolveHomeRoute(role);
  return requestedView;
}
function AppScreen() {
  const requestedView = useAppView();
  const navigate = useAppNavigate();
  const effectiveView = resolveEffectiveView(requestedView);
  React.useEffect(() => {
    if (effectiveView !== requestedView) {
      navigate(effectiveView);
    }
  }, [effectiveView, requestedView, navigate]);
  const entry = ROUTE_CONFIG[effectiveView] || ROUTE_CONFIG['/login'];
  const Component = entry.component;
  const token = getToken();
  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase();
  const showWeeklyPopup = Boolean(token) && role === 'LOG' && effectiveView !== '/login';
  return <>
      <Component />
      {showWeeklyPopup && <LicenseWeeklyPopup />}
    </>;
}
ReactDOM.createRoot(document.getElementById('root')).render(<React.StrictMode>
    <NavProvider initialView={getToken() ? resolveHomeRoute(localStorage.getItem('iconfirm_role')) : '/login'}>
      <AppScreen />
    </NavProvider>
  </React.StrictMode>);
