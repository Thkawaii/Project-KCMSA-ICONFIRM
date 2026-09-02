import { apiFetch } from './client.js';

// รายละเอียดเครื่องทุกคัน รวมสด ๆ จาก ALL PART / Planning / WH1 / WH2 / Engine
// (มาแทนตาราง Assembly เดิมที่ถูกยกเลิกไปแล้ว)
export function getMachinePlans() {
  return apiFetch('/machine-plans');
}

const norm = v => String(v || '').trim().toUpperCase();

// แปลงผลลัพธ์เป็นดัชนีสำหรับค้นด้วยคู่ (เครื่อง + IT Controller), เครื่องอย่างเดียว
// หรือเลข IT Controller อย่างเดียว
export function indexMachinePlans(rows) {
  const byPair = {};
  const byMachine = {};
  const byITC = {};

  for (const r of rows || []) {
    const machine = norm(r.machineNo);
    const itc = norm(r.itControllerNo);
    if (machine && itc) byPair[`${machine}|${itc}`] = r;
    if (machine && !byMachine[machine]) byMachine[machine] = r;
    if (itc && !byITC[itc]) byITC[itc] = r;
  }

  return { byPair, byMachine, byITC };
}

// ค้นข้อมูลเครื่องจากหมายเลขเครื่อง และ/หรือ เลข IT Controller
export function lookupMachinePlan(index, machineNo, itControllerNo) {
  if (!index) return null;
  const machine = norm(machineNo);
  const itc = norm(itControllerNo);
  return (
    (machine && itc ? index.byPair[`${machine}|${itc}`] : null) ||
    (machine ? index.byMachine[machine] : null) ||
    (itc ? index.byITC[itc] : null) ||
    // หน้า WH เก็บเลข IT Controller ไว้ในช่องหมายเลขเครื่อง จึงต้องลองค้นสลับด้วย
    (machine ? index.byITC[machine] : null) ||
    null
  );
}
