package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"BANKAPI/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type AccountsIntegrationSuite struct {
	suite.Suite
	container    testcontainers.Container
	db           *sql.DB
	userModel    database.UserModel
	accountModel database.AccountModel
}

func (s *AccountsIntegrationSuite) SetupSuite() {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.container = container

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=testdb sslmode=disable", host, port.Port())
	db, err := sql.Open("postgres", dsn)
	s.Require().NoError(err)
	s.db = db

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			first_name VARCHAR(100) NOT NULL,
			last_name  VARCHAR(100) NOT NULL,
			email      VARCHAR(255) NOT NULL UNIQUE,
			password   TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS accounts (
			id         SERIAL PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			balance    BIGINT NOT NULL DEFAULT 0,
			currency   VARCHAR(3) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id               SERIAL PRIMARY KEY,
			from_account_id  INTEGER REFERENCES accounts(id),
			to_account_id    INTEGER REFERENCES accounts(id),
			amount           BIGINT NOT NULL,
			from_currency    VARCHAR(3) NOT NULL DEFAULT 'COP',
			to_currency      VARCHAR(3) NOT NULL DEFAULT 'COP',
			exchange_rate    NUMERIC(19,6) NOT NULL DEFAULT 1,
			converted_amount BIGINT NOT NULL DEFAULT 0,
			created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	s.Require().NoError(err)

	s.userModel = database.UserModel{DB: db}
	s.accountModel = database.AccountModel{DB: db}
}

func (s *AccountsIntegrationSuite) TearDownSuite() {
	s.db.Close()
	s.container.Terminate(context.Background())
}

func (s *AccountsIntegrationSuite) SetupTest() {
	s.db.Exec("DELETE FROM transactions")
	s.db.Exec("DELETE FROM accounts")
	s.db.Exec("DELETE FROM users")
}

func (s *AccountsIntegrationSuite) createTestUser() database.User {
	user, err := s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "12345678")
	s.Require().NoError(err)
	return user
}

func (s *AccountsIntegrationSuite) TestCreateAccount_Success() {
	user := s.createTestUser()

	account, err := s.accountModel.CreateAccount(user.Id, "COP")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), user.Id, account.UserId)
	assert.Equal(s.T(), "COP", account.Currency)
	assert.Equal(s.T(), int64(0), account.Balance)
}

func (s *AccountsIntegrationSuite) TestDeposit_Success() {
	user := s.createTestUser()
	account, _ := s.accountModel.CreateAccount(user.Id, "COP")

	updated, err := s.accountModel.Deposit(account.Id, 500000)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(500000), updated.Balance)
}

func (s *AccountsIntegrationSuite) TestDeposit_MultipleDeposits() {
	user := s.createTestUser()
	account, _ := s.accountModel.CreateAccount(user.Id, "COP")

	s.accountModel.Deposit(account.Id, 300000)
	updated, err := s.accountModel.Deposit(account.Id, 200000)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(500000), updated.Balance)
}

func (s *AccountsIntegrationSuite) TestTransfer_Success() {
	user, _ := s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "12345678")
	user2, _ := s.userModel.CreateUser("Usuario", "Dos", "usuario2@gmail.com", "12345678")

	from, _ := s.accountModel.CreateAccount(user.Id, "COP")
	to, _ := s.accountModel.CreateAccount(user2.Id, "COP")

	s.accountModel.Deposit(from.Id, 1000000)

	err := s.accountModel.Transfer(from.Id, to.Id, 300000, 300000, "COP", "COP", 1.0)

	assert.NoError(s.T(), err)

	fromUpdated, _ := s.accountModel.GetAccountById(from.Id)
	toUpdated, _ := s.accountModel.GetAccountById(to.Id)

	assert.Equal(s.T(), int64(700000), fromUpdated.Balance)
	assert.Equal(s.T(), int64(300000), toUpdated.Balance)
}

func (s *AccountsIntegrationSuite) TestTransfer_InsufficientFunds() {
	user, _ := s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "12345678")
	user2, _ := s.userModel.CreateUser("Usuario", "Dos", "usuario2@gmail.com", "12345678")

	from, _ := s.accountModel.CreateAccount(user.Id, "COP")
	to, _ := s.accountModel.CreateAccount(user2.Id, "COP")

	s.accountModel.Deposit(from.Id, 100000)

	err := s.accountModel.Transfer(from.Id, to.Id, 500000, 500000, "COP", "COP", 1.0)

	assert.ErrorIs(s.T(), err, database.ErrInsufficientFunds)

	fromUpdated, _ := s.accountModel.GetAccountById(from.Id)
	assert.Equal(s.T(), int64(100000), fromUpdated.Balance)
}

func (s *AccountsIntegrationSuite) TestWithdraw_Success() {
	user := s.createTestUser()
	account, _ := s.accountModel.CreateAccount(user.Id, "COP")
	s.accountModel.Deposit(account.Id, 500000)

	updated, err := s.accountModel.Deposit(account.Id, -200000)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(300000), updated.Balance)
}

func TestAccountsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AccountsIntegrationSuite))
}