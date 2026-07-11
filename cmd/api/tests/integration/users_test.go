package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"BANKAPI/internal/database"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type UsersIntegrationSuite struct {
	suite.Suite
	container testcontainers.Container
	db        *sql.DB
	userModel database.UserModel
}

func (s *UsersIntegrationSuite) SetupSuite() {
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
		)
	`)
	s.Require().NoError(err)

	s.userModel = database.UserModel{DB: db}
}

func (s *UsersIntegrationSuite) TearDownSuite() {
	s.db.Close()
	s.container.Terminate(context.Background())
}

func (s *UsersIntegrationSuite) SetupTest() {
	s.db.Exec("DELETE FROM users")
}

func (s *UsersIntegrationSuite) TestCreateUser_Success() {
	user, err := s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "hashedpassword")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "Fernando", user.FirstName)
	assert.Equal(s.T(), "fernando@gmail.com", user.Email)
	assert.NotZero(s.T(), user.Id)
}

func (s *UsersIntegrationSuite) TestCreateUser_DuplicateEmail() {
	s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "hash1")
	_, err := s.userModel.CreateUser("Otro", "Usuario", "fernando@gmail.com", "hash2")

	assert.Error(s.T(), err)
}

func (s *UsersIntegrationSuite) TestGetUserByEmail_Success() {
	s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "hashedpassword")

	user, err := s.userModel.GetUserByEmail("fernando@gmail.com")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "Fernando", user.FirstName)
}

func (s *UsersIntegrationSuite) TestGetUserByEmail_NotFound() {
	_, err := s.userModel.GetUserByEmail("noexiste@gmail.com")

	assert.Error(s.T(), err)
}

func (s *UsersIntegrationSuite) TestGetUserById_Success() {
	created, _ := s.userModel.CreateUser("Fernando", "Agamez", "fernando@gmail.com", "hash")

	user, err := s.userModel.GetUserById(created.Id)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), created.Id, user.Id)
}

func TestUsersIntegrationSuite(t *testing.T) {
	suite.Run(t, new(UsersIntegrationSuite))
}