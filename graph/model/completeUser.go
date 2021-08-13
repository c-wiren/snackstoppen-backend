package model

import "time"

type CompleteUser struct {
	ID        int
	Username  *string
	Password  string
	Email     string
	Firstname *string
	Lastname  *string
	Role      *string
	Image     *string
	Created   time.Time
	Logout    time.Time
}
