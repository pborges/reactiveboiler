package websock

import (
	"encoding/json"
	"log"
	"sync"
)

type CentralLocking struct {
	db   []*Lock
	lock sync.Mutex
	Log  *log.Logger
}

func (c *CentralLocking) log(msg string) {
	if c.Log != nil {
		c.Log.Output(3, msg)
	}
}

type Lock struct {
	Key    string
	RLocks int
	WLock  bool
	rwlock sync.RWMutex
	self   sync.Mutex
	parent *CentralLocking
}

func (c *CentralLocking) CreateLock(key string) *Lock {
	c.lock.Lock()
	defer c.lock.Unlock()
	l := &Lock{
		Key:    key,
		parent: c,
	}
	c.db = append(c.db, l)
	return l
}

func (l *Lock) Lock() {
	l.self.Lock()
	defer l.self.Unlock()
	l.parent.log("lock " + l.Key)
	l.rwlock.Lock()
	l.WLock = true
}

func (l *Lock) Unlock() {
	l.self.Lock()
	defer l.self.Unlock()
	l.parent.log("unlock " + l.Key)
	l.rwlock.Unlock()
	l.WLock = false
}

func (l *Lock) RLock() {
	l.self.Lock()
	defer l.self.Unlock()
	l.parent.log("rlock " + l.Key)
	l.rwlock.RLock()
	l.RLocks++
}

func (l *Lock) RUnlock() {
	l.self.Lock()
	defer l.self.Unlock()
	l.parent.log("runlock " + l.Key)
	l.rwlock.RUnlock()
	l.RLocks--
}

func (c *CentralLocking) Delete(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.log("delete " + key)
	for i, l := range c.db {
		if l.Key == key {
			c.db = append(c.db[:i], c.db[i+1:]...)
			return
		}
	}
}

func (c *CentralLocking) MarshalJSON() ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	locks := map[string]interface{}{}
	for _, l := range c.db {
		l.self.Lock()
		locks[l.Key] = map[string]interface{}{
			"rlocks": l.RLocks,
			"wlock":  l.WLock,
		}
		l.self.Unlock()
	}
	return json.Marshal(locks)
}
