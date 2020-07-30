package pool_test

import (
	"github.com/wanggudlak/go-pool"
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

var (
	InitialConnCount = 5
	MaximumConnCount = 30
	network          = "tcp"
	address          = "127.0.0.1:9092"
	instanceConnFunc = func() (pool.Conn, error) { return net.Dial(network, address) }
)

func init() {
	go simpleTCPServer()
	time.Sleep(time.Millisecond * 300)
	rand.Seed(time.Now().UTC().UnixNano())
}

func simpleTCPServer() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			buffer := make([]byte, 256)
			conn.Read(buffer)
		}()
	}
}

func TestNewChannelPool(t *testing.T) {
	_, err := pool.NewChannelPool(InitialConnCount, MaximumConnCount, instanceConnFunc)
	if err != nil {
		t.Errorf("NewChannelPool error: %s", err)
	}
}

func TestChannelPool_Get(t *testing.T) {
	p, _ := pool.NewChannelPool(InitialConnCount, MaximumConnCount, instanceConnFunc)
	defer p.Close()

	_, err := p.Get()
	if err != nil {
		t.Errorf("Get pool error: %s", err)
	}

	if p.Len() != (InitialConnCount - 1) {
		t.Errorf("Get pool error. Expecting %d connect instances, got %d", (InitialConnCount - 1), p.Len())
	}

	var wg sync.WaitGroup
	for i := 0; i < (InitialConnCount - 1); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Get()
			if err != nil {
				t.Errorf("Get pool error : %s", err)
			}
		}()
	}
	wg.Wait()

	if p.Len() != 0 {
		t.Errorf("Get pool error. Expecting %d connect instances, got %d", 0, p.Len())
	}

	_, err = p.Get()
	if err != nil {
		t.Errorf("Get pool error: %s", err)
	}
}

func TestChannelPool_Put(t *testing.T) {
	p, err := pool.NewChannelPool(0, MaximumConnCount, instanceConnFunc)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	conns := make([]pool.Conn, MaximumConnCount)
	for i := 0; i < MaximumConnCount; i++ {
		conn, _ := p.Get()
		conns[i] = conn
	}

	for _, conn := range conns {
		conn.Close()
	}

	if p.Len() != MaximumConnCount {
		t.Errorf("Put error len. Expecting %d connect instances, got %d", MaximumConnCount, p.Len())
	}

	conn, _ := p.Get()
	p.Close()

	conn.Close()
	if p.Len() != 0 {
		t.Errorf("Put error. Closed pool shouldn't allow to put connections.")
	}
}

func TestChannelPool_PutUnusableConn(t *testing.T) {
	p, _ := pool.NewChannelPool(InitialConnCount, MaximumConnCount, instanceConnFunc)
	defer p.Close()

	conn, _ := p.Get()
	conn.Close()

	poolSize := p.Len()
	conn, _ = p.Get()
	conn.Close()
	if p.Len() != poolSize {
		t.Errorf("Pool size is expected to equal to initial size")
	}

	conn, _ = p.Get()
	if cw, ok := conn.(*pool.ConnWrapper); !ok {
		t.Errorf("从连接池中获取连接实例失败")
	} else {
		cw.MarkUnusable()
	}
	conn.Close()
	if p.Len() != poolSize-1 {
		t.Errorf("Pool size is expected to be %d, got %d", poolSize-1, p.Len())
	}
}

func TestChannelPool_Close(t *testing.T) {
	p, _ := pool.NewChannelPool(InitialConnCount, MaximumConnCount, instanceConnFunc)
	p.Close()

	c := p.(*pool.ChannelPool)
	if c.GetConnsChan() != nil {
		t.Errorf("Close pool error, conns channel should be nil")
	}
	if c.GetInstanceConnFunc() != nil {
		t.Errorf("Close pool error, instance connect function should be nil")
	}

	_, err := p.Get()
	if err == nil {
		t.Errorf("Close pool error, get connect should return an error")
	}

	if p.Len() != 0 {
		t.Errorf("Close pool error, pool capacity expecting 0, got %d", p.Len())
	}
}

func TestChannelPoolConcurrent1(t *testing.T) {
	p, _ := pool.NewChannelPool(InitialConnCount, MaximumConnCount, instanceConnFunc)
	pipe := make(chan pool.Conn, 0)

	go func() {
		p.Close()
	}()

	for i := 0; i < MaximumConnCount; i++ {
		go func() {
			conn, _ := p.Get()
			pipe <- conn
		}()

		go func() {
			conn := <-pipe
			if conn == nil {
				return
			}
			conn.Close()
		}()
	}
}

func TestChannelPoolConcurrent2(t *testing.T) {
	p, _ := pool.NewChannelPool(0, MaximumConnCount, instanceConnFunc)

	var wg sync.WaitGroup

	go func() {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				conn, _ := p.Get()
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				conn.Close()
				wg.Done()
			}(i)
		}
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			conn, _ := p.Get()
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			conn.Close()
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestChannelPoolConcurrent3(t *testing.T) {
	p, _ := pool.NewChannelPool(0, 1, instanceConnFunc)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		p.Close()
		wg.Done()
	}()

	if conn, err := p.Get(); err == nil {
		conn.Close()
	}

	wg.Wait()
}
