package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	app := &cli.App{
		Name: "kexpress",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "postgres-user", Aliases: []string{"u"}, Value: "postgres"},
			&cli.StringFlag{Name: "postgres-password", Aliases: []string{"P"}, Required: true},
			&cli.StringFlag{Name: "postgres-db", Aliases: []string{"d"}, Value: "lolive"},
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
	log.Println("Hello")
	return nil
}
