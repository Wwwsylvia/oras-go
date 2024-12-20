package credentials

import (
	"errors"
	"io"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials/internal/config"
)

type ReadOnlyFileStore struct {
	*config.ReadOnlyConfig
}

var ErrReadOnlyStore = errors.New("cannot modify content of the read-only store")

func NewReadOnlyFileStoreFromReader(reader io.Reader) (*ReadOnlyFileStore, error) {
	cfg, err := config.LoadFromReader(reader)
	if err != nil {
		return nil, err
	}
	return &ReadOnlyFileStore{ReadOnlyConfig: cfg}, nil
}

func (fs *ReadOnlyFileStore) Get(serverAddress string) (auth.Credential, error) {
	return fs.ReadOnlyConfig.GetCredential(serverAddress)
}

func (fs *ReadOnlyFileStore) Put(serverAddress string, cred auth.Credential) error {
	return ErrReadOnlyStore
}

func (fs *ReadOnlyFileStore) Delete(serverAddress string) error {
	return ErrReadOnlyStore
}
