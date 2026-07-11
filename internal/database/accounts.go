package database

import (
	"database/sql"
	"errors"
)

type AccountModel struct {
	DB *sql.DB
}

type Account struct {
	Id        int    `json:"id"`
	UserId    int    `json:"userId"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"createdAt"`
}

type Transaction struct {
	Id              int     `json:"id"`
	FromAccount     int     `json:"fromAccount"`
	ToAccount       int     `json:"toAccount"`
	Amount          int64   `json:"amount"`
	FromCurrency    string  `json:"fromCurrency"`
	ToCurrency      string  `json:"toCurrency"`
	ExchangeRate    float64 `json:"exchangeRate"`
	ConvertedAmount int64   `json:"convertedAmount"`
	CreatedAt       string  `json:"createdAt"`
}

type PaginatedTransactions struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	Limit        int           `json:"limit"`
	TotalPages   int           `json:"totalPages"`
}

var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrAccountNotFound = errors.New("account not found")

func (m AccountModel) CreateAccount(userId int, currency string) (Account, error) {
	query := `
		INSERT INTO accounts (user_id, balance, currency)
		VALUES ($1, 0, $2)
		RETURNING id, user_id, balance, currency, created_at
	`

	var account Account
	err := m.DB.QueryRow(query, userId, currency).Scan(
		&account.Id,
		&account.UserId,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)
	if err != nil {
		return Account{}, err
	}

	return account, nil
}

func (m AccountModel) GetAccountById(id int) (Account, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at
		FROM accounts
		WHERE id = $1
	`

	var account Account
	err := m.DB.QueryRow(query, id).Scan(
		&account.Id,
		&account.UserId,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)
	if err != nil {
		return Account{}, err
	}

	return account, nil
}

func (m AccountModel) GetAccountsByUser(userId int) ([]Account, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at
		FROM accounts
		WHERE user_id = $1
	`

	rows, err := m.DB.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		err := rows.Scan(
			&account.Id,
			&account.UserId,
			&account.Balance,
			&account.Currency,
			&account.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	if accounts == nil {
		return []Account{}, nil
	}

	return accounts, nil
}

func (m AccountModel) Deposit(accountId int, amount int64) (Account, error) {
	query := `
		UPDATE accounts
		SET balance = balance + $1
		WHERE id = $2
		RETURNING id, user_id, balance, currency, created_at
	`

	var account Account
	err := m.DB.QueryRow(query, amount, accountId).Scan(
		&account.Id,
		&account.UserId,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)
	if err != nil {
		return Account{}, err
	}

	return account, nil
}

func (m AccountModel) Transfer(
	fromAccountId, toAccountId int,
	amount, convertedAmount int64,
	fromCurrency, toCurrency string,
	exchangeRate float64,
) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var balance int64
	err = tx.QueryRow(`SELECT balance FROM accounts WHERE id = $1`, fromAccountId).Scan(&balance)
	if err != nil {
		return ErrAccountNotFound
	}

	if balance < amount {
		return ErrInsufficientFunds
	}

	_, err = tx.Exec(`
		UPDATE accounts SET balance = balance - $1 WHERE id = $2
	`, amount, fromAccountId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE accounts SET balance = balance + $1 WHERE id = $2
	`, convertedAmount, toAccountId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO transactions 
		(from_account_id, to_account_id, amount, from_currency, to_currency, exchange_rate, converted_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, fromAccountId, toAccountId, amount, fromCurrency, toCurrency, exchangeRate, convertedAmount)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (m AccountModel) GetTransactionsByAccount(accountId, page, limit int) (PaginatedTransactions, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM transactions
		WHERE from_account_id = $1 OR to_account_id = $1
	`

	var total int
	err := m.DB.QueryRow(countQuery, accountId).Scan(&total)
	if err != nil {
		return PaginatedTransactions{}, err
	}

	offset := (page - 1) * limit

	query := `
		SELECT id, from_account_id, to_account_id, amount,
		       from_currency, to_currency, exchange_rate, converted_amount, created_at
		FROM transactions
		WHERE from_account_id = $1 OR to_account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := m.DB.Query(query, accountId, limit, offset)
	if err != nil {
		return PaginatedTransactions{}, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		err := rows.Scan(
			&t.Id,
			&t.FromAccount,
			&t.ToAccount,
			&t.Amount,
			&t.FromCurrency,
			&t.ToCurrency,
			&t.ExchangeRate,
			&t.ConvertedAmount,
			&t.CreatedAt,
		)
		if err != nil {
			return PaginatedTransactions{}, err
		}
		transactions = append(transactions, t)
	}

	if transactions == nil {
		transactions = []Transaction{}
	}

	totalPages := (total + limit - 1) / limit

	return PaginatedTransactions{
		Transactions: transactions,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}, nil
}