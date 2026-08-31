import { apiFetch } from './client.js';
export function getQAConfirmedTable() {
  return apiFetch('/qa/confirmed');
}
