package stream

import (
	"testing"
	"time"
)

var TestBaseTime = time.Now()

func MockEntries() []*StreamEntry {
	res := []*StreamEntry{}
	for i := 0; i < 10; i++ {
		typ := StreamEntryMutation
		if i == 0 || i == 5 {
			typ = StreamEntrySnapshot
		}
		res = append(res, &StreamEntry{
			Data: StateData{
				"test": i + 1,
			},
			Timestamp: TestBaseTime.Add(time.Duration(-10+i) * time.Second),
			Type:      typ,
		})
	}
	return res
}

func TestEntryAfter(t *testing.T) {
	mb := &MemoryBackend{Entries: MockEntries()}
	se, _ := mb.GetEntryAfter(
		TestBaseTime.Add(time.Duration(-9)*time.Second),
		StreamEntrySnapshot)
	if se == nil {
		t.Fail()
	}
	if se.Data["test"].(int) != 6 {
		t.Fail()
	}
}

func TestSnapshotBefore(t *testing.T) {
	mb := &MemoryBackend{Entries: MockEntries()}
	se, _ := mb.GetSnapshotBefore(TestBaseTime.Add(time.Duration(-9) * time.Second))
	if se == nil {
		t.Fail()
	}
	if se.Data["test"].(int) != 1 {
		t.Fail()
	}
}
