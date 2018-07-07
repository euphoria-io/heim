package scope

// Breakpointer provides a pair of methods for synchronizing across
// goroutines and injecting errors. The Check method can be used
// to provide a point of synchronization/injection. In normal operation,
// this method will quickly return nil. A unit test can then use
// Breakpoint, with the same parameters, to obtain a bidirectional
// error channel. Receiving from this channel will block until Check
// is called. The call to Check will block until an error value (or nil)
// is sent back into the channel.
type Breakpointer interface {
	// Breakpoint returns an error channel that can be used to synchronize
	// with a call to Check with the exact same parameters from another
	// goroutine. The call to Check will send a nil value across this
	// channel, and then receive a value to return to its caller.
	Breakpoint(scope ...interface{}) chan error

	// Check synchronizes with a registered breakpoint to obtain an error
	// value to return, or immediately returns nil if no breakpoint is
	// registered.
	Check(scope ...interface{}) error
}

type breakpoint struct {
	c chan error
}

type bpmap kvmap

func (b bpmap) get(create bool, scope ...interface{}) chan error {
	switch len(scope) {
	case 0:
		return nil
	case 1:
		if obj, ok := b[scope[0]]; ok {
			if ch, ok := obj.(chan error); ok {
				return ch
			}
			return nil
		}
		if !create {
			return nil
		}
		ch := make(chan error)
		b[scope[0]] = ch
		return ch
	default:
		if obj, ok := b[scope[0]]; ok {
			if bpm, ok := obj.(bpmap); ok {
				return bpm.get(create, scope[1:]...)
			}
			return nil
		}
		bpm := bpmap{}
		b[scope[0]] = bpm
		return bpm.get(create, scope[1:]...)
	}
}
