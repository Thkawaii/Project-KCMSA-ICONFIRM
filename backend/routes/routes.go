package routes

import (
	"iconfirm/controllers"
	"iconfirm/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {

	// Public API
	r.POST("/login", controllers.Login)

	// Protected API
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())

	// Master Data — LOG role uploads it, everyone else needs to read it
	// (TSF/QA compare scans against it), so only require auth, not a role.
	masterData := auth.Group("/master-data")
	{
		masterData.GET("", controllers.GetMasterData)
		masterData.POST("", controllers.CreateMasterData)

		// แก้ไขทะเบียนได้เฉพาะ role UPLOAD เหมือนกับฝั่ง machine-spec
		manage := masterData.Group("")
		manage.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			manage.POST("/upload", controllers.UploadMasterData)
			manage.POST("/preview", controllers.PreviewMasterDataChanges)
			manage.PATCH("/:id", controllers.UpdateMasterData)
			manage.DELETE("/:id", controllers.DeleteMasterData)
			manage.DELETE("", controllers.ClearMasterData)
		}
	}

	// Machine Spec — full machine spec sheets uploaded via Excel by the
	// "UPLOAD" role (uploadview@kobelco.com). Reads are open to any
	// authenticated role since QA/TSF also need to look these up.
	machineSpec := auth.Group("/machine-spec")
	{
		machineSpec.GET("", controllers.GetMachineSpecs)
		machineSpec.GET("/by-machine/:machineNo", controllers.GetMachineSpecByMachineNo)
		machineSpec.GET("/:id", controllers.GetMachineSpecByID)
		machineSpec.GET("/export", controllers.ExportMachineSpecs)

		upload := machineSpec.Group("")
		upload.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			upload.POST("/upload/:type", controllers.UploadMachineSpec)
			upload.DELETE("/:id", controllers.DeleteMachineSpec)
		}
	}

	// ─────────────────────────────────────────────────────────────────────
	// Upload Data — อัปโหลดไฟล์ Excel วางแผน/คลัง 4 ชนิด (Planning / WH1 /
	// WH2 / Engine) เพิ่มเข้ามาต่อจากการอัปโหลด IT Controller / Machine Spec
	// เดิม จัดการโดย role UPLOAD เหมือนกัน อ่านได้ทุก role ที่ login แล้ว
	// ─────────────────────────────────────────────────────────────────────
	uploadData := auth.Group("/upload-data")
	{
		uploadData.GET("", controllers.GetUploadData)
		uploadData.GET("/export", controllers.ExportUploadData)

		manage := uploadData.Group("")
		manage.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			manage.POST("/upload/:dataset", controllers.UploadDataFile)
			// ทดลองอ่านไฟล์ (dry-run) เพื่อดูผลการแม็ปคอลัมน์ก่อนอัปโหลดจริง
			manage.POST("/preview/:dataset", controllers.PreviewUploadDataMapping)
			// ปั๊มตาราง Assembly อัตโนมัติจาก Planning/WH1/Engine + ทะเบียนกลาง
			manage.POST("/assembly/generate", controllers.GenerateAssembly)
			manage.PUT("/:id", controllers.UpdateUploadDataRow)
			manage.DELETE("/:id", controllers.DeleteUploadDataRow)
			manage.DELETE("", controllers.ClearUploadData)
		}
	}

	// ─────────────────────────────────────────────────────────────────────
	// Format Config — ตั้งค่ารองรับ "การเปลี่ยน format" ตอนรัน (สิทธิ์ UPLOAD)
	//   /format-config/column-alias  จับคู่หัวคอลัมน์ไฟล์ → คอลัมน์มาตรฐาน (หน้า Upload Data)
	//   /format-config/code-alias    จับคู่ค่า P/N/S/N/Machine No. รูปแบบใหม่ → แถวทะเบียนกลาง
	// ─────────────────────────────────────────────────────────────────────
	formatConfig := auth.Group("/format-config")
	{
		// อ่านได้ทุก role ที่ login (เผื่อหน้าอื่นอยากโชว์การตั้งค่า)
		formatConfig.GET("/column-alias", controllers.GetColumnAliases)
		formatConfig.GET("/code-alias", controllers.GetCodeAliases)

		manage := formatConfig.Group("")
		manage.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			manage.POST("/column-alias", controllers.CreateColumnAlias)
			manage.DELETE("/column-alias/:id", controllers.DeleteColumnAlias)

			manage.POST("/code-alias", controllers.CreateCodeAlias)
			manage.POST("/code-alias/upload", controllers.UploadCodeAliases)
			manage.DELETE("/code-alias/:id", controllers.DeleteCodeAlias)
		}
	}

	// Part Confirmation — สแกน tag แล้วบันทึกทันที (MC/ITC/CV/SM/MP/PH)
	partCheck := auth.Group("/part-check")
	partCheck.Use(middleware.RoleMiddleware("WH", "LOG"))
	{
		partCheck.GET("", controllers.GetPartChecks)
		partCheck.POST("", controllers.ScanPartCheck)
		partCheck.DELETE("/:id", controllers.DeletePartCheck)

		// ถ่ายรูปป้ายยืนยันหลังสแกน (ITC) — เรียก Claude Vision อ่าน P/N/S/N/IMEI
		// จากรูปมาเทียบกับค่าที่สแกน/ดึงจาก master data ไว้ (ดู partcheck_photo.go)
		partCheck.POST("/:id/photo", controllers.UploadPartCheckPhoto)
	}

	// ─────────────────────────────────────────────────────────────────────
	// Import License — บัญชีแสดงหมายเลขเครื่องแนบท้ายใบอนุญาตนำเข้า
	//
	// WH อัปโหลดไฟล์ Excel ที่ได้มาพร้อมใบอนุญาต (เช่น E05036901604 /
	// TQ60610) เก็บไว้เป็น "ตารางอ้างอิง" แล้วหน้า Part Confirmation เอา
	// ค่าที่สแกนได้มาเทียบว่าตรงกันไหม — หลักการเดียวกับ Master Data
	// ─────────────────────────────────────────────────────────────────────
	// อ่านบัญชี (GET) เปิดให้ทั้ง WH (หน้า Part Confirmation ต้องดึงบัญชีมาเทียบ) และ
	// WH_MANAGER — ส่วนการแก้ไข (อัปโหลด/ลบ/verify) จำกัดเฉพาะ WH_MANAGER
	importLicense := auth.Group("/import-license")
	importLicense.Use(middleware.RoleMiddleware("WH", "LOG"))
	{
		importLicense.GET("", controllers.GetImportLicenseItems)
		importLicense.GET("/summary", controllers.GetImportLicenseSummary)
		importLicense.GET("/alerts", controllers.GetImportLicenseAlerts)

		manage := importLicense.Group("")
		manage.Use(middleware.RoleMiddleware("LOG"))
		{
			manage.POST("/upload", controllers.UploadImportLicenseItems)
			manage.POST("/preview", controllers.PreviewImportLicenseMapping)
			manage.POST("/verify", controllers.VerifyImportLicenseCode)
			manage.POST("/renew", controllers.RenewImportLicense)
			manage.DELETE("/:id", controllers.DeleteImportLicenseItem)
			manage.DELETE("", controllers.ClearImportLicenseItems)
		}
	}

	// ─────────────────────────────────────────────────────────────────────
	// Export License — บัญชีใบอนุญาตส่งออก (คู่กับ Import License)
	//
	// WH อัปโหลดไฟล์ Excel/CSV ที่มีคอลัมน์ ใบขน (Date) / Exception License /
	// Serial Number / Expire date เก็บไว้เป็น "ตารางอ้างอิง" ฝั่งขาออก
	// สิทธิ์เดียวกับ import-license (role WH)
	// ─────────────────────────────────────────────────────────────────────
	exportLicense := auth.Group("/export-license")
	exportLicense.Use(middleware.RoleMiddleware("LOG"))
	{
		exportLicense.GET("", controllers.GetExportLicense)
		exportLicense.GET("/alerts", controllers.GetExportLicenseAlerts)
		exportLicense.GET("/:id/trace", controllers.GetExportLicenseTrace)
		exportLicense.POST("/upload", controllers.UploadExportLicense)
		exportLicense.POST("/preview", controllers.PreviewExportLicenseMapping)
		exportLicense.DELETE("/:id", controllers.DeleteExportLicense)
		exportLicense.DELETE("", controllers.ClearExportLicense)
	}

	// ─────────────────────────────────────────────────────────────────────
	// WH Stock — ตารางอ้างอิงเพิ่มเติมของ Warehouse ที่อัปโหลดจาก Excel
	//   /wh-stock/mc   = ชีต MC  (สต๊อกเครื่อง/ออเดอร์ เอาไว้เช็คของเข้าคลัง)
	//   /wh-stock/inv  = ชีต Inv (รายการอินวอยซ์ + ตำแหน่งจัดเก็บ)
	// สิทธิ์เดียวกับ import-license (role WH)
	// ─────────────────────────────────────────────────────────────────────
	whStock := auth.Group("/wh-stock")
	whStock.Use(middleware.RoleMiddleware("LOG"))
	{
		whStock.GET("/mc", controllers.GetWHMachineStock)
		whStock.POST("/mc/upload", controllers.UploadWHMachineStock)
		whStock.DELETE("/mc/:id", controllers.DeleteWHMachineStock)
		whStock.DELETE("/mc", controllers.ClearWHMachineStock)

		whStock.GET("/inv", controllers.GetWHInvoice)
		whStock.POST("/inv/upload", controllers.UploadWHInvoice)
		whStock.DELETE("/inv/:id", controllers.DeleteWHInvoice)
		whStock.DELETE("/inv", controllers.ClearWHInvoice)
	}

	// ─────────────────────────────────────────────────────────────────────
	// IT Controller (Phase 4: part ใหม่) — ระบบ unit-centric เต็มเส้นทาง
	//
	// เดิม controller/model ชุดนี้เขียนไว้ครบแต่ไม่เคย register route เลย
	// (เรียกจาก frontend api/Itcontroller.js ไม่ได้) — เปิดใช้ที่นี่
	//
	// สิทธิ์: LOG (Logistics) ดูแลนำเข้า/จัดสรร/ส่งออก/อัปโหลด Serial List,
	// WH สแกนรับเข้าคลัง — ให้ทั้งสอง role เข้าถึงกลุ่มนี้ได้ (เหมือน import-license)
	// ─────────────────────────────────────────────────────────────────────
	itc := auth.Group("/it-controller")
	itc.Use(middleware.RoleMiddleware("WH", "LOG"))
	{
		// เอกสาร PDF (Invoice / PO / Import / Export License)
		itc.GET("/documents", controllers.GetITCDocuments)
		itc.POST("/documents", controllers.UploadITCDocument)

		// ใบอนุญาตนำเข้า (กสทช.)
		itc.GET("/import-licenses", controllers.GetImportLicenses)
		itc.POST("/import-licenses", controllers.UpsertImportLicense)

		// ใบอนุญาตส่งออก
		itc.GET("/export-licenses", controllers.GetExportLicenses)
		itc.POST("/export-licenses", controllers.CreateExportLicense)
		itc.GET("/export-licenses/:licenseNo/attachment", controllers.DownloadExportAttachment)

		// unit lifecycle: อัปโหลด Serial List → รับเข้า → จัดสรรประเทศ → จ่าย/ส่งออก
		itc.POST("/units/upload", controllers.UploadSerialList)
		itc.GET("/units", controllers.GetITCUnits)
		itc.POST("/units/receive", controllers.ReceiveITCUnit)
		itc.POST("/units/allocate", controllers.AllocateITCUnits)
		itc.POST("/units/allocate-split", controllers.AllocateITCSplit)
		itc.POST("/units/issue", controllers.IssueITCUnit)
		itc.POST("/units/export", controllers.ExportITCUnit)

		// แจ้งเตือนใบอนุญาตใกล้หมดอายุ + รายงานรายสัปดาห์ + trace ราย unit
		itc.GET("/alerts", controllers.GetITCAlerts)
		itc.GET("/report/weekly", controllers.GetITCWeeklyReport)
		itc.GET("/trace/:itControllerNo", controllers.TraceITCUnit)
	}

	// Generic photo upload — any authenticated role can upload (WH/TSF/QA
	// all attach photos at various steps). Returns a URL to store on the
	// record (e.g. TSFOperator.PhotoURL).
	auth.POST("/uploads", controllers.UploadPhoto)

	// Users — for dropdowns like "เลือกผู้ตรวจสอบ" (InspectedBy). Read-only,
	// no password ever included.
	auth.GET("/users", controllers.GetUsers)

	// Admin — จัดการผู้ใช้ (เพิ่ม/แก้/ลบ/ดูรายชื่อ) เฉพาะ role ADMIN
	admin := auth.Group("/admin/users")
	admin.Use(middleware.RoleMiddleware("ADMIN"))
	{
		admin.GET("", controllers.GetAdminUsers)
		admin.POST("", controllers.CreateUser)
		admin.PATCH("/:id", controllers.UpdateUser)
		admin.DELETE("/:id", controllers.DeleteUser)
	}

	// ─────────────────────────────────────────────────────────────────────
	// MFG Assembly — ผลตรวจตอนประกอบเสร็จ ของฝั่ง MFG (role MFG)
	//
	// สแกน QR ของเครื่องที่ประกอบเสร็จ (บรรจุ Machine No + IT Controller No.)
	// ระบบบันทึกคู่ + flag สถานะ (OK/UNKNOWN/REUSED/DUPLICATE) และแก้ไข/ลบ/
	// เพิ่มเองได้
	// ─────────────────────────────────────────────────────────────────────
	mfgAssembly := auth.Group("/mfg-assembly")
	mfgAssembly.Use(middleware.RoleMiddleware("MFG"))
	{
		mfgAssembly.GET("", controllers.GetMFGAssemblies)
		mfgAssembly.POST("/scan", controllers.ScanMFGAssembly)
		mfgAssembly.POST("", controllers.CreateMFGAssembly)
		mfgAssembly.PATCH("/:id", controllers.UpdateMFGAssembly)
		mfgAssembly.DELETE("/:id", controllers.DeleteMFGAssembly)

		// ถ่ายรูปป้ายยืนยันหลังสแกน — เก็บรูปเป็นหลักฐานเฉย ๆ
		mfgAssembly.POST("/:id/photo", controllers.UploadMFGAssemblyPhoto)
	}

	// QA
	qa := auth.Group("/qa")
	qa.Use(middleware.RoleMiddleware("QA"))
	{
		qa.GET("", controllers.GetQA)
		qa.POST("", controllers.CreateQA)
		// ตารางสรุป QA — รวมข้อมูล WH + MFG + Master Data เมื่อครบเงื่อนไข
		// (MFG = MATCHED และ WH Part Confirmation ตรงกับใบอนุญาต)
		qa.GET("/confirmed", controllers.GetQAConfirmedTable)
	}

	qaConfirm := auth.Group("/qa-confirm")
	qaConfirm.Use(middleware.RoleMiddleware("QA"))
	{
		qaConfirm.GET("", controllers.GetQAConfirm)
		qaConfirm.POST("/:id", controllers.ConfirmQA)
	}

	// Audit log — read-only, any authenticated role
	auditLog := auth.Group("/audit-log")
	{
		auditLog.GET("", controllers.GetAuditLog)
	}
}
