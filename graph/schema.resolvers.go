package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"image"
	"image/png"
	"math/big"
	"strings"
	"time"

	"github.com/c-wiren/snackstoppen-backend/auth"
	"github.com/c-wiren/snackstoppen-backend/graph/generated"
	"github.com/c-wiren/snackstoppen-backend/graph/model"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go/v7"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/crypto/bcrypt"
)

func (r *mutationResolver) CreateReview(ctx context.Context, review model.NewReview) (*model.Review, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return nil, &gqlerror.Error{Message: "Must be logged in", Extensions: map[string]interface{}{"code": "UNAUTHORIZED"}}
	}

	// Insert review into DB
	rows, err := r.DB.Query(ctx, `INSERT INTO reviews (chips_id, rating, review, user_id)
	VALUES ($1, $2, $3, $4)
	RETURNING id, review, rating, created`, review.Chips, review.Rating, review.Review, user.ID)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("insert review failed"))
	}
	if !rows.Next() {
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
	user := auth.ForContext(ctx)
	if user == nil || user.Role != "admin" {
		return nil, &gqlerror.Error{Message: "Must be admin", Extensions: map[string]interface{}{"code": "FORBIDDEN"}}
	}
	var imageURL *string
	var originalImage image.Image
	if chip.Image != nil {
		var err error
		originalImage, err = png.Decode(chip.Image.File)
		if err != nil {
			return nil, &gqlerror.Error{Message: "Invalid image", Extensions: map[string]interface{}{"code": "INVALID_IMAGE"}}
		}
		url := chip.Brand + "-" + chip.Slug + ".png"
		imageURL = &url
	}
	// Insert chip into DB
	commandTag, err := r.DB.Exec(ctx, `INSERT INTO chips (name,category,subcategory,slug,image,ingredients,brand_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		chip.Name, chip.Category, chip.Subcategory, chip.Slug, imageURL, chip.Ingredients, chip.Brand)
	if commandTag.RowsAffected() != 1 || err != nil {
		return nil, gqlerror.Errorf("Could not create chip")
	}

	if chip.Image != nil {
		buff := bytes.NewBuffer(nil)
		png.Encode(buff, originalImage)
		_, err = r.S3.PutObject(ctx, "snackstoppen", "original/snacks/"+*imageURL, buff, int64(buff.Len()), minio.PutObjectOptions{ContentType: "image/png"})
		if err != nil {
			fmt.Println(err)
		}
		resizedImage := imaging.Fit(originalImage, 60, 60, imaging.Box)
		png.Encode(buff, resizedImage)
		_, err = r.S3.PutObject(ctx, "snackstoppen", "sm/snacks/"+*imageURL, buff, int64(buff.Len()), minio.PutObjectOptions{ContentType: "image/png"})
		if err != nil {
			fmt.Println(err)
		}
		resizedImage = imaging.Fit(originalImage, 224, 224, imaging.Box)
		png.Encode(buff, resizedImage)
		_, err = r.S3.PutObject(ctx, "snackstoppen", "md/snacks/"+*imageURL, buff, int64(buff.Len()), minio.PutObjectOptions{ContentType: "image/png"})
		if err != nil {
			fmt.Println(err)
		}
		resizedImage = imaging.Fit(originalImage, 640, 640, imaging.Box)
		png.Encode(buff, resizedImage)
		_, err = r.S3.PutObject(ctx, "snackstoppen", "lg/snacks/"+*imageURL, buff, int64(buff.Len()), minio.PutObjectOptions{ContentType: "image/png"})
		if err != nil {
			fmt.Println(err)
			// Remove chip from db
			_, err := r.DB.Exec(ctx, `DELETE FROM chips
			WHERE slug=$1 AND brand_id=$2`,
				chip.Slug, chip.Brand)
			fmt.Println(err)
			panic(fmt.Errorf("create chip s3 upload error"))
		}
	}
	return nil, nil
}

func (r *mutationResolver) CreateUser(ctx context.Context, user model.NewUser) (*model.LoginResponse, error) {
	// Parse JWT
	emailToken, err := jwt.Parse(user.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(auth.Secret), nil
	})
	if err != nil || !emailToken.Valid {
		return nil, &gqlerror.Error{Message: "The code is expired", Extensions: map[string]interface{}{"code": "EXPIRED_EMAIL_VERIFICATION"}}
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
		return nil, &gqlerror.Error{Message: "Incorrect code", Extensions: map[string]interface{}{"code": "INVALID_EMAIL_VERIFICATION"}}
	}

	// Create password hash
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 10)

	// Insert user into DB
	rows, err := r.DB.Query(ctx, `INSERT INTO users (username, email, password, firstname, lastname)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING username, id, email, firstname, lastname, role, image, created, logout`, user.Username, user.Email, string(passwordHash), user.Firstname, user.Lastname)
	if !rows.Next() || err != nil {
		return nil, gqlerror.Errorf("Could not create user")
	}
	defer rows.Close()
	completeUser := model.CompleteUser{}
	err = rows.Scan(&completeUser.Username, &completeUser.ID, &completeUser.Email, &completeUser.Firstname, &completeUser.Lastname, &completeUser.Role, &completeUser.Image, &completeUser.Created, &completeUser.Logout)
	if err != nil {
		panic(fmt.Errorf("db row scan error"))
	}

	return auth.CreateLoginResponse(
		completeUser,
		true), nil
}

func (r *mutationResolver) ValidateEmail(ctx context.Context, email string) (string, error) {
	// Check if email exists
	rows, err := r.DB.Query(ctx, "SELECT 1 FROM users WHERE email=$1", email)
	if err != nil {
		panic(fmt.Errorf("db query error"))
	}
	defer rows.Close()
	if rows.Next() {
		return "", &gqlerror.Error{Message: "Email already exists", Extensions: map[string]interface{}{"code": "EXISTING_EMAIL"}}
	}
	// Generate random code
	nBig, _ := rand.Int(rand.Reader, big.NewInt(10000))
	code := fmt.Sprintf("%04d", nBig)

	// Send email with code
	message := r.Mailgun.NewMessage("Snackstoppen <noreply@snackstoppen.se>", "Verifieringskod från Snackstoppen", "", email)
	message.SetHtml(fmt.Sprintf("<p><b>%s</b> är din verifieringskod för Snackstoppen.</p><p>Hälsningar,<br>Snackstoppen</p>", code))
	_, _, err = r.Mailgun.Send(ctx, message)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("could not send mailgun email"))
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
	tokenString, _ := token.SignedString([]byte(auth.Secret))
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
		return nil, &gqlerror.Error{Message: "Incorrect credentials", Extensions: map[string]interface{}{"code": "USER_INPUT_ERROR"}}
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
		return nil, &gqlerror.Error{Message: "Incorrect credentials", Extensions: map[string]interface{}{"code": "USER_INPUT_ERROR"}}
	}

	return auth.CreateLoginResponse(
		completeUser,
		true), nil
}

func (r *mutationResolver) Refresh(ctx context.Context, token string) (*model.LoginResponse, error) {
	// Parse JWT
	refreshToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(auth.Secret), nil
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
		return nil, &gqlerror.Error{Message: "All devices has been logged out", Extensions: map[string]interface{}{"code": "AUTHENTICATION_ERROR"}}
	}

	return auth.CreateLoginResponse(
		completeUser,
		false), nil
}

func (r *mutationResolver) LogoutAll(ctx context.Context) (*bool, error) {
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

func (r *mutationResolver) Like(ctx context.Context, review int) (*bool, error) {
	user := auth.ForContext(ctx)
	// Insert like into database
	commandTag, err := r.DB.Exec(ctx, `INSERT INTO likes(review_id, user_id)
	values($1, $2);`, review, user.ID)
	if commandTag.RowsAffected() != 1 || err != nil {
		return nil, gqlerror.Errorf("Could not create like")
	}
	return nil, nil
}

func (r *mutationResolver) Unlike(ctx context.Context, review int) (*bool, error) {
	user := auth.ForContext(ctx)
	// Remove like from database
	commandTag, err := r.DB.Exec(ctx, `DELETE FROM likes
	WHERE review_id=$1 AND user_id=$2;`, review, user.ID)
	if commandTag.RowsAffected() != 1 || err != nil {
		return nil, gqlerror.Errorf("Like could not be removed")
	}
	return nil, nil
}

func (r *mutationResolver) DeleteReview(ctx context.Context, review int) (*bool, error) {
	user := auth.ForContext(ctx)
	// Remove review from database
	commandTag, err := r.DB.Exec(ctx, `DELETE FROM reviews
	WHERE id=$1 AND user_id=$2;`, review, user.ID)
	if commandTag.RowsAffected() != 1 || err != nil {
		return nil, gqlerror.Errorf("Review could not be deleted")
	}
	return nil, nil
}

func (r *queryResolver) Search(ctx context.Context, q string) (*model.SearchResponse, error) {
	q = strings.TrimSpace(q)
	if len(q) < 3 {
		return &model.SearchResponse{}, nil
	}
	qArray := strings.Fields(q)
	qArgs := make([]interface{}, len(qArray))
	for i, str := range qArray {
		qArgs[i] = str
	}
	query := `SELECT chips.id,chips.name,chips.slug,chips.image,chips.brand_id, brands.name
	FROM chips INNER JOIN brands ON chips.brand_id=brands.id
	WHERE`
	for i := range qArray {
		if i > 0 {
			query += " and"
		}
		query += fmt.Sprintf(` word_similarity($%d, chips.name || ' ' || brands.name) > 0.6`, i+1)
	}
	query += `ORDER BY reviews DESC, length(chips.name), brands.name LIMIT 10;`
	var chips []*model.Chip
	rows, err := r.DB.Query(ctx, query, qArgs...)
	if err != nil {
		fmt.Print(err)
		panic(fmt.Errorf("search (chips) query failed"))
	}
	for rows.Next() {
		chip := &model.Chip{}
		brand := &model.Brand{}
		chip.Brand = brand
		err := rows.Scan(&chip.ID, &chip.Name, &chip.Slug, &chip.Image, &brand.ID, &brand.Name)
		if err != nil {
			fmt.Print(err)
			panic(fmt.Errorf("search (chips) scan failed"))

		}
		chips = append(chips, chip)
	}

	var user *model.User
	rows, err = r.DB.Query(ctx, `SELECT id, username, firstname,lastname, image FROM users WHERE username=$1`, q)
	if err != nil {
		fmt.Print(err)
		panic(fmt.Errorf("search (user) query failed"))

	}
	defer rows.Close()
	if rows.Next() {
		user = &model.User{}
		err := rows.Scan(&user.ID, &user.Username, &user.Firstname, &user.Lastname, &user.Image)
		if err != nil {
			fmt.Print(err)
			panic(fmt.Errorf("search (user) scan failed"))

		}
	}
	return &model.SearchResponse{Chips: chips, User: user}, nil
}

func (r *queryResolver) Chip(ctx context.Context, brand string, slug string) (*model.Chip, error) {
	rows, err := r.DB.Query(ctx, `SELECT chips.name,category,subcategory,chips.slug,chips.image,ingredients,chips.id,brands.id,brands.image,brands.count,brands.name
	FROM chips INNER JOIN brands ON chips.brand_id=brands.id WHERE chips.brand_id=$1 AND chips.slug=$2 LIMIT 1`, brand, slug)
	if err != nil {
		fmt.Print(err)
		panic(fmt.Errorf("chip query failed"))

	}
	defer rows.Close()
	if rows.Next() {
		chip := &model.Chip{}
		brand := &model.Brand{}
		chip.Brand = brand
		err := rows.Scan(&chip.Name, &chip.Category, &chip.Subcategory, &chip.Slug, &chip.Image, &chip.Ingredients, &chip.ID, &brand.ID, &brand.Image, &brand.Count, &brand.Name)
		if err != nil {
			fmt.Print(err)
			panic(fmt.Errorf("chip scan failed"))

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
		panic(fmt.Errorf("chips query failed"))

	}
	for rows.Next() {
		chip := &model.Chip{}
		brand := &model.Brand{}
		chip.Brand = brand
		err := rows.Scan(&chip.Name, &chip.Category, &chip.Subcategory, &chip.Slug, &chip.Image, &chip.Ingredients, &chip.ID, &brand.ID, &brand.Image, &brand.Count, &brand.Name)
		if err != nil {
			fmt.Print(err)
			panic(fmt.Errorf("chips scan failed"))

		}
		chips = append(chips, chip)
	}

	return chips, nil
}

func (r *queryResolver) Brand(ctx context.Context, id string) (*model.Brand, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, image, name, count FROM brands WHERE id=$1 LIMIT 1`, id)
	if err != nil {
		fmt.Print(err)
		panic(fmt.Errorf("brand query failed"))

	}
	defer rows.Close()
	if rows.Next() {
		brand := &model.Brand{}
		err := rows.Scan(&brand.ID, &brand.Image, &brand.Name, &brand.Count)
		if err != nil {
			fmt.Print(err)
			panic(fmt.Errorf("brand scan failed"))

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
		panic(fmt.Errorf("brands query failed"))

	}
	for rows.Next() {
		brand := &model.Brand{}
		err := rows.Scan(&brand.ID, &brand.Image, &brand.Name, &brand.Count)
		if err != nil {
			fmt.Print(err)
			panic(fmt.Errorf("brands scan failed"))

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
			panic(fmt.Errorf("reviews (chips) query failed"))

		}
		for rows.Next() {
			review := &model.Review{}
			user := &model.User{}
			review.User = user
			err := rows.Scan(&review.ID, &review.Rating, &review.Review, &review.Created, &review.Edited, &user.Username, &user.Firstname, &user.Lastname, &user.Image)
			if err != nil {
				fmt.Print(err)
				panic(fmt.Errorf("reviews (chips) scan failed"))

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
			panic(fmt.Errorf("reviews (author) query failed"))

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
				panic(fmt.Errorf("reviews (author) query failed"))
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
