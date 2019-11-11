package cold_wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	bolt "github.com/etcd-io/bbolt"
	"github.com/tyler-smith/go-bip39"
	"io"
	"sources.witchery.io/coven/cold-wallet/config"
	"sources.witchery.io/coven/cold-wallet/internal/encrypting"
	"sources.witchery.io/packages/wallet/coin"
)

const (
	mainDataBucket    = "mainDataBucket"
	mnemonicPhraseKey = "mk"
	privatePrefix     = "prv"
	publicPrefix      = "pub"
	// Purpose is a constant set to 44' (or 0x8000002C) following the BIP43 recommendation.
	// It indicates that the subtree of this node is used according to this specification.
	Purpose = 0x8000002C
)

type Wallet interface {
	Export(w io.Writer, passphrase string) error
	Create(passphrase string, forceRun bool) (string, error)
	Import(mnemonic, passphrase string, forceRun bool) error
	Backup(passphrase string) (string, error)
}

type WalletExistsError struct {
	msg string
}

func (e *WalletExistsError) Error() string {
	return "wallet already exists"
}

type wallet struct {
	cfg *config.Config
	db  *bolt.DB
}

func (w *wallet) Export(writer io.Writer, passphrase string) error {
	enc, err := encrypting.New(passphrase)
	if err != nil {
		return err
	}

	return w.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(mainDataBucket)).Cursor()

		prefix := []byte(publicPrefix)
		var m = map[string]string{}
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			k = bytes.TrimPrefix(k, prefix)

			ev, err := enc.Decrypt(v)
			if err != nil {
				return err
			}

			m[string(k)] = string(ev)
		}

		enc := json.NewEncoder(writer)
		enc.SetIndent("", "    ")
		err := enc.Encode(&m)
		if err != nil {
			return err
		}

		return nil
	})
}

func (w *wallet) Create(passphrase string, force bool) (string, error) {
	entropy, err := bip39.NewEntropy(w.cfg.SeedEntropyBits)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}
	err = w.Import(mnemonic, passphrase, force)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func (w *wallet) Import(mnemonic, passphrase string, force bool) error {
	if force {
		if err := w.truncateDatabase(); err != nil {
			return err
		}
	} else {
		mnemonic, err := w.retrieve(mnemonicPhraseKey, passphrase)
		if err != nil {
			return err
		}

		if mnemonic != "" {
			return new(WalletExistsError)
		}
	}

	err := w.storeChainKeyPairs(mnemonic, passphrase)
	if err != nil {
		return err
	}

	err = w.store(mnemonicPhraseKey, mnemonic, passphrase)
	if err != nil {
		return err
	}

	return nil
}

func (w *wallet) Backup(passphrase string) (string, error) {
	mnemonic, err := w.retrieve(mnemonicPhraseKey, passphrase)
	if err != nil {
		return "", err
	}

	if mnemonic == "" {
		return "", errors.New("something went wrong, unable to backup")
	}
	return mnemonic, nil
}

func (w *wallet) storeChainKeyPairs(mnemonic, passphrase string) error {
	tx, err := w.db.Begin(true)
	if err != nil {
		return err
	}

	masterKey, err := masterKeyFromMnemonic(mnemonic)
	if err != nil {
		return tx.Rollback()
	}

	enc, err := encrypting.New(passphrase)
	if err != nil {
		return err
	}

	for coinName, coinType := range coin.SupportedCoinTypes() {
		if coinName == coin.NameTestBTC {
			masterKey.SetNet(&chaincfg.TestNet3Params)
		} else {
			masterKey.SetNet(&chaincfg.MainNetParams)
		}
		coinChainKey, err := GetCurrencyChain(masterKey, coinType)
		if err != nil {
			return tx.Rollback()
		}

		keyPrv := bytes.Join([][]byte{[]byte(privatePrefix), []byte(coinName)}, []byte{})
		keyPub := bytes.Join([][]byte{[]byte(publicPrefix), []byte(coinName)}, []byte{})

		v, err := enc.Encrypt([]byte(coinChainKey.String()))
		if err != nil {
			return tx.Rollback()
		}

		err = tx.Bucket([]byte(mainDataBucket)).Put(keyPrv, v)
		if err != nil {
			return tx.Rollback()
		}

		neutered, _ := coinChainKey.Neuter()

		v, err = enc.Encrypt([]byte(neutered.String()))
		if err != nil {
			return tx.Rollback()
		}

		err = tx.Bucket([]byte(mainDataBucket)).Put(keyPub, v)
		if err != nil {
			return tx.Rollback()
		}

		fmt.Printf("%s is %s\n", string(keyPrv), coinChainKey.String())
		fmt.Printf("%s is %s\n", string(keyPub), neutered.String())
	}

	return tx.Commit()
}

func masterKeyFromMnemonic(mnemonic string) (*hdkeychain.ExtendedKey, error) {
	seedBytes, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return hdkeychain.NewMaster(seedBytes, &chaincfg.MainNetParams)
}

func GetCurrencyChain(master *hdkeychain.ExtendedKey, coinType uint32) (*hdkeychain.ExtendedKey, error) {
	child, err := master.Child(Purpose) // m/44'
	if err != nil {
		return nil, err
	}

	child, err = child.Child(coinType) // m/44'/coinType`
	if err != nil {
		return nil, err
	}

	child, err = child.Child(hdkeychain.HardenedKeyStart) // m/44'/coinType'/account'
	if err != nil {
		return nil, err
	}

	child, err = child.Child(0) // m/44'/coinType'/account'/0
	if err != nil {
		return nil, err
	}

	return child, nil
}

func (w *wallet) retrieve(key, passphrase string) (string, error) {
	var data []byte
	err := w.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(mainDataBucket))
		data = b.Get([]byte(key))
		return nil
	})

	if err != nil {
		return "", err
	}

	enc, err := encrypting.New(passphrase)
	if err != nil {
		return "", err
	}

	data, err = enc.Decrypt(data)
	if err != nil {
		return  "", err
	}

	return string(data), nil
}
func (w *wallet) store(key, value, passphrase string) error {
	v := []byte(value)
	enc, err := encrypting.New(passphrase)
	if err != nil {
		return err
	}

	v, err = enc.Encrypt(v)
	if err != nil {
		return err
	}

	return w.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(mainDataBucket))
		err := b.Put([]byte(key), v)
		return err
	})
}

func (w *wallet) truncateDatabase() error {
	tx, err := w.db.Begin(true)
	if err != nil {
		return err
	}

	bct := []byte(mainDataBucket)

	err = tx.DeleteBucket(bct)
	if err != nil {
		return err
	}

	_, err = tx.CreateBucket(bct)
	if err != nil {
		return err
	}

	return tx.Commit()
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
