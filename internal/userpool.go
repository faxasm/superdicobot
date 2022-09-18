package userpool

import (
	"github.com/gempir/go-twitch-irc/v3"
	"go.uber.org/ratelimit"
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
	rl      ratelimit.Limiter
}

func New(ln int, channel string, client *twitch.Client, rl ratelimit.Limiter) (m *TTLMap) {
	m = &TTLMap{
		m:       make(map[string]*userPool, ln),
		channel: channel,
		client:  client,
		rl:      rl,
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
		v = v + k.value + "->" + i + ": " + strconv.FormatInt(timeRemaining, 10) + "s\n"
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
		m.rl.Take()
		m.client.Say(m.channel, "/untimeout "+username)
	}
	m.l.Unlock()
	m.Clear()
}
