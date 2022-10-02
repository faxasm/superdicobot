package userpool

import (
	"strconv"
	"sync"
	"time"
)

type cmdPool struct {
	value     string
	lastValid int64
}

type TTLCmdMap struct {
	m map[string]*cmdPool
	l sync.Mutex
}

func NewCmdPool(ln int) (m *TTLCmdMap) {
	m = &TTLCmdMap{
		m: make(map[string]*cmdPool, ln),
	}

	go func() {
		for now := range time.Tick(time.Second) {
			m.l.Lock()
			for k, v := range m.m {
				if now.Unix()-v.lastValid > 0 {
					delete(m.m, k)
				}
			}
			m.l.Unlock()
		}
	}()
	return
}

func (m *TTLCmdMap) Len() int {
	return len(m.m)
}

func (m *TTLCmdMap) Put(k, v string, lastValid int64) {
	m.l.Lock()
	it, ok := m.m[k]
	if !ok {
		it = &cmdPool{value: v}
		m.m[k] = it
	}
	it.lastValid = lastValid
	m.l.Unlock()
}

func (m *TTLCmdMap) Get(k string) (v string) {
	m.l.Lock()
	if it, ok := m.m[k]; ok {
		v = it.value
	}
	m.l.Unlock()
	return
}

func (m *TTLCmdMap) Display() (v string) {
	m.l.Lock()
	for i, k := range m.m {
		timeRemaining := k.lastValid - time.Now().Unix()
		v = v + i + "->" + strconv.FormatInt(timeRemaining, 10) + "s || "
	}
	m.l.Unlock()
	return
}

func (m *TTLCmdMap) Clear() {
	m.l.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	m.l.Unlock()
	return
}
