package e2e_test

import (
	"BANKAPI/internal/cache"
	"BANKAPI/internal/database"
	"BANKAPI/internal/mailer"
	ws "BANKAPI/internal/websocket"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type e2eApp struct {
	jwtSecret string
	db        *database.Models
	logger    *zap.Logger
	mailer    *mailer.Mailer
	cache     *cache.Cache
	hub       *ws.Hub
}

type e2eClaims struct {
	UserId int `json:"userId"`
	jwt.RegisteredClaims
}

func setupTestRouter(models *database.Models, c *cache.Cache, secret string) *gin.Engine {
	logger, _ := zap.NewDevelopment()

	m := mailer.NewMailer("test-key", "test@test.com")

	hub := ws.NewHub()
	go hub.Run()

	app := &e2eApp{
		jwtSecret: secret,
		db:        models,
		logger:    logger,
		mailer:    m,
		cache:     c,
		hub:       hub,
	}

	return app.buildRouter()
}

func (app *e2eApp) buildRouter() *gin.Engine {
	g := gin.New()

	g.Use(cors.Default())
	g.Use(gin.Recovery())

	v1 := g.Group("/v1")
	{
		v1.POST("/users/register", app.registerUser)
		v1.POST("/users/login", app.loginUser)
		v1.POST("/auth/refresh", app.refreshToken)
		v1.POST("/auth/logout", app.logout)

		auth := v1.Group("/")
		auth.Use(app.authMiddleware())
		{
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

func (app *e2eApp) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &e2eClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(app.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims := token.Claims.(*e2eClaims)
		c.Set("userId", claims.UserId)
		c.Next()
	}
}

func (app *e2eApp) generateToken(userId int) (string, error) {
	claims := e2eClaims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(app.jwtSecret))
}