package heroldo

// TokenRegistryEntry represents a username and its associated token.
//
// Depending on the specific implementation of the registry, the "token" may
// may be something like a hash of the actual token.
type TokenRegistryEntry struct {
	Username string
	Token    string
}

// TokenRegistry defines the interface for managing authentication tokens.
type TokenRegistry interface {
	// NewToken generates and stores a new token for the given username.
	//
	// It's responsibility of the implementation to make the token large enough,
	// securely generated (with a CSPRNG), and suitable for putting in an
	// Authorization HTTP header.
	//
	// It's also responsibility of the implementation to securely store the tokens
	// in a secure way (hash, encryption, etc.)
	NewToken(username string) (string, error)

	// VerifyToken checks whether a token is valid and returns the associated
	// username.
	//
	// A nil error and empty username indicates that there's no associated
	// username to that token. A nil error and non-empty username indicates
	// the token is valid.
	VerifyToken(token string) (string, error)

	// ListTokens returns all stored username-token pairs.
	//
	// Depending on the implementation, TokenRegistryEntry.Token may be empty, a hash
	// or some other thing, depending on how tokens are actually stored.
	ListTokens() ([]TokenRegistryEntry, error)

	// DeleteTokenByUsername removes the token associated with the given username.
	DeleteTokenByUsername(username string) error

	// DeleteTokenByToken removes the given token from the registry.
	DeleteTokenByToken(token string) error

	// Close releases any resources held by the registry.
	Close() error
}
