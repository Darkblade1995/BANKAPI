package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (app *application) healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := app.db.Users.DB.PingContext(ctx); err != nil {
		dbStatus = "unavailable"
	}

	redisStatus := "ok"
	if err := app.cache.Ping(ctx); err != nil {
		redisStatus = "unavailable"
	}

	status := "ok"
	httpStatus := http.StatusOK

	if dbStatus != "ok" || redisStatus != "ok" {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status": status,
		"services": gin.H{
			"database": dbStatus,
			"redis":    redisStatus,
		},
	})
}