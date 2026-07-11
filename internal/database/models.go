package database

import "database/sql"

type Models struct {
	Users    UserModel
	Accounts AccountModel
	Tokens   TokenModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:    UserModel{DB: db},
		Accounts: AccountModel{DB: db},
		Tokens:   TokenModel{DB: db},
	}
}