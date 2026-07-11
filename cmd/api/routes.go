package main

import (
	"net/http"

	"BANKAPI/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (app *application) routes() http.Handler {
	docs.SwaggerInfo.BasePath = "/v1"

	g := gin.New()

	g.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	g.OPTIONS("/*any", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	g.Use(gin.Recovery())
	g.Use(rateLimitMiddleware())
	g.Use(loggerMiddleware(app.logger))

	g.GET("/health", app.healthCheck)
	g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := g.Group("/v1")
	{
		v1.POST("/users/register", app.registerUser)
		v1.POST("/users/login", app.loginUser)
		v1.POST("/auth/refresh", app.refreshToken)
		v1.POST("/auth/logout", app.logout)

		auth := v1.Group("/")
		auth.Use(app.authMiddleware())
		{
			auth.GET("/ws", app.hub.HandleWebSocket)
			auth.POST("/accounts", app.createAccount)
			auth.GET("/accounts", app.listAccounts)
			auth.GET("/accounts/:id", app.getAccount)
			auth.POST("/accounts/:id/deposit", app.deposit)
			auth.POST("/accounts/:id/withdraw", app.withdraw)
			auth.POST("/transfers", app.transfer)
			auth.GET("/accounts/:id/transactions", app.getTransactions)
		}
	}

	return g
}