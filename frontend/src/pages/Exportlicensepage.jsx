import AppShell from '../components/AppShell.jsx';
import { WH_NAV_ITEMS, WHExportLicensePanel } from './Importlicensepage.jsx';
export default function ExportLicensePage() {
  return <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Export License</h2>
        </div>
      </div>

      <WHExportLicensePanel />
    </AppShell>;
}
