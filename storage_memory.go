package stream

import (
	"sort"
	"sync"
	"time"
)

// A general purpose memory backend for storage.
type MemoryBackend struct {
	Entries    []*StreamEntry
	EntriesMtx sync.RWMutex

	subscribers    []chan<- *StreamEntry
	subscribersMtx sync.RWMutex
}

// Retrieve the first snapshot before timestamp. Return nil for no data.
func (mb *MemoryBackend) GetSnapshotBefore(timestamp time.Time) (*StreamEntry, error) {
	mb.EntriesMtx.RLock()
	defer mb.EntriesMtx.RUnlock()

	_, idx := mb.findClosest(timestamp)

	// Go back 1 index, the one we found is AT OR AFTER timestamp.
	// Thus we need to go back at least 1 to get BEFORE that timestamp.
	idx--

	if idx < 0 {
		return nil, nil
	}

	// Iterate backward in time until we have a entry that matches.
	for i := idx; i >= 0; i-- {
		ent := mb.Entries[i]
		if ent.Type == StreamEntrySnapshot {
			return ent, nil
		}
	}
	return nil, nil
}

// Get the next entry after the timestamp. Return nil for no data.
// Filter by the filter type, or don't filter if StreamEntryAny
func (mb *MemoryBackend) GetEntryAfter(timestamp time.Time, filterType StreamEntryType) (*StreamEntry, error) {
	mb.EntriesMtx.RLock()
	defer mb.EntriesMtx.RUnlock()

	closest, idx := mb.findClosest(timestamp)
	if closest == nil || filterType == StreamEntryAny {
		return closest, nil
	}

	// Iterate forward in time until we have a entry that matches.
	entryCount := len(mb.Entries)
	for i := idx; i < entryCount; i++ {
		ent := mb.Entries[i]
		if ent.Type == filterType {
			return ent, nil
		}
	}
	return nil, nil
}

// Note: RLock EntriesMtx before calling
func (mb *MemoryBackend) findClosest(timestamp time.Time) (*StreamEntry, int) {
	entryCount := len(mb.Entries)
	if entryCount == 0 {
		return nil, -1
	}
	// Find the smallest index ON OR AFTER the required timestamp.
	idx := sort.Search(entryCount, func(i int) bool {
		return mb.Entries[i].Timestamp.Equal(timestamp) ||
			mb.Entries[i].Timestamp.After(timestamp)
	})
	if idx == entryCount || idx < 0 {
		return nil, -1
	}
	return mb.Entries[idx], idx
}

// Store a stream entry.
func (mb *MemoryBackend) SaveEntry(entry *StreamEntry) error {
	mb.EntriesMtx.Lock()
	defer mb.EntriesMtx.Unlock()

	if mb.Entries == nil || len(mb.Entries) == 0 {
		mb.Entries = []*StreamEntry{entry}
		return nil
	}

	_, idx := mb.findClosest(entry.Timestamp)
	if idx != -1 {
		s := mb.Entries
		mb.Entries = append(s[:idx], append([]*StreamEntry{entry}, s[idx:]...)...)
	} else {
		mb.Entries = append(mb.Entries, entry)
	}

	mb.subscribersMtx.RLock()
	for _, sub := range mb.subscribers {
		select {
		case sub <- entry:
		default:
		}
	}
	mb.subscribersMtx.RUnlock()
	return nil
}

// Amend an old entry
func (mb *MemoryBackend) AmendEntry(entry *StreamEntry, oldTimestamp time.Time) error {
	_, idx := mb.findClosest(oldTimestamp)
	if idx == -1 {
		return nil
	}

	mb.Entries[idx] = entry
	return nil
}

func (mb *MemoryBackend) EntryAdded(ch chan<- *StreamEntry) {
	if ch == nil {
		return
	}

	mb.subscribersMtx.Lock()
	defer mb.subscribersMtx.Unlock()

	mb.subscribers = append(mb.subscribers, ch)
}

func (mb *MemoryBackend) ForEachEntry(cb func(entry *StreamEntry) error) error {
	mb.EntriesMtx.RLock()
	defer mb.EntriesMtx.RUnlock()

	for _, ent := range mb.Entries {
		if err := cb(ent); err != nil {
			return err
		}
	}
	return nil
}
