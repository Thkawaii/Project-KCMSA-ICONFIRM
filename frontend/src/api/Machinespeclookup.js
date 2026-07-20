import { apiFetch } from './client.js'

export function getMachineSpecByMachineNo(machineNo) {
  return apiFetch(`/machine-spec/by-machine/${encodeURIComponent(machineNo)}`)
}