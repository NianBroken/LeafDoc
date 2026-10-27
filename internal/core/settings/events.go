package settings

import "sync"

// Observers receive committed setting changes, including changes loaded by snapshot restore.
var changes = struct {
	sync.Mutex
	version uint64
	clients map[chan struct{}]bool
}{clients: make(map[chan struct{}]bool)}

func notifyCommitted() {
	changes.Lock()
	defer changes.Unlock()
	changes.version++
	for client := range changes.clients {
		select {
		case client <- struct{}{}:
		default:
		}
	}
}

func Version() uint64 {
	changes.Lock()
	defer changes.Unlock()
	return changes.version
}

// Subscribe coalesces changes while a client fetches the latest authoritative values.
func Subscribe() (<-chan struct{}, func()) {
	client := make(chan struct{}, 1)
	changes.Lock()
	changes.clients[client] = true
	changes.Unlock()
	return client, func() { changes.Lock(); delete(changes.clients, client); changes.Unlock() }
}
