package registries

import (
	"crypto/sha3"
	"errors"
	"fmt"

	"github.com/blackhawk42/heroldo/pkg/heroldo"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.etcd.io/bbolt"
)

type BBoltTokenRegistry struct {
	db          *bbolt.DB
	tokenLength int
}

var ErrTokensBucketDoesNotExist = errors.New("tokens bucket does not exist")
var ErrUsersBucketDoesNotExist = errors.New("users bucket does not exist")

func NewBBoltTokenRegistry(path string, bboltOptions *bbolt.Options, tokenLength int) (*BBoltTokenRegistry, error) {
	if tokenLength <= 0 {
		tokenLength = 64
	}

	db, err := bbolt.Open(path, 0600, bboltOptions)
	if err != nil {
		return nil, fmt.Errorf("while opening bbolt-based registry: %w", err)
	}

	// The tokens bucket is meant to be keyed by token hash.
	// The users bucket by username
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("tokens"))
		if err != nil {
			return fmt.Errorf("while creating tokens bucket: %w", err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("while creating users bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("while creating initial buckets: %w", err)
	}

	return &BBoltTokenRegistry{db: db, tokenLength: tokenLength}, nil
}

func (bbr *BBoltTokenRegistry) NewToken(username string) (string, error) {
	token, err := gonanoid.New(bbr.tokenLength)
	if err != nil {
		return "", fmt.Errorf("while generating token: %w", err)
	}

	hash := sha3.Sum256([]byte(token))

	err = bbr.db.Batch(func(tx *bbolt.Tx) error {
		tokensBucket := tx.Bucket([]byte("tokens"))
		if tokensBucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		usersBucket := tx.Bucket([]byte("users"))
		if usersBucket == nil {
			return ErrUsersBucketDoesNotExist
		}

		err := tokensBucket.Put([]byte(hash[:]), []byte(username))
		if err != nil {
			return fmt.Errorf("while saving data to tokens bucket: %w", err)
		}

		err = usersBucket.Put([]byte(username), []byte(hash[:]))
		if err != nil {
			return fmt.Errorf("while saving data to users bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("while setting token in registry: %w", err)
	}

	return token, nil
}

func (bbr *BBoltTokenRegistry) VerifyToken(token string) (string, error) {
	hashedToken := sha3.Sum256([]byte(token))

	savedUser := ""

	err := bbr.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		if bucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		savedUserBytes := bucket.Get([]byte(hashedToken[:]))
		savedUser = string(savedUserBytes)

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("while verifying token existence: %w", err)
	}

	return savedUser, nil
}

func (bbr *BBoltTokenRegistry) ListTokens() ([]heroldo.TokenRegistryEntry, error) {
	var entries []heroldo.TokenRegistryEntry

	err := bbr.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		if bucket == nil {
			return ErrUsersBucketDoesNotExist
		}

		err := bucket.ForEach(func(k, v []byte) error {
			// This code runs under the assumption that converting []byte to string
			// yields a copy, and therefore doesn't violate bbolt's ownership.
			entries = append(entries, heroldo.TokenRegistryEntry{Username: string(k), Token: string(v)})

			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("while getting username/token entries: %w", err)
	}

	return entries, nil
}

func (bbr *BBoltTokenRegistry) DeleteTokenByUsername(username string) error {
	usernameBytes := []byte(username)

	err := bbr.db.Batch(func(tx *bbolt.Tx) error {
		tokensBucket := tx.Bucket([]byte("tokens"))
		if tokensBucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		usersBucket := tx.Bucket([]byte("users"))
		if usersBucket == nil {
			return ErrUsersBucketDoesNotExist
		}

		tokenHash := usersBucket.Get(usernameBytes)
		if tokenHash == nil {
			return nil
		}

		err := usersBucket.Delete(usernameBytes)
		if err != nil {
			return fmt.Errorf("while deleting user from users bucket: %w", err)
		}

		err = tokensBucket.Delete(tokenHash)
		if err != nil {
			return fmt.Errorf("while deleting token from tokens bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("while deleting token: %w", err)
	}

	return nil
}

func (bbr *BBoltTokenRegistry) DeleteTokenByToken(token string) error {
	tokenHash := sha3.Sum256([]byte(token))
	tokenBytes := tokenHash[:]

	err := bbr.db.Batch(func(tx *bbolt.Tx) error {
		tokensBucket := tx.Bucket([]byte("tokens"))
		if tokensBucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		usersBucket := tx.Bucket([]byte("users"))
		if usersBucket == nil {
			return ErrUsersBucketDoesNotExist
		}

		username := tokensBucket.Get(tokenBytes)
		if username == nil {
			return nil
		}

		err := tokensBucket.Delete(tokenBytes)
		if err != nil {
			return fmt.Errorf("while deleting token from tokens bucket: %w", err)
		}

		err = usersBucket.Delete(username)
		if err != nil {
			return fmt.Errorf("while deleting user from users bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("while deleting token: %w", err)
	}

	return nil
}

func (bbr *BBoltTokenRegistry) Close() error {
	err := bbr.db.Close()
	if err != nil {
		return fmt.Errorf("while closing the bbolt-based registry: %w", err)
	}

	return nil
}
