package stream

import (
	"time"
)

// Generic interface to stream storage
type StorageBackend interface {
	// Retrieve the first snapshot before timestamp. Return nil for no data.
	GetSnapshotBefore(timestamp time.Time) (*StreamEntry, error)
	// Retrieve the first mutation after timestamp. Return nil for no data.
	GetMutationAfter(timestamp time.Time) (*StreamEntry, error)
	// Store a stream entry.
	SaveEntry(entry *StreamEntry) error
	// Amend an old entry
	AmendEntry(entry *StreamEntry, oldTimestamp time.Time) error
}
