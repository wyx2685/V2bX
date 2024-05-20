package hy2

import (
	"sync"

	"github.com/InazumaV/V2bX/common/counter"
)

type HookServer struct {
	Tag     string
	Counter sync.Map
}

func (h *HookServer) Log(id string, tx, rx uint64) (ok bool) {
	var c interface{}
	var exists bool

	if c, exists = h.Counter.Load(h.Tag); !exists {
		c = counter.NewTrafficCounter()
		h.Counter.Store(h.Tag, c)
	}

	if tc, ok := c.(*counter.TrafficCounter); ok {
		tc.Rx(id, int(rx))
		tc.Tx(id, int(tx))
		return true
	}

	return false
}
