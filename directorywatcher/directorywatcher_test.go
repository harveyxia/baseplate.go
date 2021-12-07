package directorywatcher_test

import (
	"context"
	"fmt"
	"github.com/onsi/gomega"
	"github.com/reddit/baseplate.go/directorywatcher"
	"github.com/reddit/baseplate.go/log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDirectoryWatcher(t *testing.T) {
	gomega.RegisterTestingT(t)

	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	// stores a mapping of path to event type
	eventsLog := sync.Map{}

	// start the directory watcher
	watcher, err := directorywatcher.New(
		ctx,
		directorywatcher.Config{
			Path: dir,
			OnCreate: func(path string) error {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				eventsLog.Store(path, fmt.Sprintf("create: %s", data))
				return nil
			},
			OnRemove: func(path string) error {
				eventsLog.Store(path, "delete")
				return nil
			},
			Logger: log.TestWrapper(t),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Stop()

	// 1. Create file
	fpath1 := filepath.Join(dir, "f1")
	if _, err = os.Create(fpath1); err != nil {
		t.Fatal(err)
	}
	pollFor(&eventsLog, fpath1, "create: ")

	// 2. Write to file
	if err = os.WriteFile(fpath1, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	pollFor(&eventsLog, fpath1, "create: hello world")

	// 3. Rename file
	fpath2 := filepath.Join(dir, "f2")
	if err = os.Rename(fpath1, fpath2); err != nil {
		t.Fatal(err)
	}
	// fpath1 data should be deleted
	pollFor(&eventsLog, fpath1, "delete")
	// fpath2 data should be created
	pollFor(&eventsLog, fpath2, "create: hello world")

	// 4. Delete file
	if err = os.Remove(fpath2); err != nil {
		t.Fatal(err)
	}
	pollFor(&eventsLog, fpath2, "delete")

}

// poll for async updates to store
func pollFor(store *sync.Map, key string, expectedVal string) {
	gomega.Eventually(func() string {
		if val, ok := store.Load(key); ok {
			return val.(string)
		}
		return ""
	}, "5s").Should(gomega.Equal(expectedVal))
}
