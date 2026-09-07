import { apiFetch } from './client.js';

// ค่าตั้งปัจจุบัน + สถานะรอบส่งของสัปดาห์นี้
export function getWeeklyAlertStatus() {
  return apiFetch('/weekly-alert/status');
}

// ตัวอย่างอีเมลของสัปดาห์ปัจจุบัน (ไม่ส่งจริง)
export function getWeeklyAlertPreview() {
  return apiFetch('/weekly-alert/preview');
}

// ส่งอีเมลเดี๋ยวนี้
// to = เว้นว่างไว้เพื่อส่งตามรายชื่อใน .env หรือใส่อีเมลเองเพื่อส่งทดสอบ
export function sendWeeklyAlertNow(to = '') {
  return apiFetch('/weekly-alert/send', {
    method: 'POST',
    body: JSON.stringify({ to })
  });
}
