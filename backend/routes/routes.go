package routes

import (
	"iconfirm/controllers"
	"iconfirm/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {

	r.POST("/login", controllers.Login)

	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())

	masterData := auth.Group("/master-data")
	{
		masterData.GET("", controllers.GetMasterData)
		masterData.GET("/summary", controllers.GetMasterDataSummary)

		// อัปโหลด/ลบ: UPLOAD และ ADMIN
		manage := masterData.Group("")
		manage.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			manage.POST("", controllers.CreateMasterData)
			manage.POST("/upload", controllers.UploadMasterData)
			manage.POST("/preview", controllers.PreviewMasterDataChanges)
			manage.DELETE("/:id", controllers.DeleteMasterData)
			manage.DELETE("", controllers.ClearMasterData)
		}

		// แก้ไขในตาราง (ปุ่มดินสอ): ADMIN เท่านั้น — UPLOAD แก้ข้อมูลใน Excel แล้วอัปโหลดใหม่
		edit := masterData.Group("")
		edit.Use(middleware.RoleMiddleware("ADMIN"))
		{
			edit.PATCH("/:id", controllers.UpdateMasterData)
		}
	}

	auth.GET("/machine-profile/:machineNo", controllers.GetMachineProfile)

	auth.GET("/machine-plans", controllers.GetMachinePlans)

	uploadData := auth.Group("/upload-data")
	{
		uploadData.GET("", controllers.GetUploadData)
		uploadData.GET("/export", controllers.ExportUploadData)

		manage := uploadData.Group("")
		manage.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			manage.POST("/upload/:dataset", controllers.UploadDataFile)
			manage.POST("/preview/:dataset", controllers.PreviewUploadDataMapping)
			manage.DELETE("/:id", controllers.DeleteUploadDataRow)
			manage.DELETE("", controllers.ClearUploadData)
		}

		edit := uploadData.Group("")
		edit.Use(middleware.RoleMiddleware("ADMIN"))
		{
			edit.PUT("/:id", controllers.UpdateUploadDataRow)
			edit.PATCH("/:id", controllers.UpdateUploadDataRow)
		}
	}

	formatConfig := auth.Group("/format-config")
	{
		formatConfig.GET("/column-alias", controllers.GetColumnAliases)
		formatConfig.GET("/code-alias", controllers.GetCodeAliases)

		manage := formatConfig.Group("")
		manage.Use(middleware.RoleMiddleware("ADMIN"))
		{
			manage.POST("/column-alias", controllers.CreateColumnAlias)
			manage.DELETE("/column-alias/:id", controllers.DeleteColumnAlias)

			manage.POST("/code-alias", controllers.CreateCodeAlias)
			manage.POST("/code-alias/upload", controllers.UploadCodeAliases)
			manage.DELETE("/code-alias/:id", controllers.DeleteCodeAlias)
		}
	}

	partCheck := auth.Group("/part-check")
	partCheck.Use(middleware.RoleMiddleware("WH", "LOG"))
	{
		partCheck.GET("", controllers.GetPartChecks)
		partCheck.POST("", controllers.ScanPartCheck)
		partCheck.DELETE("/:id", controllers.DeletePartCheck)

	}

	// Import / Export License: ไม่มีการแก้ไขในตาราง — LOG แก้/ลบข้อมูลใน Excel แล้วอัปโหลดไฟล์เดิมอีกครั้ง
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
			manage.POST("/complete", controllers.SetImportLicenseComplete)
			manage.POST("/bulk-delete", controllers.BulkDeleteImportLicenseItems)
			manage.DELETE("/:id", controllers.DeleteImportLicenseItem)
			manage.DELETE("", controllers.ClearImportLicenseItems)
		}
	}

	exportLicense := auth.Group("/export-license")
	exportLicense.Use(middleware.RoleMiddleware("LOG"))
	{
		exportLicense.GET("", controllers.GetExportLicense)
		exportLicense.GET("/alerts", controllers.GetExportLicenseAlerts)
		exportLicense.GET("/:id/trace", controllers.GetExportLicenseTrace)
		exportLicense.POST("/upload", controllers.UploadExportLicense)
		exportLicense.POST("/preview", controllers.PreviewExportLicenseMapping)
		exportLicense.POST("/renew", controllers.RenewExportLicense)
		exportLicense.POST("/complete", controllers.SetExportLicenseComplete)
		exportLicense.POST("/bulk-delete", controllers.BulkDeleteExportLicense)
		exportLicense.DELETE("/:id", controllers.DeleteExportLicense)
		exportLicense.DELETE("", controllers.ClearExportLicense)
	}

	auth.POST("/uploads", controllers.UploadPhoto)

	auth.GET("/users", controllers.GetUsers)

	admin := auth.Group("/admin/users")
	admin.Use(middleware.RoleMiddleware("ADMIN"))
	{
		admin.GET("", controllers.GetAdminUsers)
		admin.POST("", controllers.CreateUser)
		admin.PATCH("/:id", controllers.UpdateUser)
		admin.DELETE("/:id", controllers.DeleteUser)
	}

	adminMail := auth.Group("/admin/mail-recipients")
	adminMail.Use(middleware.RoleMiddleware("ADMIN"))
	{
		adminMail.GET("", controllers.GetMailRecipients)
		adminMail.POST("", controllers.CreateMailRecipient)
		adminMail.PATCH("/:id", controllers.UpdateMailRecipient)
		adminMail.DELETE("/:id", controllers.DeleteMailRecipient)
	}

	mfgAssembly := auth.Group("/mfg-assembly")
	mfgAssembly.Use(middleware.RoleMiddleware("MFG"))
	{
		mfgAssembly.GET("", controllers.GetMFGAssemblies)
		mfgAssembly.POST("/scan", controllers.ScanMFGAssembly)
		mfgAssembly.POST("/spec-check", controllers.CheckMFGSpecQR)
		mfgAssembly.POST("", controllers.CreateMFGAssembly)
		mfgAssembly.PATCH("/:id", controllers.UpdateMFGAssembly)
		mfgAssembly.DELETE("/:id", controllers.DeleteMFGAssembly)

		mfgAssembly.POST("/:id/photo", controllers.UploadMFGAssemblyPhoto)
	}

	qa := auth.Group("/qa")
	qa.Use(middleware.RoleMiddleware("QA"))
	{
		qa.GET("/confirmed", controllers.GetQAConfirmedTable)
		qa.GET("/part-scan-summary", controllers.GetQAPartScanSummary)
	}

	auditLog := auth.Group("/audit-log")
	{
		auditLog.GET("", controllers.GetAuditLog)
	}
}
