import { apiFetch, API_BASE_URL, getToken } from './client.js'

async function postForm(path, formData) {
  const token = getToken()

  const res = await fetch(`${API_BASE_URL}${path}`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  })

  const data = await res.json().catch(() => null)

  if (!res.ok) {
    throw new Error(data?.message || `อัปโหลดไม่สำเร็จ (${res.status})`)
  }

  return data
}

export function getDocuments(params = {}) {
  const qs = new URLSearchParams(params).toString()
  return apiFetch(`/it-controller/documents${qs ? `?${qs}` : ''}`)
}

export function uploadDocument({ file, docType, docNo, invoiceNo = '', poNo = '', remark = '' }) {
  const form = new FormData()
  form.append('file', file)
  form.append('doc_type', docType)
  form.append('doc_no', docNo)
  form.append('invoice_no', invoiceNo)
  form.append('po_no', poNo)
  form.append('remark', remark)
  return postForm('/it-controller/documents', form)
}

export function getImportLicenses() {
  return apiFetch('/it-controller/import-licenses')
}

export function saveImportLicense(payload) {
  return apiFetch('/it-controller/import-licenses', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function uploadSerialList(file) {
  const form = new FormData()
  form.append('file', file)
  return postForm('/it-controller/units/upload', form)
}

export function getUnits(params = {}) {
  const clean = Object.fromEntries(Object.entries(params).filter(([, v]) => v))
  const qs = new URLSearchParams(clean).toString()
  return apiFetch(`/it-controller/units${qs ? `?${qs}` : ''}`)
}

export function receiveUnit(itControllerNo, remark = '') {
  return apiFetch('/it-controller/units/receive', {
    method: 'POST',
    body: JSON.stringify({ it_controller_no: itControllerNo, remark }),
  })
}

export function allocateUnits(itControllerNos, country) {
  return apiFetch('/it-controller/units/allocate', {
    method: 'POST',
    body: JSON.stringify({ it_controller_nos: itControllerNos, country }),
  })
}

export function allocateSplit(splits, { importLicenseNo = '', invoiceNo = '' } = {}) {
  return apiFetch('/it-controller/units/allocate-split', {
    method: 'POST',
    body: JSON.stringify({
      splits,
      import_license_no: importLicenseNo,
      invoice_no: invoiceNo,
    }),
  })
}

export function issueUnit({
  itControllerNo,
  purpose,
  machineNo = '',
  workOrder = '',
  country = '',
  issuedTo = '',
  remark = '',
}) {
  return apiFetch('/it-controller/units/issue', {
    method: 'POST',
    body: JSON.stringify({
      it_controller_no: itControllerNo,
      purpose,
      machine_no: machineNo,
      work_order: workOrder,
      country,
      issued_to: issuedTo,
      remark,
    }),
  })
}

export function exportUnit(itControllerNo, country = '', remark = '') {
  return apiFetch('/it-controller/units/export', {
    method: 'POST',
    body: JSON.stringify({ it_controller_no: itControllerNo, country, remark }),
  })
}

export function getExportLicenses() {
  return apiFetch('/it-controller/export-licenses')
}

export function createExportLicense(payload) {
  return apiFetch('/it-controller/export-licenses', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function downloadExportAttachment(licenseNo) {
  const token = getToken()

  const res = await fetch(
    `${API_BASE_URL}/it-controller/export-licenses/${encodeURIComponent(licenseNo)}/attachment`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} },
  )

  if (!res.ok) {
    throw new Error(`ดาวน์โหลดไม่สำเร็จ (${res.status})`)
  }

  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `EXPORT_ATTACHMENT_${licenseNo}.xlsx`
  a.click()
  URL.revokeObjectURL(url)
}

export function getAlerts() {
  return apiFetch('/it-controller/alerts')
}

export function getWeeklyReport(weeks = 1) {
  return apiFetch(`/it-controller/report/weekly?weeks=${weeks}`)
}

export function traceUnit(key) {
  return apiFetch(`/it-controller/trace/${encodeURIComponent(key)}`)
}