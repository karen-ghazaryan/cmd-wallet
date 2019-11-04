package main

import (
	"errors"
	"fmt"
	bolt "github.com/etcd-io/bbolt"
	"github.com/manifoldco/promptui"
	"github.com/micro/cli"
	"log"
	"os"
	w "sources.witchery.io/coven/cold-wallet"
	"sources.witchery.io/coven/cold-wallet/config"
)

var (
	app     = cli.NewApp()
	wallet  w.Wallet
	testNet bool
	isForce bool
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("error loading config")
	}

	db, err := bolt.Open(cfg.DbPath, 0755, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()

	wallet, err = w.New(db, cfg)
	if err != nil {
		log.Fatal(err)
	}

	info()
	commands()

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func info() {
	app.Name = "Cold wallet"
	app.Usage = ""
	app.Author = "witchery.io"
	app.Version = "1.0.0"
}

func commands() {
	app.Commands = []cli.Command{
		{
			Name:      "create",
			Aliases:   []string{"c"},
			Usage:     "This command creates new wallet. Appropriate message will be returned if wallet already created.",
			ArgsUsage: "Usage: create ['passphrase']",
			Action:    create,
		},
		{
			Name:      "restore",
			Usage:     "Restore the wallet",
			ArgsUsage: "Usage: restore ['12 world mnemonic phrase', 'passphrase']",
			Action:    restore,
		},
		{
			Name:      "export",
			Aliases:   []string{"e"},
			Usage:     "Use this command to export public extended keys for all supported coins",
			ArgsUsage: "Usage: export ['passphrase']",
			Action:    export,
		},
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "test, t",
			Usage:       "set network type to test",
			Destination: &testNet,
		},
		cli.BoolFlag{
			Name:        "force, f",
			Usage:       "Use force flag if you want run force command execution",
			Destination: &isForce,
		},
	}
}

func create(c *cli.Context) {
	if c.Args().First() == "" {
		fmt.Println(c.Command.ArgsUsage)
		return
	}
	passphrase := c.Args().Get(1)

	//fmt.Println("passphrase:", passphrase)
	//fmt.Println("test:", testNet)


	err := wallet.Create(passphrase)
	if err != nil {
		log.Println(err)
	}
}

func restore(c *cli.Context) {
	if c.NArg() != 2 {
		fmt.Println(c.Command.ArgsUsage)
		return
	}

	mnemonic := c.Args().Get(0)
	passphrase := c.Args().Get(1)

	fmt.Println("mnemonic:", mnemonic)
	fmt.Println("passphrase:", passphrase)
}

func export(c *cli.Context) {
	//fmt.Println(c.Args())
	prompt := promptui.Prompt{
		Label: "Search",
		Validate: func(input string) error {
			if len(input) < 3 {
				return errors.New("search term must have at least 3 characters")
			}
			return nil
		},
	}

	keyword, err := prompt.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(keyword)
}
