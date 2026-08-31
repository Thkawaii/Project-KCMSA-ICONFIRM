import { apiFetch, API_BASE_URL, getToken } from './client.js';
export function getMasterData({
  componentType,
  code
} = {}) {
  const params = new URLSearchParams();
  if (componentType) params.set('component_type', componentType);
  if (code) params.set('code', code);
  const qs = params.toString();
  return apiFetch(`/master-data${qs ? `?${qs}` : ''}`);
}
export function deleteMasterData(id) {
  return apiFetch(`/master-data/${id}`, {
    method: 'DELETE'
  });
}
export function getMasterDataSummary({
  componentType
} = {}) {
  const params = new URLSearchParams();
  if (componentType) params.set('component_type', componentType);
  const qs = params.toString();
  return apiFetch(`/master-data/summary${qs ? `?${qs}` : ''}`);
}
export function clearMasterData({
  componentType,
  all = false
} = {}) {
  const params = new URLSearchParams();
  if (all) params.set('all', 'true');else if (componentType) params.set('component_type', componentType);
  return apiFetch(`/master-data?${params.toString()}`, {
    method: 'DELETE'
  });
}
export async function previewMasterDataChanges(file, componentType) {
  const token = getToken();
  const formData = new FormData();
  formData.append('file', file);
  if (componentType) formData.append('component_type', componentType);
  const res = await fetch(`${API_BASE_URL}/master-data/preview`, {
    method: 'POST',
    headers: token ? {
      Authorization: `Bearer ${token}`
    } : {},
    body: formData
  });
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(data?.message || `Preview failed (${res.status})`);
  }
  return data;
}
export function updateMasterData(id, patch, {
  force = false
} = {}) {
  const qs = force ? '?force=true' : '';
  return apiFetch(`/master-data/${id}${qs}`, {
    method: 'PATCH',
    body: JSON.stringify(patch)
  });
}
export async function uploadMasterData(file, componentType) {
  const token = getToken();
  const formData = new FormData();
  formData.append('file', file);
  if (componentType) formData.append('component_type', componentType);
  const res = await fetch(`${API_BASE_URL}/master-data/upload`, {
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
