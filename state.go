package stream

import (
	"encoding/json"
	"time"
)

type StateData map[string]interface{}

//go:generate stringer -type=StreamEntryType
type StreamEntryType int

const (
	StreamEntrySnapshot StreamEntryType = iota
	StreamEntryMutation
	// used when filtering stream entries
	StreamEntryAny
)

// An entry in the stream.
type StreamEntry struct {
	Timestamp time.Time
	Type      StreamEntryType
	Data      StateData
}

type StateDataPtr struct {
	StateData
}

func NewStateDataPtr(data StateData) *StateDataPtr {
	return &StateDataPtr{StateData: data}
}

func NewStateData() *StateDataPtr {
	return NewStateDataPtr(make(map[string]interface{}))
}

func CloneStateData(data StateData) *StateDataPtr {
	ptr := NewStateDataPtr(data)
	res, _ := ptr.Clone()
	return res
}

func NewStateDataFromJson(jsonData []byte) (*StateDataPtr, error) {
	var stateData map[string]interface{}
	err := json.Unmarshal(jsonData, &stateData)
	if err != nil {
		return nil, err
	}
	return &StateDataPtr{StateData: StateData(stateData)}, nil
}

// Come up with a better way to clone
func (dp *StateDataPtr) Clone() (*StateDataPtr, error) {
	data, err := json.Marshal(&dp.StateData)
	if err != nil {
		return nil, err
	}
	return NewStateDataFromJson(data)
}
