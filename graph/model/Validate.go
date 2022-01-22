package model

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

func (r NewReview) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Rating, validation.Min(1), validation.Max(10)),
		validation.Field(&r.Review, validation.Length(0, 10000)),
	)
}

func (c NewChip) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Ingredients, validation.Length(0, 2000)),
		validation.Field(&c.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&c.Slug, validation.Match(regexp.MustCompile("^[a-z0-9]+(-[a-z0-9]+)*"))),
	)
}

func (u NewUser) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Email, is.EmailFormat),
		validation.Field(&u.Username, validation.Match(regexp.MustCompile("^[a-zA-Z0-9-_]{2,20}$"))),
		validation.Field(&u.Firstname, validation.Length(0, 35)),
		validation.Field(&u.Lastname, validation.Length(0, 35)),
		validation.Field(&u.Password, validation.Required, validation.Length(8, 128)),
	)
}
