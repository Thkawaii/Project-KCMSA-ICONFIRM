import { apiFetch } from './client.js'

export function getAdminUsers({ role = '', q = '' } = {}) {
  const params = new URLSearchParams()
  if (role) params.set('role', role)
  if (q) params.set('q', q)
  const qs = params.toString()
  return apiFetch(`/admin/users${qs ? `?${qs}` : ''}`)
}

export function createAdminUser({ name, username, password, role_name, status = 'Active' }) {
  return apiFetch('/admin/users', {
    method: 'POST',
    body: JSON.stringify({ name, username, password, role_name, status }),
  })
}

export function updateAdminUser(id, patch) {
  return apiFetch(`/admin/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

export function deleteAdminUser(id) {
  return apiFetch(`/admin/users/${id}`, { method: 'DELETE' })
}
