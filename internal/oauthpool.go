package userpool

import (
	"strconv"
	"sync"
	"time"
)

type oauthPool struct {
	value     string
	lastValid int64
}

type TTLOauthMap struct {
	m map[string]*oauthPool
	l sync.Mutex
}

func NewOauthMap(ln int) (m *TTLOauthMap) {
	m = &TTLOauthMap{
		m: make(map[string]*oauthPool, ln),
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

func (m *TTLOauthMap) Len() int {
	return len(m.m)
}

func (m *TTLOauthMap) Put(k, v string, lastValid int64) {
	m.l.Lock()
	it, ok := m.m[k]
	if !ok {
		it = &oauthPool{value: v}
		m.m[k] = it
	}
	it.lastValid = lastValid
	m.l.Unlock()
}

func (m *TTLOauthMap) Get(k string) (v string) {
	m.l.Lock()
	if it, ok := m.m[k]; ok {
		v = it.value
	}
	m.l.Unlock()
	return
}

func (m *TTLOauthMap) Display() (v string) {
	m.l.Lock()
	for i, k := range m.m {
		timeRemaining := k.lastValid - time.Now().Unix()
		v = v + i + "->" + strconv.FormatInt(timeRemaining, 10) + "s || "
	}
	m.l.Unlock()
	return
}

func (m *TTLOauthMap) Clear() {
	m.l.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	m.l.Unlock()
	return
}
