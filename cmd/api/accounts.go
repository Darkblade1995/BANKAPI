package main

import (
	"errors"
	"net/http"
	"strconv"

	"BANKAPI/internal/database"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// @Summary      Crear cuenta
// @Description  Crea una nueva cuenta bancaria para el usuario autenticado
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input  body      object{currency=string}  true  "Moneda (COP, USD, EUR)"
// @Success      201    {object}  object{message=string,account=object}
// @Failure      400    {object}  object{error=string}
// @Failure      401    {object}  object{error=string}
// @Router       /accounts [post]
func (app *application) createAccount(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create account",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "account created successfully",
		"account": account,
	})
}

// @Summary      Listar cuentas
// @Description  Retorna todas las cuentas del usuario autenticado
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{accounts=array}
// @Failure      401  {object}  object{error=string}
// @Router       /accounts [get]
func (app *application) listAccounts(c *gin.Context) {
	userId, _ := c.Get("userId")

	accounts, err := app.db.Accounts.GetAccountsByUser(userId.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch accounts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": accounts,
	})
}

// @Summary      Ver cuenta
// @Description  Retorna el detalle de una cuenta específica
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "ID de la cuenta"
// @Success      200  {object}  object{account=object}
// @Failure      403  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Router       /accounts/{id} [get]
func (app *application) getAccount(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not allowed to view this account",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account": account,
	})
}

// @Summary      Depositar
// @Description  Deposita dinero en una cuenta
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int                     true  "ID de la cuenta"
// @Param        input  body      object{amount=integer}  true  "Monto en centavos"
// @Success      200    {object}  object{message=string,account=object}
// @Failure      400    {object}  object{error=string}
// @Failure      403    {object}  object{error=string}
// @Router       /accounts/{id}/deposit [post]
func (app *application) deposit(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not allowed to deposit to this account",
		})
		return
	}

	account, err = app.db.Accounts.Deposit(id, input.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to deposit",
		})
		return
	}

	go func() {
		user, err := app.db.Users.GetUserById(userId.(int))
		if err == nil {
			if err := app.mailer.SendDepositNotification(user.Email, user.FirstName, input.Amount, account.Currency); err != nil {
				app.logger.Error("failed to send deposit notification", zap.Error(err))
			}
			app.hub.SendToUser(userId.(int), "deposit", map[string]interface{}{
				"message":  "deposit received",
				"amount":   input.Amount,
				"currency": account.Currency,
				"balance":  account.Balance,
			})
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "deposit successful",
		"account": account,
	})
}

// @Summary      Retirar
// @Description  Retira dinero de una cuenta
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int                     true  "ID de la cuenta"
// @Param        input  body      object{amount=integer}  true  "Monto en centavos"
// @Success      200    {object}  object{message=string,account=object}
// @Failure      400    {object}  object{error=string}
// @Failure      403    {object}  object{error=string}
// @Router       /accounts/{id}/withdraw [post]
func (app *application) withdraw(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not allowed to withdraw from this account",
		})
		return
	}

	if account.Balance < input.Amount {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "insufficient funds",
		})
		return
	}

	account, err = app.db.Accounts.Deposit(id, -input.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to withdraw",
		})
		return
	}

	go func() {
		app.hub.SendToUser(userId.(int), "withdrawal", map[string]interface{}{
			"message":  "withdrawal successful",
			"amount":   input.Amount,
			"currency": account.Currency,
			"balance":  account.Balance,
		})
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "withdrawal successful",
		"account": account,
	})
}

// @Summary      Transferir
// @Description  Transfiere dinero entre cuentas con conversión automática de moneda
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input  body      object{fromAccountId=integer,toAccountId=integer,amount=integer}  true  "Datos de transferencia"
// @Success      200    {object}  object{message=string,fromCurrency=string,toCurrency=string,amount=integer,convertedAmount=integer,exchangeRate=number}
// @Failure      400    {object}  object{error=string}
// @Failure      403    {object}  object{error=string}
// @Router       /transfers [post]
func (app *application) transfer(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cannot transfer to the same account",
		})
		return
	}

	userId, _ := c.Get("userId")

	fromAccount, err := app.db.Accounts.GetAccountById(input.FromAccountId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source account not found"})
		return
	}

	if fromAccount.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not allowed to transfer from this account",
		})
		return
	}

	toAccount, err := app.db.Accounts.GetAccountById(input.ToAccountId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "destination account not found"})
		return
	}

	convertedAmount, exchangeRate, err := app.converter.Convert(
		input.Amount,
		fromAccount.Currency,
		toAccount.Currency,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to convert currency",
		})
		return
	}

	err = app.db.Accounts.Transfer(
		input.FromAccountId,
		input.ToAccountId,
		input.Amount,
		convertedAmount,
		fromAccount.Currency,
		toAccount.Currency,
		exchangeRate,
	)
	if err != nil {
		if errors.Is(err, database.ErrInsufficientFunds) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "insufficient funds",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to transfer",
		})
		return
	}

	go func() {
		user, err := app.db.Users.GetUserById(userId.(int))
		if err == nil {
			if err := app.mailer.SendTransferNotification(user.Email, user.FirstName, input.Amount, fromAccount.Currency, input.ToAccountId); err != nil {
				app.logger.Error("failed to send transfer notification", zap.Error(err))
			}
			app.hub.SendToUser(userId.(int), "transfer_sent", map[string]interface{}{
				"message":     "transfer sent",
				"amount":      input.Amount,
				"currency":    fromAccount.Currency,
				"toAccountId": input.ToAccountId,
			})
		}

		toAccountData, err := app.db.Accounts.GetAccountById(input.ToAccountId)
		if err == nil {
			app.hub.SendToUser(toAccountData.UserId, "transfer_received", map[string]interface{}{
				"message":       "transfer received",
				"amount":        convertedAmount,
				"currency":      toAccount.Currency,
				"fromAccountId": input.FromAccountId,
			})
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":         "transfer successful",
		"fromCurrency":    fromAccount.Currency,
		"toCurrency":      toAccount.Currency,
		"amount":          input.Amount,
		"convertedAmount": convertedAmount,
		"exchangeRate":    exchangeRate,
	})
}

// @Summary      Historial de transacciones
// @Description  Retorna el historial de transacciones de una cuenta con paginación
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int  true   "ID de la cuenta"
// @Param        page   query     int  false  "Página (default: 1)"
// @Param        limit  query     int  false  "Resultados por página (default: 10)"
// @Success      200  {object}  object{transactions=array,total=integer,page=integer,limit=integer,totalPages=integer}
// @Failure      403  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Router       /accounts/{id}/transactions [get]
func (app *application) getTransactions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	userId, _ := c.Get("userId")

	account, err := app.db.Accounts.GetAccountById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if account.UserId != userId.(int) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not allowed to view this account transactions",
		})
		return
	}

	result, err := app.db.Accounts.GetTransactionsByAccount(id, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch transactions",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}