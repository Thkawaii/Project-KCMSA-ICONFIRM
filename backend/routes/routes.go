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

	// Warehouse
	warehouse := auth.Group("/warehouse")
	warehouse.Use(middleware.RoleMiddleware("WH"))
	{
		warehouse.GET("", controllers.GetWarehouse)
		warehouse.POST("", controllers.CreateWarehouse)
		warehouse.POST("/upload", controllers.UploadWarehouseStock)
		warehouse.POST("/:id/issue", controllers.IssueWarehouse)
	}

	whConfirm := auth.Group("/wh-confirm")
	{
		whConfirm.GET("", controllers.GetWHConfirm)

		receive := whConfirm.Group("")
		receive.Use(middleware.RoleMiddleware("TSF"))
		{
			receive.POST("/:id/receive", controllers.ReceiveWHConfirm)
		}
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