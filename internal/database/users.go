package database

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

type UserModel struct {
	DB *sql.DB
}

type User struct {
	Id        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"-"`
}

func (m UserModel) CreateUser(firstName, lastName, email, password string) (User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	query := `
		INSERT INTO users (first_name, last_name, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id, first_name, last_name, email
	`

	var user User
	err = m.DB.QueryRow(query,
		firstName,
		lastName,
		email,
		string(hashedPassword),
	).Scan(
		&user.Id,
		&user.FirstName,
		&user.LastName,
		&user.Email,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (m UserModel) GetUserByEmail(email string) (User, error) {
	query := `
		SELECT id, first_name, last_name, email, password
		FROM users
		WHERE email = $1
	`

	var user User
	err := m.DB.QueryRow(query, email).Scan(
		&user.Id,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (m UserModel) GetUserById(id int) (User, error) {
	query := `
		SELECT id, first_name, last_name, email, password
		FROM users
		WHERE id = $1
	`

	var user User
	err := m.DB.QueryRow(query, id).Scan(
		&user.Id,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (m UserModel) CheckPassword(user User, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}