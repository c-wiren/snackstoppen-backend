package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/c-wiren/snackstoppen-backend/auth"
	"github.com/c-wiren/snackstoppen-backend/graph/generated"
	"github.com/c-wiren/snackstoppen-backend/graph/model"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/crypto/bcrypt"
)

func (r *mutationResolver) CreateReview(ctx context.Context, review model.NewReview) (*model.Review, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, gqlerror.Errorf("Must be logged in")
	}

	// Insert review into DB
	rows, err := r.DB.Query(ctx, `INSERT INTO reviews (chips_id, rating, review, user_id)
	VALUES ($1, $2, $3, $4)
	RETURNING id, review, rating, created`, review.Chips, review.Rating, review.Review, user.ID)
	if !rows.Next() || err != nil {
		fmt.Println(err)
		return nil, gqlerror.Errorf("Insert failed")
	}
	defer rows.Close()
	var newReview model.Review
	newReview.User = new(model.User)
	err = rows.Scan(&newReview.ID, &newReview.Review, &newReview.Rating, &newReview.Created)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("db scan review failed"))
	}

	return &newReview, nil
}

func (r *mutationResolver) CreateChip(ctx context.Context, chip model.NewChip) (*bool, error) {
	// Insert chip into DB
	commandTag, err := r.DB.Exec(ctx, `INSERT INTO chips (name,category,subcategory,slug,image,ingredients,brand_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		chip.Name, chip.Category, chip.Subcategory, chip.Slug, chip.Image, chip.Ingredients, chip.Brand)
	if commandTag.RowsAffected() != 1 || err != nil {
		return nil, gqlerror.Errorf("Incorrect email address")
	}

	return nil, nil
}

func (r *mutationResolver) CreateUser(ctx context.Context, user model.NewUser) (*model.LoginResponse, error) {
	// Parse JWT
	emailToken, err := jwt.Parse(user.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("secret"), nil
	})
	if err != nil || !emailToken.Valid {
		fmt.Println(err)
		return nil, gqlerror.Errorf("The code is expired")
	}
	claims, ok := emailToken.Claims.(jwt.MapClaims)
	if !ok {
		panic(fmt.Errorf("token claims error"))
	}
	email, _ := claims["email"].(string)
	hash, _ := claims["code"].(string)

	// Check if confirmed email is the same
	if email != user.Email {
		return nil, gqlerror.Errorf("Incorrect email address")
	}

	// Check if entered code is correct
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Code))
	if err != nil {
		return nil, gqlerror.Errorf("Incorrect code")
	}

	// Create password hash
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 10)

	// TODO: Insert all fields
	// Insert user into DB
	rows, err := r.DB.Query(ctx, `INSERT INTO users (username, email, password)
	VALUES ($1, $2, $3)
	RETURNING id, role`, user.Name, user.Email, string(passwordHash))
	if !rows.Next() || err != nil {
		return nil, gqlerror.Errorf("Could not create user")
	}
	defer rows.Close()
	completeUser := model.CompleteUser{}
	err = rows.Scan(&completeUser.Username, &completeUser.Password, &completeUser.ID, &completeUser.Email, &completeUser.Firstname, &completeUser.Lastname, &completeUser.Role, &completeUser.Image, &completeUser.Created, &completeUser.Logout)
	if err != nil {
		panic(fmt.Errorf("db row scan error"))
	}

	return auth.CreateLoginResponse(
		completeUser,
		auth.CreateAccessToken(&completeUser),
		auth.CreateRefreshToken(&completeUser)), nil
}

func (r *mutationResolver) ValidateEmail(ctx context.Context, email string) (string, error) {
	rows, err := r.DB.Query(ctx, "SELECT 1 FROM users WHERE email=$1", email)
	if err != nil {
		panic(fmt.Errorf("db query error"))
	}
	defer rows.Close()
	if rows.Next() {
		return "", gqlerror.Errorf("Email already exists")
	}
	// Generate random code
	nBig, _ := rand.Int(rand.Reader, big.NewInt(10000))
	code := fmt.Sprintf("%04d", nBig)

	// Temporarily print code
	fmt.Println("Verification code:", code)

	// Send email with code
	message := r.Mailgun.NewMessage("Snackstoppen <noreply@sandbox797116ba525741268d6b789b03c15c5b.mailgun.org>", "Verifieringskod från Snackstoppen", "", email)
	message.SetHtml(fmt.Sprintf("<p><b>%s</b> är din verifieringskod för Snackstoppen.</p><p>Hälsningar,<br>Snackstoppen</p>", code))
	_, _, err = r.Mailgun.Send(message)
	if err != nil {
		fmt.Println("Could not send email:", err)
		return "", gqlerror.Errorf("Internal server error")
	}

	// Create hash from code
	hash, _ := bcrypt.GenerateFromPassword([]byte(code), 10)

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"code":  string(hash),
		"exp":   time.Now().Add(time.Minute * 10).Unix(),
		"iat":   time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte("secret"))
	return tokenString, nil
}

func (r *mutationResolver) Login(ctx context.Context, email string, password string) (*model.LoginResponse, error) {
	// Get user from DB
	rows, err := r.DB.Query(ctx, `SELECT username, password, id, email, firstname, lastname, role, image, created, logout FROM users WHERE email=$1`, email)
	if err != nil {
		panic(fmt.Errorf("db query error"))
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, gqlerror.Errorf("User does not exist")
	}
	completeUser := model.CompleteUser{}
	err = rows.Scan(&completeUser.Username, &completeUser.Password, &completeUser.ID, &completeUser.Email, &completeUser.Firstname, &completeUser.Lastname, &completeUser.Role, &completeUser.Image, &completeUser.Created, &completeUser.Logout)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("db row scan error"))
	}

	// Check if password is correct
	err = bcrypt.CompareHashAndPassword([]byte(completeUser.Password), []byte(password))
	if err != nil {
		return nil, gqlerror.Errorf("Incorrect password")
	}

	return auth.CreateLoginResponse(
		completeUser,
		auth.CreateAccessToken(&completeUser),
		auth.CreateRefreshToken(&completeUser)), nil
}

func (r *mutationResolver) Refresh(ctx context.Context, token string) (*model.LoginResponse, error) {
	// Parse JWT
	refreshToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("secret"), nil
	})
	if err != nil || !refreshToken.Valid {
		return nil, gqlerror.Errorf("The session has expired")
	}
	claims, ok := refreshToken.Claims.(jwt.MapClaims)
	if !ok {
		panic(fmt.Errorf("token claims error"))
	}

	// Get token data
	rawId, _ := claims["id"].(float64)
	id := int(rawId)
	rawLogout, _ := claims["logout"].(string)
	logout, _ := time.Parse(time.RFC3339, rawLogout)

	// Get user from DB
	rows, err := r.DB.Query(ctx, `SELECT username, password, id, email, firstname, lastname, role, image, created, logout FROM users WHERE id=$1`, id)
	if err != nil {
		panic(fmt.Errorf("db query error"))
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, gqlerror.Errorf("User does not exist")
	}
	completeUser := model.CompleteUser{}
	err = rows.Scan(&completeUser.Username, &completeUser.Password, &completeUser.ID, &completeUser.Email, &completeUser.Firstname, &completeUser.Lastname, &completeUser.Role, &completeUser.Image, &completeUser.Created, &completeUser.Logout)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("db row scan error"))
	}

	// Check if all devices has been logged out
	if logout != completeUser.Logout {
		return nil, gqlerror.Errorf("You have been logged out")
	}

	return auth.CreateLoginResponse(
		completeUser,
		auth.CreateAccessToken(&completeUser),
		nil), nil
}

func (r *mutationResolver) LogoutAll(ctx context.Context) (*bool, error) {
	// TODO: Get user ID from token
	user := auth.ForContext(ctx)

	// Update logout date in DB
	commandTag, err := r.DB.Exec(ctx, `UPDATE users
	SET logout = NOW()
	WHERE id=$1`, user.ID)
	if commandTag.RowsAffected() != 1 || err != nil {
		panic(fmt.Errorf("db not updated with logout"))
	}
	return nil, nil
}

func (r *queryResolver) Chip(ctx context.Context, brand string, slug string) (*model.Chip, error) {
	rows, err := r.DB.Query(ctx, `SELECT chips.name,category,subcategory,chips.slug,chips.image,ingredients,chips.id,brands.id,brands.image,brands.count,brands.name
	FROM chips INNER JOIN brands ON chips.brand_id=brands.id WHERE chips.brand_id=$1 AND chips.slug=$2 LIMIT 1`, brand, slug)
	if err != nil {
		fmt.Print(err)
	}
	defer rows.Close()
	if rows.Next() {
		chip := &model.Chip{}
		brand := &model.Brand{}
		chip.Brand = brand
		err := rows.Scan(&chip.Name, &chip.Category, &chip.Subcategory, &chip.Slug, &chip.Image, &chip.Ingredients, &chip.ID, &brand.ID, &brand.Image, &brand.Count, &brand.Name)
		if err != nil {
			fmt.Print(err)
		}
		return chip, nil
	}
	return nil, nil
}

func (r *queryResolver) Chips(ctx context.Context, brand *string, orderBy *model.ChipSortByInput, limit *int, offset *int) ([]*model.Chip, error) {
	argCount := 0
	var args []interface{}
	q := `
	SELECT chips.name,category,subcategory,chips.slug,chips.image,ingredients,chips.id,brands.id,brands.image,brands.count,brands.name
	FROM chips INNER JOIN brands ON chips.brand_id=brands.id`
	if brand != nil {
		argCount++
		q += fmt.Sprint(" WHERE chips.brand_id=$", argCount)
		args = append(args, brand)
	}
	if orderBy != nil && *orderBy == model.ChipSortByInputNameAsc {
		q += " ORDER BY chips.name"
	}
	if limit != nil {
		argCount++
		q += fmt.Sprint(" LIMIT $", argCount)
		args = append(args, limit)
	}
	if offset != nil {
		argCount++
		q += fmt.Sprint(" OFFSET $", argCount)
		args = append(args, offset)
	}
	var chips []*model.Chip
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		fmt.Print(err)
	}
	for rows.Next() {
		chip := &model.Chip{}
		brand := &model.Brand{}
		chip.Brand = brand
		err := rows.Scan(&chip.Name, &chip.Category, &chip.Subcategory, &chip.Slug, &chip.Image, &chip.Ingredients, &chip.ID, &brand.ID, &brand.Image, &brand.Count, &brand.Name)
		if err != nil {
			fmt.Print(err)
		}
		chips = append(chips, chip)
	}

	return chips, nil
}

func (r *queryResolver) Brand(ctx context.Context, id string) (*model.Brand, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, image, name, count FROM brands WHERE id=$1 LIMIT 1`, id)
	if err != nil {
		fmt.Print(err)
	}
	defer rows.Close()
	if rows.Next() {
		brand := &model.Brand{}
		err := rows.Scan(&brand.ID, &brand.Image, &brand.Name, &brand.Count)
		if err != nil {
			fmt.Print(err)
		}
		return brand, nil
	}
	return nil, nil
}

func (r *queryResolver) Brands(ctx context.Context, orderBy *model.BrandSortByInput) ([]*model.Brand, error) {
	var brands []*model.Brand
	q := "SELECT id, image, name, count FROM brands"
	if orderBy != nil && *orderBy == model.BrandSortByInputNameAsc {
		q += " ORDER BY name"
	}
	rows, err := r.DB.Query(ctx, q)
	if err != nil {
		fmt.Print(err)
	}
	for rows.Next() {
		brand := &model.Brand{}
		err := rows.Scan(&brand.ID, &brand.Image, &brand.Name, &brand.Count)
		if err != nil {
			fmt.Print(err)
		}
		brands = append(brands, brand)
	}
	return brands, nil
}

func (r *queryResolver) Reviews(ctx context.Context, chips *int, author *int, limit *int, offset *int, orderBy *model.ReviewSortByInput) ([]*model.Review, error) {
	if chips != nil && author != nil {
		return nil, gqlerror.Errorf("Select by either chip or user")
	}
	if chips != nil {
		argCount := 0
		var args []interface{}
		q := `
		SELECT reviews.id, reviews.rating, reviews.review, reviews.created, reviews.edited, users.username, users.firstname, users.lastname, users.image
		FROM reviews INNER JOIN users ON reviews.user_id=users.id`
		argCount++
		q += fmt.Sprint(" WHERE reviews.chips=$", argCount)
		args = append(args, chips)
		if orderBy != nil && *orderBy == model.ReviewSortByInputDateDesc {
			q += " ORDER BY reviews.created DESC"
		}
		if limit != nil {
			argCount++
			q += fmt.Sprint(" LIMIT $", argCount)
			args = append(args, limit)
		}
		if offset != nil {
			argCount++
			q += fmt.Sprint(" OFFSET $", argCount)
			args = append(args, offset)
		}
		var reviews []*model.Review
		rows, err := r.DB.Query(ctx, q, args...)
		if err != nil {
			fmt.Print(err)
		}
		for rows.Next() {
			review := &model.Review{}
			user := &model.User{}
			review.User = user
			err := rows.Scan(&review.ID, &review.Rating, &review.Review, &review.Created, &review.Edited, &user.Username, &user.Firstname, &user.Lastname, &user.Image)
			if err != nil {
				fmt.Print(err)
			}
			reviews = append(reviews, review)
		}

		return reviews, nil
	}
	if author != nil {
		argCount := 0
		var args []interface{}
		q := `
		SELECT reviews.id, reviews.rating, reviews.review, reviews.created, reviews.edited, chips.name, chips.slug, brands.id, brands.name
		FROM reviews
		INNER JOIN chips ON reviews.chips_id=chips.id
		INNER JOIN brands ON chips.brand_id=brands.id`
		argCount++
		q += fmt.Sprint(" WHERE reviews.user_id=$", argCount)
		args = append(args, author)
		if orderBy != nil && *orderBy == model.ReviewSortByInputDateDesc {
			q += " ORDER BY reviews.created DESC"
		}
		if limit != nil {
			argCount++
			q += fmt.Sprint(" LIMIT $", argCount)
			args = append(args, limit)
		}
		if offset != nil {
			argCount++
			q += fmt.Sprint(" OFFSET $", argCount)
			args = append(args, offset)
		}
		var reviews []*model.Review
		rows, err := r.DB.Query(ctx, q, args...)
		if err != nil {
			fmt.Print(err)
		}
		for rows.Next() {
			review := &model.Review{}
			chips := &model.Chip{}
			brand := &model.Brand{}
			chips.Brand = brand
			review.Chips = chips
			err := rows.Scan(&review.ID, &review.Rating, &review.Review, &review.Created, &review.Edited, &chips.Name, &chips.Slug, &brand.ID, &brand.Name)
			if err != nil {
				fmt.Print(err)
			}
			reviews = append(reviews, review)
		}

		return reviews, nil
	}
	return nil, gqlerror.Errorf("Select by either chip or author")
}

func (r *queryResolver) User(ctx context.Context) (*model.User, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
