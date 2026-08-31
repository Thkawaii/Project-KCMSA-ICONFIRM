import Swal from 'sweetalert2';
import 'sweetalert2/dist/sweetalert2.min.css';
import { toastSuccess } from './toast.js';
export async function scanStep({
  title,
  html = '',
  placeholder = 'รอรับสัญญาณจากเครื่องสแกน...',
  confirmText = 'ต่อไป',
  cancelText = 'ยกเลิก',
  validate
}) {
  const res = await Swal.fire({
    title,
    html,
    input: 'text',
    inputPlaceholder: placeholder,
    inputAutoFocus: true,
    inputAttributes: {
      autocomplete: 'off',
      autocorrect: 'off',
      autocapitalize: 'off',
      spellcheck: 'false'
    },
    customClass: {
      popup: 'scan-popup',
      input: 'scan-popup-input'
    },
    confirmButtonText: confirmText,
    showCancelButton: true,
    cancelButtonText: cancelText,
    allowEnterKey: false,
    inputValidator: v => {
      const val = (v || '').trim();
      if (!val) return 'ยังไม่มีค่าที่สแกน';
      if (validate) return validate(val) || undefined;
      return undefined;
    },
    didOpen: () => {
      const input = Swal.getInput();
      if (!input) return;
      input.focus();
      let confirmed = false;
      const doConfirm = () => {
        if (confirmed) return;
        if (!input.value.trim()) return;
        confirmed = true;
        Swal.clickConfirm();
      };
      input.addEventListener('keydown', e => {
        if (e.key === 'Enter') {
          e.preventDefault();
          doConfirm();
        }
      });
      input.addEventListener('paste', () => {
        setTimeout(doConfirm, 0);
      });
      let burstStart = 0;
      let burstCount = 0;
      input.addEventListener('keydown', e => {
        if (e.key === 'Enter' || e.key.length !== 1) return;
        const now = Date.now();
        if (now - burstStart > 150) {
          burstStart = now;
          burstCount = 0;
        }
        burstCount++;
        if (burstCount >= 6) {
          setTimeout(doConfirm, 60);
        }
      });
    }
  });
  if (res.isConfirmed && res.value) return res.value.trim();
  return null;
}
export async function scanSelect({
  title,
  html = '',
  options
}) {
  const inputOptions = {};
  options.forEach(o => {
    inputOptions[o.value] = o.label;
  });
  const res = await Swal.fire({
    title,
    html,
    input: 'select',
    inputOptions,
    customClass: {
      popup: 'scan-popup'
    },
    inputPlaceholder: 'เลือกชนิดพาร์ท',
    confirmButtonText: 'ต่อไป',
    showCancelButton: true,
    cancelButtonText: 'ยกเลิก'
  });
  return res.isConfirmed ? res.value : null;
}
export function scanLoading(title = 'กำลังบันทึก...') {
  Swal.fire({
    title,
    allowOutsideClick: false,
    allowEscapeKey: false,
    didOpen: () => Swal.showLoading()
  });
}
export function scanClose() {
  Swal.close();
}
export function scanCloseWait() {
  return new Promise(resolve => {
    const popup = Swal.getPopup();
    if (!popup) {
      resolve();
      return;
    }
    let done = false;
    const finish = () => {
      if (done) return;
      done = true;
      resolve();
    };
    popup.addEventListener('animationend', finish, {
      once: true
    });
    popup.addEventListener('transitionend', finish, {
      once: true
    });
    Swal.close();
    setTimeout(finish, 320);
  });
}
export function scanSuccessToast(title) {
  return toastSuccess(title);
}
function escapeHtml(str) {
  return String(str ?? '').replace(/[&<>"']/g, ch => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  })[ch]);
}
export function scanErrorAlert(text, {
  hint
} = {}) {
  return Swal.fire({
    icon: 'error',
    title: 'เกิดข้อผิดพลาด',
    html: `
      <div class="scan-error-text">${escapeHtml(text)}</div>
      ${hint ? `
        <div class="scan-error-admin-hint">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" class="scan-error-admin-icon">
            <path d="M12 15.5c-4.2 0-7.5 1.7-7.5 4v1.5h15V19.5c0-2.3-3.3-4-7.5-4Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
            <circle cx="12" cy="8" r="3.75" stroke="currentColor" stroke-width="1.6"/>
          </svg>
          <span>${escapeHtml(hint)}</span>
        </div>
      ` : ''}
    `,
    confirmButtonText: 'ตกลง'
  });
}
export async function scanPhotoCapture({
  title,
  html = ''
}) {
  let stream = null;
  const res = await Swal.fire({
    title,
    html: `
      ${html}
      <video id="scan-photo-video" autoplay playsinline muted class="scan-photo-video"></video>
      <canvas id="scan-photo-canvas" style="display:none;"></canvas>
    `,
    customClass: {
      popup: 'scan-popup'
    },
    confirmButtonText: 'ถ่ายรูป',
    showCancelButton: true,
    cancelButtonText: 'ปิด',
    allowOutsideClick: false,
    allowEscapeKey: true,
    didOpen: async () => {
      const video = document.getElementById('scan-photo-video');
      try {
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
          Swal.showValidationMessage('เปิดกล้องไม่ได้: ต้องเข้าเว็บผ่าน https (เช่น https://' + location.host + ') บนมือถือ กล้องถึงจะทำงาน');
          return;
        }
        stream = await navigator.mediaDevices.getUserMedia({
          video: {
            facingMode: 'environment'
          },
          audio: false
        });
        if (video) video.srcObject = stream;
      } catch (err) {
        Swal.showValidationMessage('เปิดกล้องไม่สำเร็จ: ' + (err.message || err));
        Swal.update({
          showCancelButton: true,
          cancelButtonText: 'ปิด'
        });
      }
    },
    preConfirm: () => {
      const video = document.getElementById('scan-photo-video');
      const canvas = document.getElementById('scan-photo-canvas');
      if (!video || !canvas || !video.videoWidth) {
        Swal.showValidationMessage('กล้องยังไม่พร้อม กรุณารอสักครู่แล้วลองใหม่');
        return false;
      }
      const vw = video.videoWidth;
      const vh = video.videoHeight;
      const side = Math.min(vw, vh);
      const sx = (vw - side) / 2;
      const sy = (vh - side) / 2;
      canvas.width = side;
      canvas.height = side;
      canvas.getContext('2d').drawImage(video, sx, sy, side, side, 0, 0, side, side);
      return new Promise(resolve => {
        canvas.toBlob(blob => resolve(blob), 'image/jpeg', 0.9);
      });
    },
    willClose: () => {
      if (stream) stream.getTracks().forEach(t => t.stop());
    }
  });
  if (res.isConfirmed && res.value) return res.value;
  return null;
}
