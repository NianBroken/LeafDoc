// 本文件维护进程内配置变化版本和订阅通道，不保存跨进程通知记录。
package settings

import "sync"

// changes 在互斥锁内维护版本与订阅者，通知只有事件信号，不携带具体配置值。
var changes = struct {
	sync.Mutex
	version uint64
	clients map[chan struct{}]bool
}{clients: make(map[chan struct{}]bool)}

// notifyCommitted 增加版本并尝试向每个通道发送一次信号。
// 通道已有待处理信号时直接跳过，不等待慢订阅者，也不逐条保留所有变化。
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

// Version 返回当前进程的通知计数，重启后重新从零开始。
func Version() uint64 {
	changes.Lock()
	defer changes.Unlock()
	return changes.version
}

// Subscribe 注册容量为一的通道，返回只读端和取消函数，不发送初始状态。
// 取消只从订阅表删除，不关闭通道，不清除已经排队的信号，可重复调用。
func Subscribe() (<-chan struct{}, func()) {
	client := make(chan struct{}, 1)
	changes.Lock()
	changes.clients[client] = true
	changes.Unlock()
	return client, func() { changes.Lock(); delete(changes.clients, client); changes.Unlock() }
}
