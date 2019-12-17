package utils

import (
	"log"
	"sync"
)

//Locks is a struc for what the lock strategy is based on
type Locks struct {
	namespaces *sync.Map
}

//NewLocks initialize everything
func NewLocks() *Locks {
	return &Locks{&sync.Map{}}
}

//LoadOrStoreLock takes as param the namespace and locks it
func (l *Locks) LoadOrStoreLock(namespace string) {
	log.Printf("locking for %v", namespace)
	lock, _ := l.namespaces.LoadOrStore(namespace, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	log.Printf("locked for %v", namespace)
}

//Unlock takes the namespace and further release the lock
func (l *Locks) Unlock(namespace string) {
	log.Printf("unlocking for %v", namespace)
	lock, ok := l.namespaces.Load(namespace)
	if ok {
		lock.(*sync.Mutex).Unlock()
	}
}
