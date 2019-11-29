package main

import (
	"fmt"
	bolt "github.com/etcd-io/bbolt"
	"github.com/micro/cli"
	"log"
	"os"
	w "sources.witchery.io/coven/cold-wallet"
	"sources.witchery.io/coven/cold-wallet/config"
	"strings"
)

var (
	app      = cli.NewApp()
	wallet   w.Wallet
	testNet  bool
	forceRun bool

	backupStr = "\nPlease write down this backup phrase in safe place.\n"+
		"Without this you will not be able to restore your wallet.\n"+
		"=====================================================================================\n"+
		"%s\n"+
		"=====================================================================================\n"
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
	testFlag := cli.BoolFlag{
		Name:        "test, t",
		Usage:       "set network type to test",
		Destination: &testNet,
	}
	forceFlag := cli.BoolFlag{
		Name:        "force, f",
		Usage:       "Use force flag if you want run force command execution",
		Destination: &forceRun,
	}

	// global flags
	app.Flags = []cli.Flag{
		testFlag,
		forceFlag,
	}

	app.Commands = []cli.Command{
		{
			Name:      "create",
			Aliases:   []string{"c"},
			Usage:     "This command creates new wallet. Appropriate message will be returned if wallet already created.",
			ArgsUsage: "Usage: create ['passphrase']",
			Flags:     []cli.Flag{forceFlag},
			Action:    create,
		},
		{
			Name:      "import",
			Usage:     "Restore the wallet from 12 world mnemonic phrase",
			ArgsUsage: "Usage: import ['12 world mnemonic phrase', 'passphrase']",
			Flags:     []cli.Flag{forceFlag},
			Action:    importExisting,
		},
		{
			Name:      "export",
			Aliases:   []string{"e"},
			Usage:     "Use this command to export public extended keys for all supported coins",
			ArgsUsage: "Usage: export ['passphrase']",
			Action:    export,
		},
		{
			Name:      "backup",
			Aliases:   []string{"b"},
			Usage:     "Backup wallet",
			ArgsUsage: "Usage: backup ['passphrase']",
			Action:    backup,
		},
	}
}

func create(c *cli.Context) {
	if c.Args().First() == "" {
		fmt.Println(c.Command.ArgsUsage)
		return
	}
	passphrase := c.Args().Get(0)

	mnemonic, err := wallet.Create(passphrase, forceRun)
	if _, existsError := err.(*w.WalletExistsError); existsError {
		log.Printf("Your wallet already created.\n" +
			"You can use -f flag to overwrite existing wallet\n" +
			"NOTE: All old data will be lost forever! " +
			"Do not forget to backup your wallet first")
		return
	}
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf(backupStr, mnemonic)
}

func importExisting(c *cli.Context) {
	if c.NArg() != 2 {
		fmt.Println(c.Command.ArgsUsage)
		return
	}

	mnemonic := c.Args().Get(0)

	if len(strings.Fields(mnemonic)) != 12 {
		log.Println("Invalid mnemonic phrase")
		return
	}
	passphrase := c.Args().Get(1)

	err := wallet.Import(mnemonic, passphrase, forceRun)
	if _, existsError := err.(*w.WalletExistsError); existsError {
		log.Printf("You trying to import on top of exsisting wallet.\n" +
			"You can use -f flag to overwrite existing wallet\n" +
			"NOTE: All old data will be lost forever! " +
			"Do not forget to backup your wallet first")
		return
	}
	if err != nil {
		log.Println(err)
	}
}

func export(c *cli.Context) {
	passphrase := c.Args().First()
	if passphrase == "" {
		fmt.Println(c.Command.ArgsUsage)
		return
	}

	err := wallet.Export(os.Stdout, passphrase)
	if err != nil {
		log.Println(err)
		return
	}
}

func backup(c *cli.Context) {
	passphrase := c.Args().First()
	if passphrase == "" {
		fmt.Println(c.Command.ArgsUsage)
		return
	}

	backupPhrase, err := wallet.Backup(passphrase)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf(backupStr, backupPhrase)
}
