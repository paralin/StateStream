package stream

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// Mock backend
type MockStorageBackend struct {
	// Expecting these to be ordered by timestamp
	Entries []*StreamEntry
}

func (sb *MockStorageBackend) GetSnapshotBefore(timestamp time.Time) (*StreamEntry, error) {
	for _, entry := range sb.Entries {
		if entry.Timestamp.After(timestamp) {
			break
		}
		if entry.Type == StreamEntrySnapshot {
			return entry, nil
		}
	}
	return nil, nil
}

func (sb *MockStorageBackend) GetMutationAfter(timestamp time.Time) (*StreamEntry, error) {
	for _, entry := range sb.Entries {
		if !entry.Timestamp.After(timestamp) {
			continue
		}
		if entry.Type == StreamEntryMutation {
			return entry, nil
		}
	}
	return nil, nil
}

func (sb *MockStorageBackend) SaveEntry(entry *StreamEntry) error {
	sb.Entries = append(sb.Entries, entry)
	return nil
}

func (sb *MockStorageBackend) AmendEntry(entry *StreamEntry, timestamp time.Time) error {
	for idx, oldentry := range sb.Entries {
		if oldentry.Timestamp.Equal(timestamp) {
			sb.Entries[idx] = entry
			return nil
		}
	}
	return errors.New("Entry not found.")
}

func TestSetTimestampBeforeSnapshot(t *testing.T) {
	// snapshot is at now
	// target timestamp is at snapshot - 1 second
	now := time.Now()
	cursor := &Cursor{
		ready:             true,
		cursorType:        ReadBidirectionalCursor,
		computeMutex:      sync.Mutex{},
		timestamp:         now,
		computedTimestamp: now,
		computedState:     NewStateData(),
		lastSnapshot: &StreamEntry{
			Timestamp: now,
			Type:      StreamEntrySnapshot,
			Data:      NewStateData().StateData,
		},
	}

	cursor.SetTimestamp(now.Add(-time.Duration(1) * time.Second))
	if cursor.ready || cursor.lastSnapshot != nil || len(cursor.lastMutations) > 0 {
		t.Fatalf("Snapshot was not cleared when timestamp set before it.")
	}
}

func TestSetSameTimestamp(t *testing.T) {
	now := time.Now()
	cursor := &Cursor{
		cursorType:        ReadForwardCursor,
		ready:             true,
		timestamp:         now,
		computedTimestamp: now,
		computedState:     NewStateData(),
	}

	cursor.SetTimestamp(now)
	if !cursor.ready {
		t.Fatalf("Cursor should still be ready after setting same timestamp.")
	}
}

func TestSetTimestampOnWriteCursor(t *testing.T) {
	now := time.Now()
	cursor := &Cursor{
		cursorType:        WriteCursor,
		ready:             true,
		timestamp:         now,
		computedTimestamp: now,
		computedState:     NewStateData(),
	}

	cursor.SetTimestamp(now.Add(time.Duration(5) * time.Second))
	if !cursor.ready {
		t.Fatalf("SetTimestamp() on write cursor should do nothing.")
	}
}

func TestSimpleBidirectionalCursor(t *testing.T) {
	now := time.Now()
	storageMock := &MockStorageBackend{
		Entries: []*StreamEntry{
			&StreamEntry{
				Type:      StreamEntrySnapshot,
				Timestamp: now.Add(-time.Duration(5) * time.Second),
				Data: map[string]interface{}{
					"test": []interface{}{
						"before",
					},
				},
			},
			&StreamEntry{
				Type:      StreamEntryMutation,
				Timestamp: now,
				Data: map[string]interface{}{
					"test": map[string]interface{}{
						"$mutateIdx": map[string]interface{}{
							"0": "after",
						},
					},
				},
			},
		},
	}
	cursor := &Cursor{
		cursorType:   ReadBidirectionalCursor,
		storage:      storageMock,
		computeMutex: sync.Mutex{},
	}
	if err := cursor.Init(now.Add(-time.Duration(1) * time.Second)); err != nil {
		t.Fatalf(err.Error())
	}
	if !cursor.Ready() {
		t.Fatalf("Simple bidirectional cursor failed.")
	}
	if data, _ := cursor.State(); data["test"].([]interface{})[0] != "before" {
		t.Fatalf("Data %s unexpected.", data["test"])
	}
	cursor.SetTimestamp(now.Add(time.Duration(5) * time.Second))
	if cursor.Ready() {
		t.Fatalf("Cursor shouldn't be ready after SetTimestamp")
	}
	if err := cursor.ComputeState(); err != nil {
		t.Fatalf(err.Error())
	}
	if data, _ := cursor.State(); data["test"].([]interface{})[0] != "after" {
		t.Fatalf("Data %s unexpected after compute.", data["test"])
	}

	// Now try to rewind it
	cursor.SetTimestamp(now.Add(-time.Duration(2) * time.Second))
	if cursor.Ready() {
		t.Fatalf("Cursor shouldn't be ready after SetTimestamp")
	}
	if err := cursor.ComputeState(); err != nil {
		t.Fatalf(err.Error())
	}
	if data, _ := cursor.State(); data["test"].([]interface{})[0] != "before" {
		t.Fatalf("Data %s unexpected after rewind compute.", data["test"])
	}
}

func TestReady(t *testing.T) {
	cursor := &Cursor{}
	readyChan := make(chan bool, 1)
	cursor.computeMutex.Lock()
	cursor.ready = false
	go func() {
		readyChan <- cursor.Ready()
	}()
	select {
	case <-readyChan:
		t.Fatalf("Ready should lock computeMutex")
	default:
		cursor.computeMutex.Unlock()
	}
	if val := <-readyChan; val != false {
		t.Fatalf("Ready() returned the wrong value.")
	}
}

func TestSimpleGetters(t *testing.T) {
	now := time.Now()
	cursor := &Cursor{cursorType: ReadForwardCursor, timestamp: now}
	if cursor.GetCursorType() != ReadForwardCursor || !now.Equal(cursor.Timestamp()) {
		t.Fatalf("Simple getters aren't working properly.")
	}
	cursor.notReadyError = errors.New("test")
	if err := cursor.Error(); err == nil || err.Error() != "test" {
		t.Fail()
	}
}

func TestSetOlderTimestamp(t *testing.T) {
	now := time.Now()
	cursor := &Cursor{
		cursorType:        ReadForwardCursor,
		ready:             true,
		timestamp:         now,
		computedTimestamp: now,
		computedState:     NewStateData(),
		lastSnapshot: &StreamEntry{
			Timestamp: now.Add(-time.Duration(10) * time.Second),
			Type:      StreamEntrySnapshot,
			Data:      NewStateData().StateData,
		},
	}

	cursor.SetTimestamp(now.Add(-time.Duration(1) * time.Second))
	if cursor.ready || cursor.computedState != nil {
		t.Fatalf("Feed-forward cursor cannot be rewound.")
	}
}

func TestRewindStateNoHistory(t *testing.T) {
	// snapshot is at now - 10 seconds
	// make no history
	// current state is at now
	// target timestamp is at snapshot + 1 second
	now := time.Now()
	cursor := &Cursor{
		cursorType:        ReadBidirectionalCursor,
		ready:             false,
		computeMutex:      sync.Mutex{},
		timestamp:         now.Add(-time.Duration(9) * time.Second),
		computedTimestamp: now,
		computedState:     NewStateData(),
		lastSnapshot: &StreamEntry{
			Timestamp: now.Add(-time.Duration(10) * time.Second),
			Type:      StreamEntrySnapshot,
			Data:      NewStateData().StateData,
		},
	}
	cursor.lastSnapshot.Data["test"] = "yes"
	if err := cursor.ComputeState(); err != nil {
		t.Fatalf(err.Error())
		t.Fail()
	}

	var err error
	data, err := cursor.State()
	if err == nil {
		err = CheckValidComputation(cursor)
	}
	if err == nil && (data["test"] != "yes") {
		err = errors.New("Cursor did not rewind properly - incorrect data.")
	}
	if err != nil {
		t.Fatalf(err.Error())
		t.Fail()
	}
}

func TestRewindStateSimple(t *testing.T) {
	// snapshot is at now - 10 seconds
	// mutations at now - 10 sec, now - 9 sec, etc.
	// current state is at now
	// target timestamp is at snapshot + 2.5 second
	now := time.Now()
	snapshotTime := now.Add(-time.Duration(10) * time.Second)
	cursor := &Cursor{
		cursorType:        ReadBidirectionalCursor,
		ready:             true,
		computeMutex:      sync.Mutex{},
		timestamp:         now,
		computedTimestamp: now,
		computedState:     NewStateData(),
		lastSnapshot: &StreamEntry{
			Timestamp: snapshotTime,
			Type:      StreamEntrySnapshot,
			Data:      NewStateData().StateData,
		},
	}
	cursor.computedState.StateData["test"] = "latest"
	cursor.lastSnapshot.Data["test"] = "veryold"
	cursor.lastMutations = make([]*StreamEntry, 5)

	makeEntry := func(secs int) *StreamEntry {
		return &StreamEntry{
			Type:      StreamEntryMutation,
			Timestamp: snapshotTime.Add(time.Duration(secs) * time.Second),
			Data:      NewStateData().StateData,
		}
	}

	// We rewind to 2.5 seconds after snapshot
	// We have a mutation at 1 second, 2 second after snapshot
	// -> we should have 2 mutations afterward
	cursor.lastMutations[0] = makeEntry(1)
	cursor.lastMutations[0].Data["test"] = "not expected"
	cursor.lastMutations[1] = makeEntry(2)
	cursor.lastMutations[2] = makeEntry(3)
	cursor.lastMutations[3] = makeEntry(4)
	cursor.lastMutations[3].Data["test"] = "expected"
	cursor.lastMutations[4] = makeEntry(5)

	cursor.SetTimestamp(now.Add(-time.Duration(7500) * time.Millisecond))
	if err := cursor.ComputeState(); err != nil {
		t.Fatalf(err.Error())
		t.Fail()
	}

	var err error
	data, err := cursor.State()
	if err == nil {
		err = CheckValidComputation(cursor)
	}
	if err == nil && (data["test"] != "expected") {
		err = errors.New("Cursor did not rewind properly - incorrect data.")
	}
	if err == nil && len(cursor.lastMutations) != 2 {
		err = errors.New("Cursor did not clear lastMutations properly.")
	}
	if err != nil {
		t.Fatalf(err.Error())
		t.Fail()
	}
}

func TestFastForwardStateSimple(t *testing.T) {
	// snapshot is at now - 10 seconds
	// mutations at now - 10 sec, now - 9 sec, etc.
	// current state is at now
	// target timestamp is at snapshot + 2.5 second
	now := time.Now()
	snapshotTime := now.Add(-time.Duration(10) * time.Second)
	storageMock := &MockStorageBackend{Entries: make([]*StreamEntry, 4)}
	cursor := &Cursor{
		storage:           storageMock,
		cursorType:        ReadForwardCursor,
		computeMutex:      sync.Mutex{},
		timestamp:         snapshotTime,
		computedTimestamp: snapshotTime,
		computedState:     NewStateData(),
		lastSnapshot: &StreamEntry{
			Timestamp: snapshotTime,
			Type:      StreamEntrySnapshot,
			Data:      NewStateData().StateData,
		},
	}
	cursor.computedState.StateData["test"] = "veryold"
	cursor.lastSnapshot.Data["test"] = "veryold"

	makeEntry := func(secs int) *StreamEntry {
		return &StreamEntry{
			Type:      StreamEntryMutation,
			Timestamp: snapshotTime.Add(time.Duration(secs) * time.Second),
			Data:      NewStateData().StateData,
		}
	}

	storageMock.Entries[0] = cursor.lastSnapshot
	storageMock.Entries[1] = makeEntry(1)
	storageMock.Entries[1].Data["test"] = "expected"
	storageMock.Entries[2] = makeEntry(4)
	storageMock.Entries[2].Data["test"] = "unexpected"
	storageMock.Entries[3] = makeEntry(5)
	storageMock.Entries[3].Data["test"] = "veryunexpected"
	storageMock.Entries[3].Type = StreamEntrySnapshot

	// Fast forward to before unexpected
	cursor.SetTimestamp(snapshotTime.Add(time.Duration(2) * time.Second))
	_ = "breakpoint"
	if err := cursor.ComputeState(); err != nil {
		t.Fatalf(err.Error())
		t.Fail()
	}

	var err error
	data, err := cursor.State()
	if err == nil {
		err = CheckValidComputation(cursor)
	}
	if err == nil && (data["test"] != "expected") {
		err = errors.New("Cursor did not fast forward properly - incorrect data.")
	}
	if err != nil {
		t.Fatalf(err.Error())
		t.Fail()
	}
}

func TestInvalidStateCall(t *testing.T) {
	cursor := &Cursor{ready: false}
	if _, err := cursor.State(); err == nil {
		t.Fail()
	}
}

func CheckValidComputation(cursor *Cursor) error {
	data, err := cursor.State()
	if err != nil {
		return err
	}
	// if !cursor.computedTimestamp.Equal(cursor.timestamp) {
	// 	return fmt.Errorf("Cursor did not rewind properly, computed timestamp %v != %v.", cursor.computedTimestamp, cursor.timestamp)
	// }
	if data == nil {
		return errors.New("Data is null in computation result.")
	}
	return nil
}
