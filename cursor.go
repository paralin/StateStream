package stream

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/paralin/mutate"
)

//go:generate stringer -type=CursorType
type CursorType int

const (
	WriteCursor CursorType = iota
	ReadForwardCursor
	ReadBidirectionalCursor
)

var NoDataError error = errors.New("No data for that timestamp.")

// A cursor at a given point
type Cursor struct {
	storage StorageBackend

	// Do we have a state calculated at the desired timestamp?
	ready bool

	// is there a reason why we aren't ready?
	notReadyError error

	// Mutex for ready
	computeMutex sync.Mutex

	// Current desired time of the cursor
	timestamp time.Time

	// Last snapshot before current timestamp
	lastSnapshot *StreamEntry

	// Optimization: remember if we have a next snapshot.
	nextSnapshot *StreamEntry

	// Computed state at current timestamp
	computedState *StateDataPtr

	// Timestamp we computed the state at
	computedTimestamp time.Time

	// Type of the cursor
	cursorType CursorType

	// If we're a rewindable cursor (readbidirectional)
	lastMutations []*StreamEntry

	// If we're a write cursor
	lastMutation *StreamEntry

	// Last state before lastMutation
	lastState *StateDataPtr

	// Possibly known rate config
	rateConfig *RateConfig
}

func newCursor(storage StorageBackend, cursorType CursorType) *Cursor {
	return &Cursor{
		storage:       storage,
		cursorType:    cursorType,
		computeMutex:  sync.Mutex{},
		ready:         false,
		lastMutations: make([]*StreamEntry, 0),
	}
}

// Initializes a cursor at a timestamp.
// Timestamp is optional for a write cursor.
func (c *Cursor) Init(timestamp time.Time) error {
	if c.cursorType == WriteCursor {
		// bypass SetTimestamp limitations
		c.ready = false
		c.timestamp = time.Now()
	} else {
		c.SetTimestamp(timestamp)
	}
	return c.ComputeState()
}

// The cursor can do some internal optimizations if it knows the rate config.
func (c *Cursor) SetRateConfig(config *RateConfig) {
	if config.Validate() == nil {
		c.rateConfig = config
	} else {
		c.rateConfig = nil
	}
}

// Get the computed state
func (c *Cursor) State() (StateData, error) {
	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()
	if !c.ready {
		return nil, errors.New("Computation is not ready.")
	}
	return c.computedState.StateData, nil
}

func (c *Cursor) SetTimestamp(timestamp time.Time) {
	// It doesn't make sense to set the timestamp on a write cursor
	if c.cursorType == WriteCursor {
		return
	}

	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()

	if c.ready && timestamp.Equal(c.timestamp) {
		return
	}

	c.ready = false
	c.timestamp = timestamp
	if c.lastSnapshot != nil {
		if c.lastSnapshot.Timestamp.After(timestamp) {
			// force a recomputation of snapshot + state
			c.lastSnapshot = nil
			c.lastMutations = make([]*StreamEntry, 0)
			c.computedState = nil
		}
	}

	// Check if it's possible to use the current stored data at all
	if c.computedState != nil {
		// If we have to rewind...
		if c.computedTimestamp.After(timestamp) {
			// We can't rewind a read forward cursor
			if c.cursorType == ReadForwardCursor {
				c.computedState = nil
			}
		}
		// It should be fine to feed forward in any case
	}
}

// Getter for timestamp
func (c *Cursor) Timestamp() time.Time {
	return c.timestamp
}

// Actual timestamp of state
func (c *Cursor) ComputedTimestamp() time.Time {
	return c.computedTimestamp
}

// Getter for cursor type
func (c *Cursor) GetCursorType() CursorType {
	return c.cursorType
}

// If there was an error with the last computation, return it
func (c *Cursor) Error() error {
	return c.notReadyError
}

// Check if cursor is ready. Waits till any active computation is done.
func (c *Cursor) Ready() bool {
	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()
	return c.ready
}

// Finds the last snapshot field.
func (c *Cursor) fillLastSnapshot() error {
	data, err := c.storage.GetSnapshotBefore(c.timestamp)
	if err != nil {
		return err
	}
	if data == nil {
		return NoDataError
	}
	if data.Type != StreamEntrySnapshot {
		return errors.New("Storage backend didn't return a snapshot for GetSnapshotBefore()")
	}
	if data.Timestamp.After(c.timestamp) {
		return errors.New("Storage backend returned a snapshot after the requested timestamp.")
	}
	c.lastSnapshot = data
	if c.lastSnapshot != nil && data.Timestamp.Equal(c.lastSnapshot.Timestamp) {
		// don't clear mutations, we're using the last one again
		return nil
	}
	c.lastMutations = make([]*StreamEntry, 0)
	c.computedState = nil
	return nil
}

func (c *Cursor) fillNextSnapshot() error {
	// If we don't have a last snapshot, we can't have a next one.
	if c.lastSnapshot == nil {
		c.nextSnapshot = nil
		return nil
	}

	// If the expected next snapshot is after now, we *shouldn't* have a next one.
	// If the span of time was configured differently, we might.
	// This is still fine as NextEntry will return it.
	if c.rateConfig != nil {
		expectedNext := c.lastSnapshot.Timestamp.Add(time.Duration(c.rateConfig.KeyframeFrequency) * time.Millisecond)
		if expectedNext.After(time.Now()) {
			c.nextSnapshot = nil
			return nil
		}
	}

	// Do the db hit
	snap, err := c.storage.GetEntryAfter(c.lastSnapshot.Timestamp, StreamEntrySnapshot)
	if err != nil {
		return err
	}
	if snap != nil && snap.Type != StreamEntrySnapshot {
		return errors.New("Storage backend returned the wrong entry type.")
	}
	c.nextSnapshot = snap
	return nil
}

// Rewinds the state
// Assumptions:
//  - this is a rewindable cursor
//  - there is an old state set (computedState)
//  - the requested timestamp is after lastSnapshot
//  - the lastSnapshot is filled and BEFORE target timestamp
//  - the mutations array contains every mutation between snapshot and target
//  - the last mutation in the mutations array is the latest mutation before the old state
func (c *Cursor) rewindState() (err error) {
	defer func() {
		// If we failed, we probably messed up the computed state
		if err != nil {
			c.computedState = nil
		} // else {
		//	c.computedTimestamp = c.timestamp
		//}
	}()

	idx := len(c.lastMutations) - 1
	defer func() {
		if len(c.lastMutations) == 0 {
			return
		}
		if idx < 0 {
			c.lastMutations = make([]*StreamEntry, 0)
		} else {
			c.lastMutations = c.lastMutations[0 : idx+1]
		}
	}()

	// If we have no mutations, just use the snapshot
	if idx < 0 || c.lastMutations[0].Timestamp.After(c.timestamp) {
		if err := c.copySnapshotState(); err != nil {
			return err
		}
		if idx != 0 {
			idx = -1
		}
		return nil
	}

	if c.lastMutations[idx].Timestamp.Before(c.timestamp) {
		// we don't have to do anything
		return nil
	}

	// Apply mutations backwards until next mutation is before target.
	// Our checks above guerantee this has a base case, and idx will never be < 0.
	for !c.lastMutations[idx].Timestamp.Before(c.timestamp) {
		mutation := c.lastMutations[idx]
		newObj, err := mutate.ApplyMutationObject(c.computedState.StateData, mutation.Data)
		// If this happens it's a bug in mutate
		if err != nil {
			return err
		}
		c.computedState.StateData = newObj
		idx--
	}

	return nil
}

func (c *Cursor) applyMutation(mutation *StreamEntry) (err error) {
	var beforeObj *StateDataPtr
	if c.cursorType == ReadBidirectionalCursor || c.cursorType == WriteCursor {
		var err error
		beforeObj, err = c.computedState.Clone()
		if err != nil {
			return err
		}
	}

	stateAfter, err := mutate.ApplyMutationObject(c.computedState.StateData, mutation.Data)
	if err != nil {
		return err
	}

	if c.cursorType == ReadBidirectionalCursor {
		var beforeData interface{}
		beforeData = map[string]interface{}(beforeObj.StateData)
		c.lastMutations = append(c.lastMutations, &StreamEntry{
			Type:      StreamEntryMutation,
			Data:      mutate.BuildMutation(stateAfter, beforeData.(map[string]interface{})),
			Timestamp: mutation.Timestamp,
		})
	} else if c.cursorType == WriteCursor {
		c.lastMutation = mutation
		c.lastState = beforeObj
	}

	c.computedState = &StateDataPtr{StateData: stateAfter}
	c.computedTimestamp = mutation.Timestamp
	return nil
}

// Fast-forwards the state
// Assumptions:
//  - the lastSnapshot is filled and BEFORE target timestamp
//  - there is an old state set (computedState) at computedTimestamp
func (c *Cursor) fastForwardState() (err error) {
	defer func() {
		if err != nil {
			c.computedState = nil
		} // else {
		//	c.computedTimestamp = c.timestamp
		//}
	}()

	for c.computedTimestamp.Before(c.timestamp) {
		entry, err := c.storage.GetEntryAfter(c.computedTimestamp, StreamEntryAny)
		if err != nil {
			return err
		}
		if entry == nil {
			break
		}
		if entry.Timestamp.Before(c.computedTimestamp) {
			return errors.New("Storage backend returned an entry before requested time.")
		}
		if entry.Timestamp.After(c.timestamp) {
			if entry.Type == StreamEntrySnapshot {
				c.nextSnapshot = entry
			}
			break
		}
		if entry.Type == StreamEntryMutation {
			if err := c.applyMutation(entry); err != nil {
				return err
			}
		} else if entry.Type == StreamEntrySnapshot {
			c.lastSnapshot = entry
			c.nextSnapshot = nil
			if err := c.copySnapshotState(); err != nil {
				return err
			}
			if err := c.fillNextSnapshot(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Copies snapshot state to current state
func (c *Cursor) copySnapshotState() error {
	preClone := &StateDataPtr{StateData: c.lastSnapshot.Data}
	postClone, err := preClone.Clone()
	if err != nil {
		return err
	}
	c.computedState = postClone
	c.computedTimestamp = c.lastSnapshot.Timestamp
	c.lastMutation = nil
	c.lastState = postClone
	c.lastMutations = []*StreamEntry{}
	return nil
}

// Force the cursor to re-compute next time.
func (c *Cursor) Invalidate() {
	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()

	c.ready = false
}

func (c *Cursor) canHandleNewEntry(timestamp time.Time) error {
	if c.cursorType != WriteCursor || !c.ready {
		return errors.New("Cursor is not ready, or is not a write cursor.")
	}
	// We can't really do anything in this situation
	if c.lastState != nil && timestamp.Before(c.computedTimestamp) {
		return errors.New("Entry is before the latest entry, we can't handle this.")
	}
	return nil
}

// Handle a state entry on a writer (keeps writer up to date)
func (c *Cursor) HandleEntry(entry *StreamEntry) error {
	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()

	if err := c.canHandleNewEntry(entry.Timestamp); err != nil {
		return err
	}

	if entry.Type == StreamEntrySnapshot {
		c.lastSnapshot = entry
		return c.copySnapshotState()
	} else {
		return c.applyMutation(entry)
	}
}

// Writes a state to the end of the stream.
func (c *Cursor) WriteState(timestamp time.Time, state StateData, config *RateConfig) (writeError error) {
	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()

	if c.lastState != nil && reflect.DeepEqual(c.lastState.StateData, state) {
		return nil
	}

	if err := c.canHandleNewEntry(timestamp); err != nil {
		return err
	}

	inputState := CloneStateData(state)

	var lastChange time.Time
	if c.lastMutation == nil {
		if c.lastSnapshot != nil {
			lastChange = c.lastSnapshot.Timestamp
		}
	} else {
		lastChange = c.lastMutation.Timestamp
	}
	if c.lastState != nil && timestamp.Before(lastChange) {
		return errors.New("Cannot write entry before last change.")
	}

	// If the last thing that changed was a mutation and it's too soon to make a new mutation
	if c.lastMutation != nil && timestamp.Sub(c.lastMutation.Timestamp) < (time.Duration(config.ChangeFrequency)*time.Millisecond) {
		amendedMutation := &StreamEntry{
			Type:      StreamEntryMutation,
			Timestamp: c.lastMutation.Timestamp,
		}

		// Calculate the new mutation
		amendedMutation.Data = mutate.BuildMutation(c.lastState.StateData, inputState.StateData)
		if err := c.storage.AmendEntry(amendedMutation, c.lastMutation.Timestamp); err != nil {
			return err
		}

		// Apply the new state
		c.computedState = inputState
		c.computedTimestamp = timestamp
		c.lastMutation = amendedMutation
		return nil
	}

	// Check if we should make a new snapshot
	if c.lastState == nil || timestamp.Sub(c.lastSnapshot.Timestamp) >= (time.Duration(config.KeyframeFrequency)*time.Millisecond) {
		// Make a new snapshot
		snapshot := &StreamEntry{
			Type:      StreamEntrySnapshot,
			Data:      inputState.StateData,
			Timestamp: timestamp,
		}

		if err := c.storage.SaveEntry(snapshot); err != nil {
			return err
		}

		c.lastSnapshot = snapshot
		c.lastMutation = nil
		if err := c.copySnapshotState(); err != nil {
			c.ready = false
			c.computedState = nil
			return err
		}
		return nil
	}

	// Make a new mutation
	oldState, err := c.computedState.Clone()
	if err != nil {
		return err
	}

	newMutationEntry := &StreamEntry{
		Type:      StreamEntryMutation,
		Timestamp: timestamp,
		Data:      mutate.BuildMutation(c.lastState.StateData, inputState.StateData),
	}

	if err := c.storage.SaveEntry(newMutationEntry); err != nil {
		return err
	}

	c.lastMutation = newMutationEntry
	c.lastState = oldState
	c.computedState = inputState
	c.computedTimestamp = timestamp
	return nil
}

// Make a cursor become ready, or return the error
func (c *Cursor) ComputeState() (computeErr error) {
	c.computeMutex.Lock()
	defer c.computeMutex.Unlock()
	if c.ready {
		return nil
	}

	defer func() {
		// If we don't have data, and are a write cursor, don't treat it as an error.
		if computeErr == NoDataError && c.cursorType == WriteCursor {
			c.ready = true
			c.computedState = NewStateData()
			c.computedTimestamp = c.timestamp
			computeErr = nil
			c.lastState = nil
		} else {
			c.ready = computeErr == nil
		}
		c.notReadyError = computeErr
	}()

	// Fill the last snapshot if needed
	if c.lastSnapshot == nil {
		if err := c.fillLastSnapshot(); err != nil {
			return err
		}
		if err := c.fillNextSnapshot(); err != nil {
			return err
		}
	}

	var err error = nil

	if c.timestamp.Equal(c.lastSnapshot.Timestamp) {
		err = c.copySnapshotState()
		c.nextSnapshot = nil
	} else if c.computedState != nil {
		// If we have a computed state, we can move it to the target state
		// This is enforced in SetTimestamp()

		// We need to rewind the cursor
		if c.computedTimestamp.After(c.timestamp) {
			err = c.rewindState()
		} else {
			// We need to fast-forward the cursor
			// First fast forward the snapshot
			if c.nextSnapshot != nil && c.nextSnapshot.Timestamp.Before(c.timestamp) {
				c.lastSnapshot = c.nextSnapshot
				c.nextSnapshot = nil
				if err := c.fillNextSnapshot(); err != nil {
					return err
				}
				// If the next snapshot is STILL before the timestamp
				if c.nextSnapshot.Timestamp.Before(c.timestamp) {
					// Fast forward
					c.lastSnapshot = nil
					c.nextSnapshot = nil
					if err := c.fillLastSnapshot(); err != nil {
						return err
					}
					if err := c.fillNextSnapshot(); err != nil {
						return err
					}
				}
			}
			err = c.fastForwardState()
		}
	} else {
		c.computedTimestamp = c.lastSnapshot.Timestamp
		c.computedState = &StateDataPtr{StateData: c.lastSnapshot.Data}
		err = c.copySnapshotState()
		if err == nil {
			err = c.fastForwardState()
		}
	}

	return err
}
