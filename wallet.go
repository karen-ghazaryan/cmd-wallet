package cold_wallet

import (
	"fmt"
	bolt "github.com/etcd-io/bbolt"
	"github.com/tyler-smith/go-bip39"
	"io"
	"sources.witchery.io/coven/cold-wallet/config"
)

const (
	mainDataBucket = "mainDataBucket"
	mnemonicPhraseKey = "mk"
)

type wallet struct {
	cfg *config.Config
	db  *bolt.DB
}

func (w wallet) Export(io.Writer) error {
	panic("implement me")
}

func (w wallet) Create(passphrase string) error {
	// todo:
	// if:
	// mnemonic already in db, then show appropriate message
	// else:
	// generate and test wallet from mnemonic
	// encrypt mnemonic with passphrase
	// and save to db

	_, err := w.retrieve([]byte(mnemonicPhraseKey), []byte(passphrase))
	if err != nil {
		return err
	}

	entropy, _ := bip39.NewEntropy(w.cfg.SeedEntropyBits)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	fmt.Println(mnemonic)

	err = w.store([]byte(mnemonicPhraseKey), []byte(mnemonic), []byte(passphrase))
	if err != nil {
		return err
	}

	return nil
}

func (w wallet) retrieve(key, passphrase []byte) ([]byte, error) {
	err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(mainDataBucket))
		v := b.Get(key)
		fmt.Printf("The answer is: %s\n", v)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return nil, nil
}
func (w wallet) store(key, value, passphrase []byte) error {
	// todo: encrypt


	return w.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(mainDataBucket))
		err := b.Put(key, value)
		return err
	})
}

type Wallet interface {
	Export(io.Writer) error
	Create(passphrase string) error
}

func New(db *bolt.DB, c *config.Config) (Wallet, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(mainDataBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &wallet{
		db:  db,
		cfg: c,
	}, nil
}
