package stream

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestSimpleStreamWrite(t *testing.T) {
	storageMock := &MockStorageBackend{Entries: []*StreamEntry{}}
	stream, err := NewStream(storageMock, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	cursor, err := stream.WriteCursor()
	if err != nil {
		t.Fatalf(err.Error())
	}
	ch := make(chan *StreamEntry, 100)
	sub := cursor.SubscribeEntries(ch)

	now := time.Now()
	if err := CheckWriteState(stream, `{"test":1}`, now); err != nil {
		t.Fatalf(err.Error())
	}

	// Go 1.2 second later (should make new mutation)
	now = now.Add(time.Millisecond * time.Duration(1200))
	if err := CheckWriteState(stream, `{"test":3, "test2": 4}`, now); err != nil {
		t.Fatalf(err.Error())
	}
	if len(storageMock.Entries) != 2 || storageMock.Entries[1].Type != StreamEntryMutation {
		t.Fatalf("Did not store in storage correctly.")
	}

	// Go 10 ms later (should amend the last mutation)
	now = now.Add(time.Millisecond * time.Duration(10))
	if err := CheckWriteState(stream, `{"test":3,"test2":{"yes":false}}`, now); err != nil {
		t.Fatalf(err.Error())
	}
	if len(storageMock.Entries) != 2 || storageMock.Entries[1].Type != StreamEntryMutation {
		t.Fatalf("Did not store in storage correctly.")
	}

	sub.Unsubscribe()

	// Go 120 second later (should make new snapshot)
	now = now.Add(time.Second * time.Duration(120))
	if err := CheckWriteState(stream, `{"test":3, "test2": 4, "test3": 5}`, now); err != nil {
		t.Fatalf(err.Error())
	}
	if len(storageMock.Entries) != 3 || storageMock.Entries[2].Type != StreamEntrySnapshot {
		t.Fatalf("Did not store in storage correctly.")
	}

	if len(ch) != 3 {
		t.Fatalf("Subscribing to write cursor had %d != 3 entries.", len(ch))
	}
}

func TestStreamEntryWrite(t *testing.T) {
	storageMock := &MockStorageBackend{Entries: []*StreamEntry{}}
	stream, err := NewStream(storageMock, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	now := time.Now()
	if err := CheckWriteMutation(stream, `{"test":1}`, `{"test":1}`, now); err != nil {
		t.Fatalf(err.Error())
	}

	// Go 1.2 second later (should make new mutation)
	now = now.Add(time.Millisecond * time.Duration(1200))
	if err := CheckWriteMutation(stream, `{"test":3,"test2":{"$set":4}}`, `{"test":3,"test2":4}`, now); err != nil {
		t.Fatalf(err.Error())
	}
	if len(storageMock.Entries) != 2 || storageMock.Entries[1].Type != StreamEntryMutation {
		t.Fatalf("Did not store in storage correctly.")
	}

	// Go 10 ms later (should amend the last mutation)
	now = now.Add(time.Millisecond * time.Duration(10))
	if err := CheckWriteMutation(stream, `{"test":23,"test2":{"$unset":true}}`, `{"test":23}`, now); err != nil {
		t.Fatalf(err.Error())
	}
	if len(storageMock.Entries) != 2 || storageMock.Entries[1].Type != StreamEntryMutation {
		t.Fatalf("Did not store in storage correctly.")
	}

	// Go 120 second later (should make new snapshot)
	now = now.Add(time.Second * time.Duration(120))
	if err := CheckWriteMutation(stream, `{"test":{"$unset":true}}`, `{}`, now); err != nil {
		t.Fatalf(err.Error())
	}
	if len(storageMock.Entries) != 3 || storageMock.Entries[2].Type != StreamEntrySnapshot {
		t.Fatalf("Did not store in storage correctly.")
	}
}

func CheckWriteState(stream *Stream, state string, timestamp time.Time) error {
	stateData, err := NewStateDataFromJson([]byte(state))
	if err != nil {
		return err
	}
	stateDataBak, err := stateData.Clone()
	if err != nil {
		return err
	}
	if err := stream.WriteState(timestamp, stateData.StateData); err != nil {
		return err
	}
	cursor, err := stream.WriteCursor()
	if err != nil {
		return err
	}
	storedStateData, err := cursor.State()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(stateDataBak.StateData, storedStateData) {
		return fmt.Errorf("State was not stored properly. Expected %v != %v", stateData.StateData, storedStateData)
	}
	return nil
}

func CheckWriteMutation(stream *Stream, state, expected string, timestamp time.Time) error {
	stateData, err := NewStateDataFromJson([]byte(state))
	if err != nil {
		return err
	}
	expectedStateData, err := NewStateDataFromJson([]byte(expected))
	if err != nil {
		return err
	}
	entry := &StreamEntry{
		Timestamp: timestamp,
		Data:      stateData.StateData,
		Type:      StreamEntryMutation,
	}
	if err := stream.WriteEntry(entry); err != nil {
		return err
	}
	cursor, err := stream.WriteCursor()
	if err != nil {
		return err
	}
	storedStateData, err := cursor.State()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expectedStateData.StateData, storedStateData) {
		return fmt.Errorf("State was not stored properly. Expected %v != %v", expectedStateData.StateData, storedStateData)
	}
	return nil
}
