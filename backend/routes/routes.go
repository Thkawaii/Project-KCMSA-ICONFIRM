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

	uploadData := auth.Group("/upload-data")
	{
		uploadData.GET("", controllers.GetUploadData)
		uploadData.GET("/export", controllers.ExportUploadData)

		manage := uploadData.Group("")
		manage.Use(middleware.RoleMiddleware("UPLOAD", "ADMIN"))
		{
			manage.POST("/upload/:dataset", controllers.UploadDataFile)
			manage.POST("/preview/:dataset", controllers.PreviewUploadDataMapping)
			manage.POST("/assembly/generate", controllers.GenerateAssembly)
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

		partCheck.POST("/:id/photo", controllers.UploadPartCheckPhoto)
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
		exportLicense.DELETE("/:id", controllers.DeleteExportLicense)
		exportLicense.DELETE("", controllers.ClearExportLicense)
	}

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

	itc := auth.Group("/it-controller")
	itc.Use(middleware.RoleMiddleware("WH", "LOG"))
	{
		itc.GET("/documents", controllers.GetITCDocuments)
		itc.POST("/documents", controllers.UploadITCDocument)

		itc.GET("/import-licenses", controllers.GetImportLicenses)
		itc.POST("/import-licenses", controllers.UpsertImportLicense)

		itc.GET("/export-licenses", controllers.GetExportLicenses)
		itc.POST("/export-licenses", controllers.CreateExportLicense)
		itc.GET("/export-licenses/:licenseNo/attachment", controllers.DownloadExportAttachment)

		itc.POST("/units/upload", controllers.UploadSerialList)
		itc.GET("/units", controllers.GetITCUnits)
		itc.POST("/units/receive", controllers.ReceiveITCUnit)
		itc.POST("/units/allocate", controllers.AllocateITCUnits)
		itc.POST("/units/allocate-split", controllers.AllocateITCSplit)
		itc.POST("/units/issue", controllers.IssueITCUnit)
		itc.POST("/units/export", controllers.ExportITCUnit)

		itc.GET("/alerts", controllers.GetITCAlerts)
		itc.GET("/report/weekly", controllers.GetITCWeeklyReport)
		itc.GET("/trace/:itControllerNo", controllers.TraceITCUnit)
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
		qa.GET("", controllers.GetQA)
		qa.POST("", controllers.CreateQA)
		qa.GET("/confirmed", controllers.GetQAConfirmedTable)
	}

	qaConfirm := auth.Group("/qa-confirm")
	qaConfirm.Use(middleware.RoleMiddleware("QA"))
	{
		qaConfirm.GET("", controllers.GetQAConfirm)
		qaConfirm.POST("/:id", controllers.ConfirmQA)
	}

	auditLog := auth.Group("/audit-log")
	{
		auditLog.GET("", controllers.GetAuditLog)
	}
}
