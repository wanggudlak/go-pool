package pool

import "sync"

// Conn 连接实例需要实现的接口
type Conn interface {
	Close() error
}

// ConnWrapper 连接实例包裹
type ConnWrapper struct {
	Conn
	mu sync.RWMutex
	p  *ChannelPool

	// 标记为不能使用
	unusable bool
}

// Close 根据标记unusable决定是关闭连接实例还是将连接实例重新放入连接池
func (cw *ConnWrapper) Close() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.unusable {
		if cw.Conn != nil {
			return cw.Conn.Close()
		}
		return nil
	}
	return cw.p.put(cw.Conn)
}

// MarkUnusable 标记连接不再可用
func (cw *ConnWrapper) MarkUnusable() {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.unusable = true
}
