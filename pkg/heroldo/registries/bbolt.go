package registries

import (
	"bytes"
	"crypto/sha3"
	"errors"
	"fmt"
	"sync"

	"github.com/blackhawk42/heroldo/pkg/heroldo"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.etcd.io/bbolt"
)

// BBoltTokenRegistry is a TokenRegistry implementation backed by a bbolt
// database.
//
// Tokens are stored as SHA3-256 hashes keyed by the hash. Tokens are generated
// as a Nano ID using the matoous/go-nanoid library.
//
// Tokens are doubly-indexed in two different buckets: one that maps the hashed
// token to its username, and one that maps the username to the token.
// This means creation and deletion have some extra bookkeeping
// and there's some space overhead, but lookups should remain fast.
//
// Close closes the underlying database. Close is safe to call multiple times;
// the underlying database will only be closed once.
//
// Note that bbolt limitations of concurrent processes apply. If the database
// is not released, other processes will hang when attempting to get a lock on it.
type BBoltTokenRegistry struct {
	db           *bbolt.DB
	usersBucket  []byte
	tokensBucket []byte
	tokenLength  int
	closeOnce    sync.Once
	closeErr     error
}

// ErrTokensBucketDoesNotExist is returned when the tokens bucket is missing
// from the database.
var ErrTokensBucketDoesNotExist = errors.New("tokens bucket does not exist")

// ErrUsersBucketDoesNotExist is returned when the users bucket is missing from
// the database.
var ErrUsersBucketDoesNotExist = errors.New("users bucket does not exist")

// NewBBoltTokenRegistry creates a BBoltTokenRegistry backed by the given
// pre-opened bbolt database.
//
// If tokenLength is <= 0, it defaults to 64.
//
// If usersBucket or tokensBucket are nil, default values are []byte("users")
// and []byte("tokens"), respectively. Making them equal is an error (unless they're
// both nil). Both parameters are internally copied before use.
func NewBBoltTokenRegistry(db *bbolt.DB, tokenLength int, usersBucket []byte, tokensBucket []byte) (*BBoltTokenRegistry, error) {
	if db == nil {
		return nil, fmt.Errorf("a non-nil bbolt database must be provided")
	}

	if tokenLength <= 0 {
		tokenLength = 64
	}

	if usersBucket == nil {
		usersBucket = []byte("users")
	} else {
		usersBucket = bytes.Clone(usersBucket)
	}

	if tokensBucket == nil {
		tokensBucket = []byte("tokens")
	} else {
		tokensBucket = bytes.Clone(tokensBucket)
	}

	if bytes.Equal(usersBucket, tokensBucket) {
		return nil, fmt.Errorf("usersBucket and tokensBucket should not be equal")
	}

	// The tokens bucket is meant to be keyed by token hash.
	// The users bucket by username
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(tokensBucket)
		if err != nil {
			return fmt.Errorf("while creating tokens bucket: %w", err)
		}

		_, err = tx.CreateBucketIfNotExists(usersBucket)
		if err != nil {
			return fmt.Errorf("while creating users bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("while creating initial buckets: %w", err)
	}

	return &BBoltTokenRegistry{
		db:           db,
		tokenLength:  tokenLength,
		usersBucket:  usersBucket,
		tokensBucket: tokensBucket,
		closeErr:     nil,
	}, nil
}

// NewToken generates a new random token for the given username, stores its
// SHA3-256 hash, and returns the raw token.
//
// If the given username already had a token associated with it, it will be
// deleted and overwritten.
//
// An empty username returns an error.
//
// Note that the actual token is not recoverable, by design. This return
// value is the only chance to record or present it before it's lost. If
// a token is lost, you'll have to create a new one for the same user,
// overwriting the existing one.
func (bbr *BBoltTokenRegistry) NewToken(username string) (string, error) {
	if username == "" {
		return "", fmt.Errorf("username must not be empty")
	}

	token, err := gonanoid.New(bbr.tokenLength)
	if err != nil {
		return "", fmt.Errorf("while generating token: %w", err)
	}

	hash := sha3.Sum256([]byte(token))

	err = bbr.db.Batch(func(tx *bbolt.Tx) error {
		tokensBucket := tx.Bucket(bbr.tokensBucket)
		if tokensBucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		usersBucket := tx.Bucket(bbr.usersBucket)
		if usersBucket == nil {
			return ErrUsersBucketDoesNotExist
		}

		// Delete any other previous token associated with this username
		previousHashedToken := usersBucket.Get([]byte(username))
		if previousHashedToken != nil {
			err := tokensBucket.Delete(previousHashedToken)
			if err != nil {
				return fmt.Errorf("while removing previous token (hash: %x) associated with this user (user: %s): %w", previousHashedToken, username, err)
			}
		}

		err := tokensBucket.Put([]byte(hash[:]), []byte(username))
		if err != nil {
			return fmt.Errorf("while saving data to tokens bucket: %w", err)
		}

		// No preemptive delete necessary for user, as it will simply be overwritten
		// if existant.
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

// VerifyToken hashes the given token and looks it up in the database.
//
// It returns the associated username, or an empty string if the token is not
// found.
func (bbr *BBoltTokenRegistry) VerifyToken(token string) (string, error) {
	hashedToken := sha3.Sum256([]byte(token))

	savedUser := ""

	err := bbr.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bbr.tokensBucket)
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

// ListTokens iterates the users bucket and returns all stored
// username-token_hash pairs.
func (bbr *BBoltTokenRegistry) ListTokens() ([]heroldo.TokenRegistryEntry, error) {
	var entries []heroldo.TokenRegistryEntry

	err := bbr.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(bbr.usersBucket)
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

// DeleteTokenByUsername looks up the token hash for the given username and
// removes the entries from both the users and tokens buckets.
//
// If the username didn't exist, returns a nil error.
func (bbr *BBoltTokenRegistry) DeleteTokenByUsername(username string) error {
	usernameBytes := []byte(username)

	err := bbr.db.Batch(func(tx *bbolt.Tx) error {
		tokensBucket := tx.Bucket(bbr.tokensBucket)
		if tokensBucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		usersBucket := tx.Bucket(bbr.usersBucket)
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

// DeleteTokenByToken hashes the given token and removes the entries from both
// the tokens and users buckets.
//
// If the token didn't exist, returns a nil error.
func (bbr *BBoltTokenRegistry) DeleteTokenByToken(token string) error {
	tokenHash := sha3.Sum256([]byte(token))
	tokenBytes := tokenHash[:]

	err := bbr.db.Batch(func(tx *bbolt.Tx) error {
		tokensBucket := tx.Bucket(bbr.tokensBucket)
		if tokensBucket == nil {
			return ErrTokensBucketDoesNotExist
		}

		usersBucket := tx.Bucket(bbr.usersBucket)
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

// Close closes the underlying bbolt database.
//
// Close is safe to call multiple times; the underlying database will only
// be closed once.
//
// There may be reasons not to call this method (e. g., if the same underlying
// database is being used for other purposes). It's the caller's responsibility
// to determine this. It is safe to close the database externally and then do it
// through here.
func (bbr *BBoltTokenRegistry) Close() error {
	// NOTE: It is currently true that a bbolt database is safe to close multiple
	// times, even without guard. That's why I give the last guarantee in the documentation.
	// But this is not explicitly documented or guaranteed on their part. bbolt
	// is a very stable project, but this needs an eye to be kept on it.

	bbr.closeOnce.Do(func() {
		err := bbr.db.Close()
		if err != nil {
			bbr.closeErr = fmt.Errorf("while closing the bbolt-based registry: %w", err)
		}
	})
	return bbr.closeErr
}
