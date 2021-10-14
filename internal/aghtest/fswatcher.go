package aghtest

// FSWatcher is a mock aghos.FSWatcher implementation to use in tests.
type FSWatcher struct {
	OnEvents func() (e <-chan struct{})
	OnAdd    func(name string) (err error)
	OnClose  func() (err error)
}

// Events implements the aghos.FSWatcher interface for *FSWatcher.
func (w *FSWatcher) Events() (e <-chan struct{}) {
	return w.OnEvents()
}

// Add implements the aghos.FSWatcher interface for *FSWatcher.
func (w *FSWatcher) Add(name string) (err error) {
	return w.OnAdd(name)
}

// Close implements the aghos.FSWatcher interface for *FSWatcher.
func (w *FSWatcher) Close() (err error) {
	return w.OnClose()
}
