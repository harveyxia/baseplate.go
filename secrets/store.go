package secrets

import (
	"context"
	"github.com/reddit/baseplate.go/directorywatcher"
	"io"

	"github.com/reddit/baseplate.go/filewatcher"
	"github.com/reddit/baseplate.go/log"
)

type (
	// SecretHandlerFunc is the actual function that works with the Secrets
	SecretHandlerFunc func(sec *Secrets)
	// SecretMiddleware creates chain of SecretHandlerFunc calls
	SecretMiddleware func(next SecretHandlerFunc) SecretHandlerFunc
)

func nopSecretHandlerFunc(sec *Secrets) {}

type Store interface {
	// Close closes the underlying file or directory watcher and release associated resources.
	//
	// After Close is called, you won't get any updates to the secrets,
	// but can still access the secrets as they were before Close is called.
	//
	// It's OK to call Close multiple times. Calls after the first one are no-ops.
	//
	// Close doesn't return non-nil errors, but implements io.Closer.
	Close() error

	// AddMiddlewares registers new middlewares to the store.
	//
	// Every AddMiddlewares call will cause all already registered middlewares to be
	// called again with the latest data.
	//
	// AddMiddlewares call is not thread-safe, it should not be called concurrently.
	AddMiddlewares(middlewares ...SecretMiddleware)

	// GetSimpleSecret loads secrets from watcher, and fetches a simple secret from secrets
	GetSimpleSecret(path string) (SimpleSecret, error)

	// GetVersionedSecret loads secrets from watcher, and fetches a versioned secret from secrets
	GetVersionedSecret(path string) (VersionedSecret, error)

	// GetCredentialSecret loads secrets from watcher, and fetches a credential secret from secrets
	GetCredentialSecret(path string) (CredentialSecret, error)

	// GetVault returns a struct with a URL and token to access Vault directly. The
	// token will have policies attached based on the current EC2 server's Vault
	// role. This is only necessary if talking directly to Vault.
	//
	// This function always returns nil error.
	GetVault() (Vault, error)
}

// Store gives access to secret tokens with automatic refresh on change.
//
// This local vault allows access to the secrets cached on disk by the fetcher
// daemon. It will automatically reload the cache when it is changed. Do not
// cache or store the values returned by this class's methods but rather get
// them from this class each time you need them. The secrets are served from
// memory so there's little performance impact to doing so and you will be sure
// to always have the current version in the face of key rotation etc.
type store struct {
	watcher filewatcher.FileWatcher

	secretHandlerFunc SecretHandlerFunc
}

// NewStore returns a new instance of Store by configuring it
// with a filewatcher to watch the file in path for changes ensuring secrets
// store will always return up to date secrets.
//
// Context should come with a timeout otherwise this might block forever, i.e.
// if the path never becomes available.
func NewStore(ctx context.Context, path string, logger log.Wrapper, middlewares ...SecretMiddleware) (Store, error) {
	store := &store{
		secretHandlerFunc: nopSecretHandlerFunc,
	}
	store.secretHandler(middlewares...)

	result, err := filewatcher.New(
		ctx,
		filewatcher.Config{
			Path:   path,
			Parser: store.parser,
			Logger: logger,
		},
	)
	if err != nil {
		return nil, err
	}

	store.watcher = result
	return store, nil
}

func (s *store) parser(r io.Reader) (interface{}, error) {
	secrets, err := NewSecrets(r)
	if err != nil {
		return nil, err
	}

	s.secretHandlerFunc(secrets)

	return secrets, nil
}

// secretHandler creates the middleware chain.
func (s *store) secretHandler(middlewares ...SecretMiddleware) {
	for _, m := range middlewares {
		s.secretHandlerFunc = m(s.secretHandlerFunc)
	}
}

func (s *store) getSecrets() *Secrets {
	return s.watcher.Get().(*Secrets)
}

// Close closes the underlying filewatcher and release associated resources.
//
// After Close is called, you won't get any updates to the secret file,
// but can still access the secrets as they were before Close is called.
//
// It's OK to call Close multiple times. Calls after the first one are no-ops.
//
// Close doesn't return non-nil errors, but implements io.Closer.
func (s *store) Close() error {
	s.watcher.Stop()
	return nil
}

// AddMiddlewares registers new middlewares to the store.
//
// Every AddMiddlewares call will cause all already registered middlewares to be
// called again with the latest data.
//
// AddMiddlewares call is not thread-safe, it should not be called concurrently.
func (s *store) AddMiddlewares(middlewares ...SecretMiddleware) {
	s.secretHandler(middlewares...)
	s.secretHandlerFunc(s.getSecrets())
}

// GetSimpleSecret loads secrets from watcher, and fetches a simple secret from secrets
func (s *store) GetSimpleSecret(path string) (SimpleSecret, error) {
	return s.getSecrets().GetSimpleSecret(path)
}

// GetVersionedSecret loads secrets from watcher, and fetches a versioned secret from secrets
func (s *store) GetVersionedSecret(path string) (VersionedSecret, error) {
	return s.getSecrets().GetVersionedSecret(path)
}

// GetCredentialSecret loads secrets from watcher, and fetches a credential secret from secrets
func (s *store) GetCredentialSecret(path string) (CredentialSecret, error) {
	return s.getSecrets().GetCredentialSecret(path)
}

// GetVault returns a struct with a URL and token to access Vault directly. The
// token will have policies attached based on the current EC2 server's Vault
// role. This is only necessary if talking directly to Vault.
//
// This function always returns nil error.
func (s *store) GetVault() (Vault, error) {
	return s.getSecrets().vault, nil
}

type vaultCsiStore struct {
	watcher directorywatcher.DirectoryWatcher

	secretHandlerFunc SecretHandlerFunc
}

func (s *vaultCsiStore) Close() error {
	s.watcher.Stop()
	return nil
}

func (s *vaultCsiStore) AddMiddlewares(middlewares ...SecretMiddleware) {
	//TODO implement me
	panic("implement me")
}

func (s *vaultCsiStore) GetSimpleSecret(path string) (SimpleSecret, error) {
	//TODO implement me
	panic("implement me")
}

func (s *vaultCsiStore) GetVersionedSecret(path string) (VersionedSecret, error) {
	//TODO implement me
	panic("implement me")
}

func (s *vaultCsiStore) GetCredentialSecret(path string) (CredentialSecret, error) {
	//TODO implement me
	panic("implement me")
}

func (s *vaultCsiStore) GetVault() (Vault, error) {
	//TODO implement me
	panic("implement me")
}

func NewVaultCsiStore(ctx context.Context, path string, logger log.Wrapper, middlewares ...SecretMiddleware) (Store, error) {
	store := &vaultCsiStore{
		secretHandlerFunc: nopSecretHandlerFunc,
	}
	store.secretHandler(middlewares...)

	watcher, err := directorywatcher.New(ctx, directorywatcher.Config{
		Path:     "",
		OnCreate: nil,
		OnRemove: nil,
		Logger:   nil,
	})
	if err != nil {
		return nil, err
	}

	store.directoryWatcher = watcher

	return store, nil
}

// secretHandler creates the middleware chain.
func (s *vaultCsiStore) secretHandler(middlewares ...SecretMiddleware) {
	for _, m := range middlewares {
		s.secretHandlerFunc = m(s.secretHandlerFunc)
	}
}
