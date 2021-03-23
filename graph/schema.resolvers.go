package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/c-wiren/snackstoppen-backend/graph/generated"
	"github.com/c-wiren/snackstoppen-backend/graph/model"
)

func (r *mutationResolver) CreateChip(ctx context.Context, input model.NewChip) (*model.Chip, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Chip(ctx context.Context, brand string, slug string) (*model.Chip, error) {
	rows, err := r.DB.Query(ctx, `SELECT chips.name,category,subcategory,chips.slug,chips.image,ingredients,chips.id,brands.id,brands.image,brands.count,brands.name
	FROM chips INNER JOIN brands ON chips.brand=brands.id WHERE chips.brand=$1 AND chips.slug=$2 LIMIT 1`, brand, slug)
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
	FROM chips INNER JOIN brands ON chips.brand=brands.id`
	if brand != nil {
		argCount++
		q += fmt.Sprint(" WHERE chips.brand=$", argCount)
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

func (r *queryResolver) Reviews(ctx context.Context) ([]*model.Review, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
