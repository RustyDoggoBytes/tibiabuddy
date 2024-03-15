package main

import (
	"context"
	"database/sql"
	"rustydoggobytes/tibiabuddy/sqlc"

	_ "embed"

	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql
var ddl string

type UserSession = string

type AuthService struct {
	Ctx context.Context
	Db  *sqlc.Queries
}

func NewAuthService(db *sql.DB) *AuthService {
	queries := sqlc.New(db)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		panic(err)
	}
	return &AuthService{
		Ctx: ctx,
		Db:  queries,
	}
}

func (a AuthService) signUp(email, password string) (*sqlc.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := a.Db.CreateUser(a.Ctx, sqlc.CreateUserParams{
		Email:          email,
		HashedPassword: hashedPassword,
	})

	return &user, err
}

func (a AuthService) signIn(email, password string) (*sqlc.User, error) {
	user, err := a.Db.GetUser(a.Ctx, email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
	if err != nil {
		return nil, err
	}

	return &user, nil
}
