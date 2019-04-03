package macutils

import (
	"fmt"
	"sync"
)

/*
#cgo LDFLAGS: -framework Carbon -framework CoreFoundation

extern int runKeyHandler(int);
*/
import "C"

const (
	KeyEject = 0x92 // TODO(samthor): doesn't seem to work
	KeyF12   = 0x6F
	KeyF13   = 0x69
)

var (
	globalMutex sync.Mutex
	globalCh    = make(chan interface{})
	globalDone  bool
)

//export keyGoCallback
func keyGoCallback() {
	globalCh <- nil
}

// ListenForKey listens globally for the specified keycode.
func ListenForKey(key int) (chan int, error) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalDone {
		return nil, fmt.Errorf("TODO: can't listen for multiple keys")
	}
	globalDone = true

	go func() {
		ret := C.runKeyHandler(C.int(key))
		if ret != 0 {
			panic("could not configure keyHandler")
		}
	}()

	ret := make(chan int)
	go func() {
		for {
			<-globalCh
			ret <- key
		}
	}()
	return ret, nil
}
