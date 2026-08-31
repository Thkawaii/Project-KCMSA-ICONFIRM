import { apiFetch, API_BASE_URL, getToken } from './client.js';
export function getMFGAssemblies() {
  return apiFetch('/mfg-assembly');
}
export function scanMFGAssembly({
  machineNo,
  itControllerNo,
  serialNo,
  partType
}) {
  return apiFetch('/mfg-assembly/scan', {
    method: 'POST',
    body: JSON.stringify({
      machineNo,
      serialNo: serialNo || itControllerNo,
      itControllerNo,
      partType
    })
  });
}
export function createMFGAssembly(payload) {
  return apiFetch('/mfg-assembly', {
    method: 'POST',
    body: JSON.stringify(payload)
  });
}
export function updateMFGAssembly(id, payload) {
  return apiFetch(`/mfg-assembly/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload)
  });
}
export function deleteMFGAssembly(id) {
  return apiFetch(`/mfg-assembly/${id}`, {
    method: 'DELETE'
  });
}
export async function uploadMFGAssemblyPhoto(id, fileOrBlob) {
  const token = getToken();
  const formData = new FormData();
  formData.append('file', fileOrBlob, 'mfg-photo.jpg');
  const res = await fetch(`${API_BASE_URL}/mfg-assembly/${id}/photo`, {
    method: 'POST',
    headers: token ? {
      Authorization: `Bearer ${token}`
    } : {},
    body: formData
  });
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(data?.message || `Upload failed (${res.status})`);
  }
  return data;
}
