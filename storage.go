package stream

import (
	"time"
)

// Generic interface to stream storage
type StorageBackend interface {
	// Retrieve the first snapshot before timestamp. Return nil for no data.
	GetSnapshotBefore(timestamp time.Time) (*StreamEntry, error)
	// Get the next entry after the timestamp. Return nil for no data.
	// Filter by the filter type, or don't filter if StreamEntryAny
	GetEntryAfter(timestamp time.Time, filterType StreamEntryType) (*StreamEntry, error)
	// Store a stream entry.
	SaveEntry(entry *StreamEntry) error
	// Amend an old entry
	AmendEntry(entry *StreamEntry, oldTimestamp time.Time) error
	// Iterate over all entries
	ForEachEntry(func(entry *StreamEntry) error) error
}

type StreamingStorageBackend interface {
	EntryAdded(chan<- *StreamEntry)
}
