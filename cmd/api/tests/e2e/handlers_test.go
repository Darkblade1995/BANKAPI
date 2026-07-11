package e2e_test

import (
	"BANKAPI/internal/database"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (app *e2eApp) registerUser(c *gin.Context) {
	var input struct {
		FirstName string `json:"firstName" binding:"required"`
		LastName  string `json:"lastName" binding:"required"`
		Email     string `json:"email" binding:"required"`
		Password  string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := app.db.Users.GetUserByEmail(input.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already in use"})
		return
	}

	user, err := app.db.Users.CreateUser(input.FirstName, input.LastName, input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user":    user,
	})
}

func (app *e2eApp) loginUser(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := app.db.Users.GetUserByEmail(input.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, err := app.generateToken(user.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	refreshToken := hex.EncodeToString(tokenBytes)

	err = app.db.Tokens.CreateRefreshToken(user.Id, refreshToken, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "login successful",
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"user":         user,
	})
}

func (app *e2eApp) refreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := app.db.Tokens.GetRefreshToken(input.RefreshToken)
	if err != nil || token.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	accessToken, err := app.generateToken(token.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessToken": accessToken})
}

func (app *e2eApp) logout(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app.db.Tokens.DeleteRefreshToken(input.RefreshToken)

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (app *e2eApp) createAccount(c *gin.Context) {
	var input struct {
		Currency string `json:"currency" binding:"required,len=3"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("userId")

	account, err := app.db.Accounts.CreateAccount(userId.(int), input.Currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "account created successfully",
		"account": account,
	})
}

func (app *e2eApp) listAccounts(c *gin.Context) {
	userId, _ := c.Get("userId")

	accounts, err := app.db.Accounts.GetAccountsByUser(userId.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

func (app *e2eApp) getAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	userId, _ := c.Get("userId")

	account, err := app.db.Accounts.GetAccountById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if account.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not allowed to view this account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"account": account})
}

func (app *e2eApp) deposit(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	var input struct {
		Amount int64 `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("userId")

	account, err := app.db.Accounts.GetAccountById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if account.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not allowed to deposit to this account"})
		return
	}

	account, err = app.db.Accounts.Deposit(id, input.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deposit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "deposit successful",
		"account": account,
	})
}

func (app *e2eApp) withdraw(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	var input struct {
		Amount int64 `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("userId")

	account, err := app.db.Accounts.GetAccountById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if account.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not allowed to withdraw from this account"})
		return
	}

	if account.Balance < input.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds"})
		return
	}

	account, err = app.db.Accounts.Deposit(id, -input.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to withdraw"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "withdrawal successful",
		"account": account,
	})
}

func (app *e2eApp) transfer(c *gin.Context) {
	var input struct {
		FromAccountId int   `json:"fromAccountId" binding:"required"`
		ToAccountId   int   `json:"toAccountId" binding:"required"`
		Amount        int64 `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.FromAccountId == input.ToAccountId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot transfer to the same account"})
		return
	}

	userId, _ := c.Get("userId")

	fromAccount, err := app.db.Accounts.GetAccountById(input.FromAccountId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source account not found"})
		return
	}

	if fromAccount.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not allowed to transfer from this account"})
		return
	}

	toAccount, err := app.db.Accounts.GetAccountById(input.ToAccountId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "destination account not found"})
		return
	}

	err = app.db.Accounts.Transfer(
		input.FromAccountId,
		input.ToAccountId,
		input.Amount,
		input.Amount,
		fromAccount.Currency,
		toAccount.Currency,
		1.0,
	)
	if err != nil {
		if err == database.ErrInsufficientFunds {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "transfer successful",
		"fromCurrency": fromAccount.Currency,
		"toCurrency":   toAccount.Currency,
		"amount":       input.Amount,
	})
}

func (app *e2eApp) getTransactions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	userId, _ := c.Get("userId")

	account, err := app.db.Accounts.GetAccountById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if account.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not allowed to view this account transactions"})
		return
	}

	transactions, err := app.db.Accounts.GetTransactionsByAccount(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}