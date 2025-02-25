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
	if config == nil || config.RecordRate == nil {
		config = DefaultStreamConfig()
	}

	return &Stream{config: *config, storage: storage}, nil
}

func (s *Stream) GetConfig() Config {
	return s.config
}

func (s *Stream) GetStorage() StorageBackend {
	return s.storage
}

// Reset writer to force a db hit.
func (s *Stream) ResetWriter() {
	s.initLock.Lock()
	defer s.initLock.Unlock()
	s.writeCursor = nil
}

// Sets the minimum time between mutations to 0
func (s *Stream) DisableAmends() {
	s.config.RecordRate.ChangeFrequency = 0
	wc, _ := s.WriteCursor()
	if wc != nil && wc.rateConfig != nil {
		wc.rateConfig.ChangeFrequency = 0
	}
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

func (c *Stream) WriteEntry(entry *StreamEntry) error {
	cursor, err := c.WriteCursor()
	if err != nil {
		return err
	}
	return cursor.WriteEntry(entry, c.config.RecordRate)
}

// Build a new cursor
func (s *Stream) BuildCursor(cursorType CursorType) *Cursor {
	return newCursor(s.storage, cursorType)
}
