# Backend Unit Tests

ชุด unit test ครอบคลุมตรรกะหลักของระบบฝั่ง backend (Go). เทสต์ทั้งหมด**ไม่ต้องมี PostgreSQL**
— ใช้ SQLite ในหน่วยความจำ (in-memory) ต่อ 1 เทสต์ จึงรันเร็วและแยกกันสะอาด

## วิธีรัน

```bash
cd backend

# ครั้งแรกครั้งเดียว: เพิ่มไดรเวอร์ SQLite แบบ pure-Go (ไม่ต้องมี C compiler / CGO)
go get github.com/glebarez/sqlite

go test ./...            # รันทั้งหมด
go test ./... -v         # แสดงรายชื่อทุกเทสต์
go test ./controllers/ -run TestScanPartCheck -v   # เจาะเฉพาะกลุ่ม
go test ./... -cover      # ดู coverage
```

## โครงสร้าง

| ไฟล์ | ครอบคลุม |
|------|----------|
| `models/connectivity_test.go` | จำแนกชนิดการเชื่อมต่อ IT Controller (Satellite/4G High/4G Normal) + ค่าคงที่สถานะ |
| `controllers/maintest_test.go` | โครงพื้นฐาน: สร้าง DB in-memory, migrate, helper สร้าง gin.Context + user |
| `controllers/helpers_test.go` | ฟังก์ชัน normalize ค่าจาก Excel/บาร์โค้ด (12 หลัก, ตัด .0, unwrap `="..."`, ฯลฯ) |
| `controllers/import_license_test.go` | `matchImportLicense` ครบทุกสถานะ (MATCH / NOT_FOUND / WRONG_INVOICE / WRONG_PRODNO / DUPLICATE) + CodeAlias |
| `controllers/partcheck_test.go` | WH flow: `resolveITControllerMaster` (ดึงเลขเครื่องจาก P/N+S/N) + `ScanPartCheck` (ตรง/ไม่พบ/ไม่ต้องเทียบ) |
| `controllers/mfg_assembly_test.go` | ดึง IT Controller จากหมายเลขเครื่อง, เทียบกับแผน (`plannedITCForMachine`), สถานะ MATCHED/DUPLICATE, `ScanMFGAssembly` |
| `controllers/assembly_generate_test.go` | ปั๊มตาราง Assembly อัตโนมัติ (Engine ∪ Planning + WH1 + Import License) + upsert ไม่ซ้ำ + Matching Assembly |
| `controllers/master_data_test.go` | รายงานสรุป Connectivity (`GetMasterDataSummary`) + Guard กันแก้กุญแจ match ใน `UpdateMasterData` (บล็อก/force/audit) |
| `controllers/auth_test.go` | Login/JWT, username ใช้ร่วมกันแล้วแยกด้วยรหัสผ่าน, user ปิดใช้งาน |

## หมายเหตุ

- ถ้าองค์กรอยากใช้ `github.com/mattn/go-sqlite3` แทน ให้เปลี่ยน import ใน `maintest_test.go`
  เป็น `"gorm.io/driver/sqlite"` แล้วต้องเปิด `CGO_ENABLED=1` (ต้องมี C compiler)
- เทสต์ปิด foreign-key constraint ตอน migrate (`DisableForeignKeyConstraintWhenMigrating`)
  เพื่อให้ seed ข้อมูลบางส่วนได้โดยไม่ต้องมี record อ้างอิงครบทุกตาราง
