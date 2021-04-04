// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type Brand struct {
	ID    string  `json:"id"`
	Count int     `json:"count"`
	Image *string `json:"image"`
	Name  string  `json:"name"`
}

type LoginResponse struct {
	User    *User   `json:"user"`
	Token   string  `json:"token"`
	Refresh *string `json:"refresh"`
}

type NewChip struct {
	Brand       string  `json:"brand"`
	Category    string  `json:"category"`
	Image       *string `json:"image"`
	Ingredients *string `json:"ingredients"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Subcategory *string `json:"subcategory"`
}

type NewReview struct {
	Chips  int     `json:"chips"`
	Rating int     `json:"rating"`
	Review *string `json:"review"`
}

type NewUser struct {
	Name     string  `json:"name"`
	Image    *string `json:"image"`
	Password string  `json:"password"`
	Email    string  `json:"email"`
	Code     string  `json:"code"`
	Token    string  `json:"token"`
}

type Review struct {
	ID      int        `json:"id"`
	Chips   *Chip      `json:"chips"`
	Rating  int        `json:"rating"`
	Review  *string    `json:"review"`
	User    *User      `json:"user"`
	Created time.Time  `json:"created"`
	Edited  *time.Time `json:"edited"`
}

type SearchResponse struct {
	User  *User   `json:"user"`
	Chips []*Chip `json:"chips"`
}

type User struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Firstname *string    `json:"firstname"`
	Lastname  *string    `json:"lastname"`
	Created   *time.Time `json:"created"`
	Image     *string    `json:"image"`
}

type BrandSortByInput string

const (
	BrandSortByInputNameAsc BrandSortByInput = "NAME_ASC"
)

var AllBrandSortByInput = []BrandSortByInput{
	BrandSortByInputNameAsc,
}

func (e BrandSortByInput) IsValid() bool {
	switch e {
	case BrandSortByInputNameAsc:
		return true
	}
	return false
}

func (e BrandSortByInput) String() string {
	return string(e)
}

func (e *BrandSortByInput) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = BrandSortByInput(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid BrandSortByInput", str)
	}
	return nil
}

func (e BrandSortByInput) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type ChipSortByInput string

const (
	ChipSortByInputNameAsc ChipSortByInput = "NAME_ASC"
)

var AllChipSortByInput = []ChipSortByInput{
	ChipSortByInputNameAsc,
}

func (e ChipSortByInput) IsValid() bool {
	switch e {
	case ChipSortByInputNameAsc:
		return true
	}
	return false
}

func (e ChipSortByInput) String() string {
	return string(e)
}

func (e *ChipSortByInput) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ChipSortByInput(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ChipSortByInput", str)
	}
	return nil
}

func (e ChipSortByInput) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type ReviewSortByInput string

const (
	ReviewSortByInputDateDesc ReviewSortByInput = "DATE_DESC"
)

var AllReviewSortByInput = []ReviewSortByInput{
	ReviewSortByInputDateDesc,
}

func (e ReviewSortByInput) IsValid() bool {
	switch e {
	case ReviewSortByInputDateDesc:
		return true
	}
	return false
}

func (e ReviewSortByInput) String() string {
	return string(e)
}

func (e *ReviewSortByInput) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ReviewSortByInput(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ReviewSortByInput", str)
	}
	return nil
}

func (e ReviewSortByInput) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
