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
	"github.com/robfig/cron/v3"
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
	if err = db.Ping(); err != nil {
		return err
	}

	c := cron.New()
	// Одежда
	c.AddFunc("21 0 * * *", func() {
		if err := service.CrawlProductList(db, 5235); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 5235); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// для взрослых
	c.AddFunc("19 2 * * *", func() {
		if err := service.CrawlProductList(db, 7304); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 7304); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// Спорт и отдых
	c.AddFunc("11 3 * * *", func() {
		if err := service.CrawlProductList(db, 7331); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 7331); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// Товары для дома
	c.AddFunc("13 4 * * *", func() {
		if err := service.CrawlProductList(db, 6260); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 6260); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// Акссесуары
	c.AddFunc("11 6 * * *", func() {
		if err := service.CrawlProductList(db, 5675); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 5675); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// красота 5919
	c.AddFunc("37 8 * * *", func() {
		if err := service.CrawlProductList(db, 5919); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 5919); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// Книги 7954
	c.AddFunc("01 18 * * *", func() {
		if err := service.CrawlProductList(db, 7954); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}

		log.Println("INFO: Run crawl products")
		if err := service.CrawlProducts(db, 7954); err != nil {
			log.Printf("ERROR: CrawlProducts, %v\n", err)
		}
		log.Println("DONE")
	})
	// Бытовая техника
	c.AddFunc("11 19 * * *", func() {
		if err := service.CrawlCategories(db); err != nil {
			log.Printf("ERROR: CrawlCategories, %v\n", err)

		}
		if err := service.CrawlProductList(db, 5087); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}
		if err := service.CrawlProducts(db, 5087); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}
		log.Println("DONE")
	})
	// Зоотовары
	c.AddFunc("11 21 * * *", func() {
		if err := service.CrawlProductList(db, 7827); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}
		if err := service.CrawlProducts(db, 7827); err != nil {
			log.Printf("ERROR: CrawlProductList, %v\n", err)
		}
		log.Println("DONE")
	})
	c.Start()

	signalChan := make(chan os.Signal, 1)
	// SIGTERM is called when Ctrl+C was pressed
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-signalChan

	//	c.Stop()
	db.Close()

	return nil
}
