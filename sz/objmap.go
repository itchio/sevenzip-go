package sz

import (
	"sync/atomic"
	"unsafe"

	"github.com/cornelk/hashmap"
)

var seed int64 = 1

//==========================
// OutStream
//==========================

var outStreams = &hashmap.HashMap{}

func reserveOutStreamId(obj *OutStream) {
	obj.id = atomic.AddInt64(&seed, 1)
	outStreams.Set(obj.id, unsafe.Pointer(obj))
}

func freeOutStreamId(id int64) {
	outStreams.Del(id)
}

//==========================
// InStream
//==========================

var inStreams = &hashmap.HashMap{}

func reserveInStreamId(obj *InStream) {
	obj.id = atomic.AddInt64(&seed, 1)
	inStreams.Set(obj.id, unsafe.Pointer(obj))
}

func freeInStreamId(id int64) {
	inStreams.Del(id)
}

//==========================
// ExtractCallback
//==========================

var extractCallbacks = &hashmap.HashMap{}

func reserveExtractCallbackId(obj *ExtractCallback) {
	obj.id = atomic.AddInt64(&seed, 1)
	extractCallbacks.Set(obj.id, unsafe.Pointer(obj))
}

func freeExtractCallbackId(id int64) {
	extractCallbacks.Del(id)
}
