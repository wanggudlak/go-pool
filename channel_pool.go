package pool

import (
	"errors"
	"fmt"
	"sync"
)

// ChannelPool 基于带缓存的channel实现Pool接口
type ChannelPool struct {
	mu           sync.RWMutex
	connsChan    chan Conn
	instanceConn InstanceConn
}

// InstanceConn 是可以创建一个连接实例的函数
type InstanceConn func() (Conn, error)

func NewChannelPool(initialConnCount, maxConnCount int, instanceConnFunc InstanceConn) (Pool, error) {
	if initialConnCount < 0 || maxConnCount <= 0 || initialConnCount > maxConnCount {
		return nil, errors.New("无效参数：initialConnCount, maxConnCount不能小于0且initialConnCount不能大于maxConnCount")
	}

	c := &ChannelPool{
		connsChan:    make(chan Conn, maxConnCount),
		instanceConn: instanceConnFunc,
	}

	for i := 0; i < initialConnCount; i++ {
		conn, err := instanceConnFunc()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("创建一个连接实例的函数发生错误，不能加入到连接池中：%s", err)
		}
		c.connsChan <- conn
	}

	return c, nil
}

func (p *ChannelPool) GetConnsChan() chan Conn {
	return p.connsChan
}

func (p *ChannelPool) GetInstanceConnFunc() InstanceConn {
	return p.instanceConn
}

// Get 实现Pool接口中的Get方法。如果连接池中没有可用的连接实例，将通过InstanceConn函数创建一个新的连接实例，
// 并放入连接池
func (p *ChannelPool) Get() (Conn, error) {
	var errPoolClosed = errors.New("pool is closed")
	connsChan, instanceConnFunc := p.getConnsChanAndInstanceConnFunc()
	if nil == connsChan {
		return nil, errPoolClosed
	}

	select {
	case conn := <-connsChan:
		if nil == conn {
			return nil, errPoolClosed
		}
		return p.wrapConn(conn), nil
	default:
		conn, err := instanceConnFunc()
		if err != nil {
			return nil, err
		}
		p.put(conn)

		return p.wrapConn(conn), nil
	}
}

func (p *ChannelPool) put(conn Conn) error {
	if nil == conn {
		return errors.New("连接实例是nil，被拒绝加入连接池")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if nil == p.connsChan {
		// 连接池已被关闭，那么关闭已经实例化的连接实例
		return conn.Close()
	}

	// 将连接实例放入连接池，如果连接池已满，将连接实例放入通道将会阻塞，default分支会执行
	select {
	case p.connsChan <- conn:
		return nil
	default:
		// 连接池已满，关闭已经实例化的连接
		return conn.Close()
	}
}

// Close 实现Pool接口中的Close方法，用途是关闭连接池及连接实例
func (p *ChannelPool) Close() {
	p.mu.Lock()
	connsChan := p.connsChan
	p.connsChan = nil
	p.instanceConn = nil
	p.mu.Unlock()

	if nil == connsChan {
		return
	}

	close(connsChan)
	for conn := range connsChan {
		conn.Close()
	}
}

func (p *ChannelPool) Len() int {
	connsChan, _ := p.getConnsChanAndInstanceConnFunc()
	return len(connsChan)
}

func (p *ChannelPool) getConnsChanAndInstanceConnFunc() (chan Conn, InstanceConn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connsChan, p.instanceConn
}

func (p *ChannelPool) wrapConn(conn Conn) Conn {
	cw := &ConnWrapper{
		p: p,
	}
	cw.Conn = conn
	return cw
}
