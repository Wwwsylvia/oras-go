package oras_test

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
)

func Test(t *testing.T) {
	data := []byte("hello world")
	h := sha1.New()
	h.Write(data)
	desc := ocispec.Descriptor{
		MediaType: "application/test",
		Size:      int64(len(data)),
		Digest:    digest.NewDigestFromBytes("sha1", h.Sum(nil)),
	}

	t.Run("test memory", func(t *testing.T) {
		s := memory.New()
		ctx := context.Background()
		if err := s.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			t.Errorf("failed to push: %v", err)
		}

		if _, err := s.Exists(ctx, desc); err != nil {
			t.Errorf("failed to check existence: %v", err)
		}

		if _, err := s.Fetch(ctx, desc); err != nil {
			t.Errorf("failed to fetch: %v", err)
		}
	})

	t.Run("test file store", func(t *testing.T) {
		s, err := file.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create file store: %v", err)
		}
		ctx := context.Background()

		if err := s.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			t.Errorf("failed to push: %v", err)
		}

		if _, err := s.Exists(ctx, desc); err != nil {
			t.Errorf("failed to check existence: %v", err)
		}

		if _, err := s.Fetch(ctx, desc); err != nil {
			t.Errorf("failed to fetch: %v", err)
		}
	})

	t.Run("test oci store", func(t *testing.T) {
		s, err := oci.New(t.TempDir())
		if err != nil {
			t.Fatalf("failed to create file store: %v", err)
		}
		ctx := context.Background()

		if err := s.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			t.Errorf("failed to push: %v", err)
		}

		if _, err := s.Exists(ctx, desc); err != nil {
			t.Errorf("failed to check existence: %v", err)
		}

		if _, err := s.Fetch(ctx, desc); err != nil {
			t.Errorf("failed to fetch: %v", err)
		}
	})

	t.Run("test repo push", func(t *testing.T) {
		repo, err := remote.NewRepository("localhost:5000/testdgst250321")
		if err != nil {
			t.Fatalf("failed to create repo: %v", err)
		}
		repo.PlainHTTP = true
		ctx := context.Background()

		if err := repo.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			t.Errorf("failed to push: %v", err)
		}
	})

	t.Run("test repo exists", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				t.Errorf("unexpected access: %s %s", r.Method, r.URL)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			switch r.URL.Path {
			case "/v2/test/blobs/" + desc.Digest.String():
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Docker-Content-Digest", desc.Digest.String())
				w.Header().Set("Content-Length", strconv.Itoa(int(desc.Size)))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()
		uri, err := url.Parse(ts.URL)
		if err != nil {
			t.Fatalf("invalid test http server: %v", err)
		}

		repo, err := remote.NewRepository(uri.Host + "/test")
		if err != nil {
			t.Fatalf("NewRepository() error = %v", err)
		}
		repo.PlainHTTP = true
		store := repo.Blobs()
		ctx := context.Background()

		_, err = store.Exists(ctx, desc)
		if err != nil {
			t.Fatalf("Blobs.Exists() error = %v", err)
		}
	})

	t.Run("test repo fetch", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("unexpected access: %s %s", r.Method, r.URL)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			switch r.URL.Path {
			case "/v2/test/blobs/" + desc.Digest.String():
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Docker-Content-Digest", desc.Digest.String())
				if _, err := w.Write(data); err != nil {
					t.Errorf("failed to write %q: %v", r.URL, err)
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()
		uri, err := url.Parse(ts.URL)
		if err != nil {
			t.Fatalf("invalid test http server: %v", err)
		}

		repo, err := remote.NewRepository(uri.Host + "/test")
		if err != nil {
			t.Fatalf("NewRepository() error = %v", err)
		}
		repo.PlainHTTP = true
		store := repo.Blobs()
		ctx := context.Background()

		_, err = store.Fetch(ctx, desc)
		if err != nil {
			t.Fatalf("Blobs.Fetch() error = %v", err)
		}
	})

	t.Run("test repo resolve", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				t.Errorf("unexpected access: %s %s", r.Method, r.URL)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			switch r.URL.Path {
			case "/v2/test/blobs/" + desc.Digest.String():
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Docker-Content-Digest", desc.Digest.String())
				w.Header().Set("Content-Length", strconv.Itoa(int(desc.Size)))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()
		uri, err := url.Parse(ts.URL)
		if err != nil {
			t.Fatalf("invalid test http server: %v", err)
		}

		repoName := uri.Host + "/test"
		repo, err := remote.NewRepository(repoName)
		if err != nil {
			t.Fatalf("NewRepository() error = %v", err)
		}
		repo.PlainHTTP = true
		store := repo.Blobs()
		ctx := context.Background()

		got, err := store.Resolve(ctx, desc.Digest.String())
		if err != nil {
			t.Fatalf("Blobs.Resolve() error = %v", err)
		}
		fmt.Println(got)
	})
}
