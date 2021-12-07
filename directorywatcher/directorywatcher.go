package directorywatcher

import (
	"context"
	"github.com/reddit/baseplate.go/log"
	"gopkg.in/fsnotify.v1"
)

// DirectoryWatcher watches for changes to files in a directory in a goroutine, invoking
// the provided on create handler in response to file creates and writes,
// and the delete handler in response for file renames and removals.
type DirectoryWatcher interface {
	// Stop the DirectoryWatcher.
	Stop()
}

// Invoked on file creates and writes.
type OnCreate func(path string) error

// Invoked on file renames (on the old file name) and removals.
type OnRemove func(path string) error

// Config defines the config to be used in New function.
//
// Can be deserialized from YAML.
type Config struct {
	// The path to the directory to be watched, required.
	Path string `yaml:"path"`

	// Invoked when files are created or written.
	OnCreate OnCreate

	// Invoked when files are deleted or renamed.
	// E.g. if "/dir/f1" is renamed to "/dir/f2", fsNotify will report a rename event for f1 and create event for f2
	OnRemove OnRemove

	// Optional. When non-nil, it will be used to log errors.
	Logger log.Wrapper `yaml:"logger"`
}

type directoryWatcher struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *directoryWatcher) Stop() {
	w.cancel()
}

func (w *directoryWatcher) watcherLoop(
	watcher *fsnotify.Watcher,
	onCreate OnCreate,
	onRemove OnRemove,
	logger log.Wrapper,
) {
	for {
		select {
		case <-w.ctx.Done():
			watcher.Close()
			return

		case err := <-watcher.Errors:
			logger.Log(context.Background(), "directorywatcher: watcher error: "+err.Error())

		case ev := <-watcher.Events:
			switch ev.Op {
			case fsnotify.Create, fsnotify.Write:
				if err := onCreate(ev.Name); err != nil {
					logger.Log(w.ctx, "directorywatcher: create handler error: "+err.Error())
				}
			case fsnotify.Remove, fsnotify.Rename:
				if err := onRemove(ev.Name); err != nil {
					logger.Log(w.ctx, "directorywatcher: remove handler error: "+err.Error())
				}
			default:
				// Ignore uninterested events, i.e. chmod.
			}
		}
	}
}

func New(ctx context.Context, cfg Config) (DirectoryWatcher, error) {
	var fsWatcher *fsnotify.Watcher
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = fsWatcher.Add(cfg.Path)
	if err != nil {
		return nil, err
	}

	watcher := &directoryWatcher{}
	watcher.ctx, watcher.cancel = context.WithCancel(ctx)

	// start the watcher loop with provided handlers
	go watcher.watcherLoop(fsWatcher, cfg.OnCreate, cfg.OnRemove, cfg.Logger)

	return watcher, nil
}
