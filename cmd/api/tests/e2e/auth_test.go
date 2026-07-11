package e2e_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"BANKAPI/internal/cache"
	"BANKAPI/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type E2EAuthSuite struct {
	suite.Suite
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	db             *sql.DB
	router         *gin.Engine
	app            *testApp
}

type testApp struct {
	db     *database.Models
	cache  *cache.Cache
	secret string
}

func setupPostgres(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(3 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=testdb sslmode=disable", host, port.Port())

	return container, dsn, nil
}

func setupRedis(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")
	addr := fmt.Sprintf("%s:%s", host, port.Port())

	return container, addr, nil
}

func (s *E2EAuthSuite) SetupSuite() {
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	pgContainer, dsn, err := setupPostgres(ctx)
	s.Require().NoError(err)
	s.pgContainer = pgContainer

	redisContainer, redisAddr, err := setupRedis(ctx)
	s.Require().NoError(err)
	s.redisContainer = redisContainer

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
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id         SERIAL PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token      TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
		CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
	`)
	s.Require().NoError(err)

	models := &database.Models{
		Users:  database.UserModel{DB: db},
		Tokens: database.TokenModel{DB: db},
	}

	c := cache.NewCache(redisAddr)
	s.router = setupTestRouter(models, c, "test-secret-key")
}

func (s *E2EAuthSuite) TearDownSuite() {
	s.db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.pgContainer.Terminate(ctx)
	s.redisContainer.Terminate(ctx)
}

func (s *E2EAuthSuite) SetupTest() {
	s.db.Exec("DELETE FROM refresh_tokens")
	s.db.Exec("DELETE FROM users")
}

func (s *E2EAuthSuite) TestRegister_Success() {
	body := map[string]string{
		"firstName": "Fernando",
		"lastName":  "Agamez",
		"email":     "fernando@gmail.com",
		"password":  "12345678",
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/users/register", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(s.T(), "user registered successfully", resp["message"])
}

func (s *E2EAuthSuite) TestRegister_DuplicateEmail() {
	body := map[string]string{
		"firstName": "Fernando",
		"lastName":  "Agamez",
		"email":     "fernando@gmail.com",
		"password":  "12345678",
	}
	data, _ := json.Marshal(body)

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/v1/users/register", bytes.NewBuffer(data))
	req1.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/v1/users/register", bytes.NewBuffer(data))
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w2, req2)

	assert.Equal(s.T(), http.StatusConflict, w2.Code)
}

func (s *E2EAuthSuite) TestLogin_Success() {
	s.registerUser("fernando@gmail.com", "12345678")

	body := map[string]string{
		"email":    "fernando@gmail.com",
		"password": "12345678",
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/users/login", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(s.T(), resp["accessToken"])
	assert.NotEmpty(s.T(), resp["refreshToken"])
}

func (s *E2EAuthSuite) TestLogin_WrongPassword() {
	s.registerUser("fernando@gmail.com", "12345678")

	body := map[string]string{
		"email":    "fernando@gmail.com",
		"password": "wrongpassword",
	}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/users/login", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *E2EAuthSuite) TestLogout_Success() {
	s.registerUser("fernando@gmail.com", "12345678")
	_, refreshToken := s.loginUser("fernando@gmail.com", "12345678")

	body := map[string]string{"refreshToken": refreshToken}
	data, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/auth/logout", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *E2EAuthSuite) TestRefresh_AfterLogout() {
	s.registerUser("fernando@gmail.com", "12345678")
	_, refreshToken := s.loginUser("fernando@gmail.com", "12345678")

	body := map[string]string{"refreshToken": refreshToken}
	data, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/auth/logout", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/v1/auth/refresh", bytes.NewBuffer(data))
	req2.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w2, req2)

	assert.Equal(s.T(), http.StatusUnauthorized, w2.Code)
}

func (s *E2EAuthSuite) registerUser(email, password string) {
	body := map[string]string{
		"firstName": "Fernando",
		"lastName":  "Agamez",
		"email":     email,
		"password":  password,
	}
	data, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/users/register", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
}

func (s *E2EAuthSuite) loginUser(email, password string) (string, string) {
	body := map[string]string{"email": email, "password": password}
	data, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/users/login", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["accessToken"].(string), resp["refreshToken"].(string)
}

func TestE2EAuthSuite(t *testing.T) {
	suite.Run(t, new(E2EAuthSuite))
}
