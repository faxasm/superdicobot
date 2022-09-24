package userpool

import (
	"github.com/gempir/go-twitch-irc/v3"
	"strconv"
	"sync"
	"time"
)

type userPool struct {
	value     string
	lastValid int64
}

type TTLMap struct {
	m       map[string]*userPool
	channel string
	client  *twitch.Client
	l       sync.Mutex
}

func New(ln int, channel string, client *twitch.Client) (m *TTLMap) {
	m = &TTLMap{
		m:       make(map[string]*userPool, ln),
		channel: channel,
		client:  client,
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

func (m *TTLMap) Len() int {
	return len(m.m)
}

func (m *TTLMap) Put(k, v string, lastValid int64) {
	m.l.Lock()
	it, ok := m.m[k]
	if !ok {
		it = &userPool{value: v}
		m.m[k] = it
	}
	it.lastValid = lastValid
	m.l.Unlock()
}

func (m *TTLMap) Get(k string) (v string) {
	m.l.Lock()
	if it, ok := m.m[k]; ok {
		v = it.value
	}
	m.l.Unlock()
	return
}

func (m *TTLMap) Display() (v string) {
	m.l.Lock()
	for i, k := range m.m {
		timeRemaining := k.lastValid - time.Now().Unix()
		v = v + i + "->" + strconv.FormatInt(timeRemaining, 10) + "s || "
	}
	m.l.Unlock()
	return
}

func (m *TTLMap) Clear() {
	m.l.Lock()
	for k := range m.m {
		delete(m.m, k)
	}
	m.l.Unlock()
	return
}

func (m *TTLMap) UnTimeout() {
	m.l.Lock()
	for username, _ := range m.m {
		m.client.Say(m.channel, "/untimeout "+username)
	}
	m.l.Unlock()
	m.Clear()
}
