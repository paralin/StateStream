package stream

import (
	"errors"
	"sync"
	"time"
)

// A state stream instance
type Stream struct {
	config  Config
	storage StorageBackend

	// If initialized, keep a cursor of the latest state.
	writeCursor *Cursor
	initLock    sync.Mutex
}

// Creates a new stream, with a default config if config is nil.
func NewStream(storage StorageBackend, config *Config) (*Stream, error) {
	if storage == nil {
		return nil, errors.New("Storage must be defined.")
	}
	if config == nil {
		config = DefaultStreamConfig()
	}

	return &Stream{config: *config, storage: storage}, nil
}

// Initialize the stream for writing.
// If not called, done automatically at the first write.
func (s *Stream) InitWriter() error {
	s.initLock.Lock()
	defer s.initLock.Unlock()

	if s.writeCursor != nil {
		return nil
	}
	cursor := s.BuildCursor(WriteCursor)
	if err := cursor.Init(time.Now()); err != nil {
		return err
	}
	if !cursor.Ready() {
		return errors.New("Write cursor not ready after init.")
	}
	s.writeCursor = cursor
	return nil
}

// Get the write cursor
func (s *Stream) WriteCursor() (*Cursor, error) {
	if s.writeCursor == nil {
		if err := s.InitWriter(); err != nil {
			return nil, err
		}
	}
	return s.writeCursor, nil
}

func (c *Stream) WriteState(timestamp time.Time, state StateData) error {
	cursor, err := c.WriteCursor()
	if err != nil {
		return err
	}
	return cursor.WriteState(timestamp, state, c.config.RecordRate)
}

// Build a new cursor
func (s *Stream) BuildCursor(cursorType CursorType) *Cursor {
	return newCursor(s.storage, cursorType)
}
