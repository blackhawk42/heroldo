package heroldo

type TokenRegistryEntry struct {
	Username string
	Token    string
}

type TokenRegistry interface {
	NewToken(username string) (string, error)
	VerifyToken(token string) (string, error)
	ListTokens() ([]TokenRegistryEntry, error)
	DeleteTokenByUsername(username string) error
	DeleteTokenByToken(token string) error
	Close() error
}
