const STORE_KEY = 'iconfirm_export_license_alert_dismissed';
function expiryDay(item) {
  const raw = item?.ExpiryDate;
  if (!raw) return 'no-exp';
  const d = raw instanceof Date ? raw : new Date(raw);
  if (Number.isNaN(d.getTime())) return 'no-exp';
  return d.toISOString().slice(0, 10);
}
export function exportDismissKey(item) {
  const license = item?.ExceptionLicense || '—';
  const status = item?.Status || '—';
  return `${license}|${status}|${expiryDay(item)}`;
}
export function readExportDismissed() {
  try {
    const raw = localStorage.getItem(STORE_KEY);
    if (!raw) return {};
    const obj = JSON.parse(raw);
    return obj && typeof obj === 'object' ? obj : {};
  } catch {
    return {};
  }
}
function writeExportDismissed(map) {
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify(map));
  } catch {}
}
export function addExportDismissed(item) {
  const map = readExportDismissed();
  map[exportDismissKey(item)] = new Date().toISOString();
  writeExportDismissed(map);
  return map;
}
export function removeExportDismissed(item) {
  const map = readExportDismissed();
  delete map[exportDismissKey(item)];
  writeExportDismissed(map);
  return map;
}
export function clearExportDismissed() {
  writeExportDismissed({});
  return {};
}
export function pruneExportDismissed(items = []) {
  const map = readExportDismissed();
  const live = new Set((items || []).map(exportDismissKey));
  let changed = false;
  const next = {};
  for (const [k, v] of Object.entries(map)) {
    if (live.has(k)) next[k] = v;else changed = true;
  }
  if (changed) writeExportDismissed(next);
  return next;
}