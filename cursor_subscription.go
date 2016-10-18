package stream

type CursorEntrySubscription interface {
	Unsubscribe()
}

type cursorEntrySubscription struct {
	unsubFunc func()
}

func (s *cursorEntrySubscription) Unsubscribe() {
	s.unsubFunc()
}
