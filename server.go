package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/c-wiren/snackstoppen-backend/auth"
	"github.com/c-wiren/snackstoppen-backend/graph"
	"github.com/c-wiren/snackstoppen-backend/graph/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/rs/cors"
)

const defaultPort = "5000"
const dbURLDev = "postgresql://localhost/snackstoppen_dev"
const mailgunDomain = "mg.snackstoppen.se"

var dev bool

func main() {
	dev = os.Getenv("APP_ENV") != "production"
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	secret := os.Getenv("SECRET")
	if secret != "" {
		auth.Secret = secret
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

	mg := mailgun.NewMailgun(mailgunDomain, os.Getenv("MAILGUN_KEY"))
	mg.SetAPIBase(mailgun.APIBaseEU)

	router := chi.NewRouter()
	router.Use(cors.New(cors.Options{
		AllowCredentials: true,
		AllowedHeaders:   []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization"},
	}).Handler)

	router.Use(auth.Middleware())
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{DB: dbpool, Mailgun: mg}}))
	if dev {
		router.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	}
	router.Handle("/graphql", srv)

	log.Printf("Server listening on port %s", port)
	if dev {
		log.Printf("GraphQL playground running on http://localhost:%s/", port)
	}

	log.Fatal(http.ListenAndServe(":"+port, router))
}
