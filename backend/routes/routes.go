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
		manage.Use(middleware.RoleMiddleware("UPLOAD"))
		{
			manage.POST("/upload", controllers.UploadMasterData)
			manage.DELETE("/:id", controllers.DeleteMasterData)
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
		upload.Use(middleware.RoleMiddleware("UPLOAD"))
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
		manage.Use(middleware.RoleMiddleware("UPLOAD"))
		{
			manage.POST("/upload/:dataset", controllers.UploadDataFile)
			manage.DELETE("/:id", controllers.DeleteUploadDataRow)
			manage.DELETE("", controllers.ClearUploadData)
		}
	}

	// Part Confirmation — สแกน tag แล้วบันทึกทันที (MC/ITC/CV/SM/MP/PH)
	partCheck := auth.Group("/part-check")
	partCheck.Use(middleware.RoleMiddleware("WH"))
	{
		partCheck.GET("", controllers.GetPartChecks)
		partCheck.POST("", controllers.ScanPartCheck)
		partCheck.DELETE("/:id", controllers.DeletePartCheck)
	}

	// ─────────────────────────────────────────────────────────────────────
	// Matching Assembly — ผลการจับคู่ประกอบ IT Controller เข้ากับเครื่อง
	// สร้างอัตโนมัติเมื่อสแกน IT Controller สำเร็จบนหน้า Part Confirmation
	// และแก้ไข/ลบ/เพิ่มเองได้ (สิทธิ์เดียวกับ part-check คือ role WH)
	// ─────────────────────────────────────────────────────────────────────
	matchingAssembly := auth.Group("/matching-assembly")
	matchingAssembly.Use(middleware.RoleMiddleware("WH"))
	{
		matchingAssembly.GET("", controllers.GetMatchingAssemblies)
		matchingAssembly.POST("", controllers.CreateMatchingAssembly)
		matchingAssembly.PATCH("/:id", controllers.UpdateMatchingAssembly)
		matchingAssembly.DELETE("/:id", controllers.DeleteMatchingAssembly)
	}

	// ─────────────────────────────────────────────────────────────────────
	// Import License — บัญชีแสดงหมายเลขเครื่องแนบท้ายใบอนุญาตนำเข้า
	//
	// WH อัปโหลดไฟล์ Excel ที่ได้มาพร้อมใบอนุญาต (เช่น E05036901604 /
	// TQ60610) เก็บไว้เป็น "ตารางอ้างอิง" แล้วหน้า Part Confirmation เอา
	// ค่าที่สแกนได้มาเทียบว่าตรงกันไหม — หลักการเดียวกับ Master Data
	// ─────────────────────────────────────────────────────────────────────
	importLicense := auth.Group("/import-license")
	importLicense.Use(middleware.RoleMiddleware("WH"))
	{
		importLicense.GET("", controllers.GetImportLicenseItems)
		importLicense.GET("/summary", controllers.GetImportLicenseSummary)
		importLicense.GET("/alerts", controllers.GetImportLicenseAlerts)
		importLicense.POST("/upload", controllers.UploadImportLicenseItems)
		importLicense.POST("/verify", controllers.VerifyImportLicenseCode)
		importLicense.DELETE("/:id", controllers.DeleteImportLicenseItem)
		importLicense.DELETE("", controllers.ClearImportLicenseItems)
	}

	// ─────────────────────────────────────────────────────────────────────
	// Export License — บัญชีใบอนุญาตส่งออก (คู่กับ Import License)
	//
	// WH อัปโหลดไฟล์ Excel/CSV ที่มีคอลัมน์ ใบขน (Date) / Exception License /
	// Serial Number / Expire date เก็บไว้เป็น "ตารางอ้างอิง" ฝั่งขาออก
	// สิทธิ์เดียวกับ import-license (role WH)
	// ─────────────────────────────────────────────────────────────────────
	exportLicense := auth.Group("/export-license")
	exportLicense.Use(middleware.RoleMiddleware("WH"))
	{
		exportLicense.GET("", controllers.GetExportLicense)
		exportLicense.GET("/alerts", controllers.GetExportLicenseAlerts)
		exportLicense.POST("/upload", controllers.UploadExportLicense)
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
	whStock.Use(middleware.RoleMiddleware("WH"))
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

	// Generic photo upload — any authenticated role can upload (WH/TSF/QA
	// all attach photos at various steps). Returns a URL to store on the
	// record (e.g. TSFOperator.PhotoURL).
	auth.POST("/uploads", controllers.UploadPhoto)

	// Users — for dropdowns like "เลือกผู้ตรวจสอบ" (InspectedBy). Read-only,
	// no password ever included.
	auth.GET("/users", controllers.GetUsers)

	// TSF Operator
	tsf := auth.Group("/tsf")
	tsf.Use(middleware.RoleMiddleware("TSF"))
	{
		tsf.GET("", controllers.GetTSF)
		tsf.GET("/by-machine/:machineNo", controllers.GetTSFByMachine)
		tsf.POST("", controllers.CreateTSF)
		tsf.PATCH("/:id", controllers.UpdateTSF)
	}

	// TSF confirm — QA also needs to read/confirm these, so it's
	// intentionally outside the "TSF" role-only group above.
	tsfConfirm := auth.Group("/tsf-confirm")
	{
		tsfConfirm.GET("", controllers.GetTSFConfirm)
		tsfConfirm.POST("/:id", controllers.ConfirmTSF)
	}

	// QA
	qa := auth.Group("/qa")
	qa.Use(middleware.RoleMiddleware("QA"))
	{
		qa.GET("", controllers.GetQA)
		qa.POST("", controllers.CreateQA)
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
