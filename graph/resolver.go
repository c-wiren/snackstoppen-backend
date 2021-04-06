package graph

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/mailgun/mailgun-go/v4"
)

//go:generate go run github.com/99designs/gqlgen

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB      *pgxpool.Pool
	Mailgun *mailgun.MailgunImpl
}
