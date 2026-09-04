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
		masterData.POST("", controllers.CreateMasterData)

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

	auth.GET("/machine-profile/:machineNo", controllers.GetMachineProfile)

	// รายละเอียดเครื่องทุกคัน รวมจาก ALL PART / Planning / WH1 / WH2 / Engine
	// ใช้แสดงรายละเอียดในหน้า WH, MFG และ LOG
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
			manage.PUT("/:id", controllers.UpdateUploadDataRow)
			manage.DELETE("/:id", controllers.DeleteUploadDataRow)
			manage.DELETE("", controllers.ClearUploadData)
		}
	}

	formatConfig := auth.Group("/format-config")
	{
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

	partCheck := auth.Group("/part-check")
	partCheck.Use(middleware.RoleMiddleware("WH", "LOG"))
	{
		partCheck.GET("", controllers.GetPartChecks)
		partCheck.POST("", controllers.ScanPartCheck)
		partCheck.DELETE("/:id", controllers.DeletePartCheck)

	}

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

	mfgAssembly := auth.Group("/mfg-assembly")
	mfgAssembly.Use(middleware.RoleMiddleware("MFG"))
	{
		mfgAssembly.GET("", controllers.GetMFGAssemblies)
		mfgAssembly.POST("/scan", controllers.ScanMFGAssembly)
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
