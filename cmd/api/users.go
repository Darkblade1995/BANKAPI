package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// @Summary      Registrar usuario
// @Description  Crea un nuevo usuario en el sistema
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      object{firstName=string,lastName=string,email=string,password=string}  true  "Datos del usuario"
// @Success      201    {object}  object{message=string,user=object}
// @Failure      400    {object}  object{error=string}
// @Failure      409    {object}  object{error=string}
// @Router       /users/register [post]
func (app *application) registerUser(c *gin.Context) {
	var input struct {
		FirstName string `json:"firstName" binding:"required,min=2"`
		LastName  string `json:"lastName" binding:"required,min=2"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := app.db.Users.CreateUser(
		input.FirstName,
		input.LastName,
		input.Email,
		input.Password,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") ||
			strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "a user with that email already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	go func() {
		if err := app.mailer.SendWelcomeEmail(user.Email, user.FirstName); err != nil {
			app.logger.Error("failed to send welcome email", zap.Error(err))
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user":    user,
	})
}

// @Summary      Login
// @Description  Autentica un usuario y devuelve tokens JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      object{email=string,password=string}  true  "Credenciales"
// @Success      200    {object}  object{message=string,accessToken=string,refreshToken=string,user=object}
// @Failure      400    {object}  object{error=string}
// @Failure      401    {object}  object{error=string}
// @Router       /users/login [post]
func (app *application) loginUser(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	locked, err := app.cache.IsAccountLocked(ctx, input.Email)
	if err != nil {
		app.logger.Error("failed to check account lock", zap.Error(err))
	}
	if locked {
		c.JSON(http.StatusLocked, gin.H{
			"error": "account is temporarily locked due to too many failed attempts — try again in 30 minutes",
		})
		return
	}

	user, err := app.db.Users.GetUserByEmail(input.Email)
	if err != nil {
		app.cache.IncrFailedAttempts(ctx, input.Email)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid email or password",
		})
		return
	}

	if err := app.db.Users.CheckPassword(user, input.Password); err != nil {
		count, _ := app.cache.IncrFailedAttempts(ctx, input.Email)

		if count >= 3 {
			app.cache.LockAccount(ctx, input.Email)

			go func() {
				if err := app.mailer.SendSecurityAlert(user.Email, user.FirstName); err != nil {
					app.logger.Error("failed to send security alert", zap.Error(err))
				}
			}()

			c.JSON(http.StatusLocked, gin.H{
				"error": "account locked due to too many failed attempts — try again in 30 minutes",
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": fmt.Sprintf("invalid email or password — %d attempts remaining", 3-count),
		})
		return
	}

	app.cache.ResetFailedAttempts(ctx, input.Email)

	accessToken, err := app.generateAccessToken(user.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate access token",
		})
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate refresh token",
		})
		return
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = app.db.Tokens.CreateRefreshToken(user.Id, refreshToken, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save refresh token",
		})
		return
	}

	go func() {
		if err := app.mailer.SendLoginNotification(user.Email, user.FirstName); err != nil {
			app.logger.Error("failed to send login notification", zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":      "login successful",
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"user":         user,
	})
}

// @Summary      Refresh token
// @Description  Genera un nuevo access token y rota el refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      object{refreshToken=string}  true  "Refresh token"
// @Success      200    {object}  object{accessToken=string,refreshToken=string}
// @Failure      401    {object}  object{error=string}
// @Router       /auth/refresh [post]
func (app *application) refreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rt, err := app.db.Tokens.GetRefreshToken(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired refresh token",
		})
		return
	}

	// Borrar el refresh token actual
	if err := app.db.Tokens.DeleteRefreshToken(input.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to rotate refresh token",
		})
		return
	}

	// Generar nuevo access token
	accessToken, err := app.generateAccessToken(rt.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate access token",
		})
		return
	}

	// Generar nuevo refresh token
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate refresh token",
		})
		return
	}

	// Guardar nuevo refresh token en BD
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := app.db.Tokens.CreateRefreshToken(rt.UserId, newRefreshToken, expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  accessToken,
		"refreshToken": newRefreshToken,
	})
}

// @Summary      Logout
// @Description  Invalida el refresh token del usuario
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      object{refreshToken=string}  true  "Refresh token"
// @Success      200    {object}  object{message=string}
// @Failure      500    {object}  object{error=string}
// @Router       /auth/logout [post]
func (app *application) logout(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := app.db.Tokens.DeleteRefreshToken(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}
