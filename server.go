package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/c-wiren/snackstoppen-backend/graph"
	"github.com/c-wiren/snackstoppen-backend/graph/generated"
	"github.com/jackc/pgx/v4/pgxpool"
)

const defaultPort = "5000"
const dbURLDev = "postgresql://localhost/snackstoppen_dev"

var dev bool

func main() {
	dev = os.Getenv("APP_ENV") != "production"
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dev {
		dbURL = dbURLDev
	}
	dbpool, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()
	log.Printf("Connected to DB")

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{DB: dbpool}}))
	if dev {
		http.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	}
	http.Handle("/graphql", srv)

	log.Printf("Server listening on port %s", port)
	if dev {
		log.Printf("GraphQL playground running on http://localhost:%s/", port)
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
