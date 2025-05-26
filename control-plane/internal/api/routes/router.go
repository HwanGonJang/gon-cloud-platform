// control-plane/internal/api/routes/router.go
package routes

import (
	"gon-cloud-platform/control-plane/internal/api/handlers"
	"gon-cloud-platform/control-plane/internal/api/middleware"
	"gon-cloud-platform/control-plane/internal/database"
	"gon-cloud-platform/control-plane/internal/database/repositories"
	"gon-cloud-platform/control-plane/internal/messaging"
	"gon-cloud-platform/control-plane/internal/network"
	"gon-cloud-platform/control-plane/internal/services"
	"gon-cloud-platform/control-plane/internal/utils"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, db *database.Connection, mq *messaging.RabbitMQ, config *utils.Config, logger *utils.Logger) {
	// Initialize repositories
	userRepo := repositories.NewUserRepository(db.DB)
	vpcRepo := repositories.NewVPCRepository(db.DB)

	// Initialize managers
	ovsManager := network.NewOVSManager()

	// Initialize services
	authService := services.NewAuthService(userRepo, config, logger)
	vpcService := services.NewVPCService(vpcRepo, ovsManager, logger)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, config, logger)
	vpcHandler := handlers.NewVPCHandler(vpcService, logger)
	subnetHandler := handlers.NewSubnetHandler(db, mq)
	instanceHandler := handlers.NewInstanceHandler(db, mq)
	securityGroupHandler := handlers.NewSecurityGroupHandler(db, mq)

	// Middleware
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "gcp-api-server",
		})
	})

	// Auth routes (no authentication required)
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.GET("/me", authHandler.GetCurrentUser)
	}

	// API routes (authentication required)
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(config.JWT.Secret))
	{
		// VPC routes
		vpc := api.Group("/vpcs")
		{
			vpc.GET("", vpcHandler.ListVPCs)
			vpc.POST("", vpcHandler.CreateVPC)
			vpc.GET("/:id", vpcHandler.GetVPC)
			vpc.PUT("/:id", vpcHandler.UpdateVPC)
			vpc.DELETE("/:id", vpcHandler.DeleteVPC)
		}

		// Subnet routes
		subnet := api.Group("/subnets")
		{
			subnet.GET("", subnetHandler.ListSubnets)
			subnet.POST("", subnetHandler.CreateSubnet)
			subnet.GET("/:id", subnetHandler.GetSubnet)
			subnet.PUT("/:id", subnetHandler.UpdateSubnet)
			subnet.DELETE("/:id", subnetHandler.DeleteSubnet)
		}

		// Instance routes
		instance := api.Group("/instances")
		{
			instance.GET("", instanceHandler.ListInstances)
			instance.POST("", instanceHandler.CreateInstance)
			instance.GET("/:id", instanceHandler.GetInstance)
			instance.PUT("/:id", instanceHandler.UpdateInstance)
			instance.DELETE("/:id", instanceHandler.DeleteInstance)
			instance.POST("/:id/start", instanceHandler.StartInstance)
			instance.POST("/:id/stop", instanceHandler.StopInstance)
			instance.POST("/:id/restart", instanceHandler.RestartInstance)
		}

		// Security Group routes
		sg := api.Group("/security-groups")
		{
			sg.GET("", securityGroupHandler.ListSecurityGroups)
			sg.POST("", securityGroupHandler.CreateSecurityGroup)
			sg.GET("/:id", securityGroupHandler.GetSecurityGroup)
			sg.PUT("/:id", securityGroupHandler.UpdateSecurityGroup)
			sg.DELETE("/:id", securityGroupHandler.DeleteSecurityGroup)
			sg.POST("/:id/rules", securityGroupHandler.AddRule)
			sg.DELETE("/:id/rules/:rule_id", securityGroupHandler.RemoveRule)
		}

		// Instance Types
		api.GET("/instance-types", instanceHandler.ListInstanceTypes)

		// Images
		api.GET("/images", instanceHandler.ListImages)
	}
}
