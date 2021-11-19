package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/isqad/kexpress/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli/v2"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	app := &cli.App{
		Name: "kexpress",
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
	err = db.Ping()
	if err != nil {
		return err
	}

	// c := cron.New()
	// c.AddFunc("*/1 * * * *", func() {
	//		log.Println("Hello")
	//})
	//c.Start()
	// err := service.LoadProduct(915181)

	// err = service.LoadProductList(db, 0, 10014)
	err = service.CrawlCategories(db)
	if err != nil {
		return err
	}

	log.Println("All category loaded")

	signalChan := make(chan os.Signal, 1)
	// SIGTERM is called when Ctrl+C was pressed
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-signalChan

	db.Close()

	return nil
}
