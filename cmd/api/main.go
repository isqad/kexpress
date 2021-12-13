package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/isqad/kexpress/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli/v2"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	app := &cli.App{
		Name: "kexpress-api",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "postgres-user", Aliases: []string{"u"}, Value: "postgres"},
			&cli.StringFlag{Name: "postgres-password", Aliases: []string{"P"}, Required: true},
			&cli.StringFlag{Name: "postgres-db", Aliases: []string{"d"}, Value: "kexpress"},
			&cli.StringFlag{Name: "postgres-host", Aliases: []string{"c"}, Value: "localhost"},
			&cli.StringFlag{Name: "postgres-port", Aliases: []string{"p"}, Value: "15432"},
		},
		Action: startServer,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func startServer(ctx *cli.Context) error {
	dataSrcName := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		ctx.String("postgres-user"),
		ctx.String("postgres-password"),
		ctx.String("postgres-host"),
		ctx.String("postgres-port"),
		ctx.String("postgres-db"),
	)
	db, err := sqlx.Connect("pgx", dataSrcName)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/api/v1/roots", func(w http.ResponseWriter, r *http.Request) {
		rubrics, err := service.RootCategories(db)
		if err != nil {
			log.Fatal(err)
		}

		response, err := json.Marshal(rubrics)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(response)
	})
	r.Get("/api/v1/categories", func(w http.ResponseWriter, r *http.Request) {
		params, ok := r.URL.Query()["root_id"]
		if !ok || len(params[0]) < 1 {
			log.Fatal(errors.New("No root_id"))
		}
		rootID, err := strconv.ParseInt(params[0], 10, 64)
		if err != nil {
			log.Fatal(errors.New("No root_id"))
		}

		rubrics, err := service.CategoryLeaves(db, rootID)
		if err != nil {
			log.Fatal(err)
		}
		response, err := json.Marshal(rubrics)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(response)
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("app").ParseFiles(
			"web/templates/layout.html",
			"web/templates/index.html",
		)
		if err != nil {
			log.Fatal(err)
		}

		tmpl.ExecuteTemplate(w, "layout.html", nil)
	})
	// Serve static assets
	// serves files from web/static dir
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	staticPrefix := "/static/"
	staticDir := path.Join(cwd, "web", staticPrefix)
	r.Method("GET", staticPrefix+"*", http.StripPrefix(staticPrefix, http.FileServer(http.Dir(staticDir))))

	// Configure the HTTP server
	server := &http.Server{
		Addr:              ":3000",
		Handler:           r,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	// Start HTTP server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
