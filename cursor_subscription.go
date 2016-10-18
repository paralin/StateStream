package stream

type cursorEntrySubscription struct {
	unsubFunc func()
}

func (s *cursorEntrySubscription) Unsubscribe() {
	s.unsubFunc()
}
