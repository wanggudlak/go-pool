# go-pool
go语言实现的一个通用连接池

# 首先感谢大佬们为此类案例所做的先行工作
Pool [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/fatih/pool) [![Build Status](http://img.shields.io/travis/fatih/pool.svg?style=flat-square)](https://travis-ci.org/fatih/pool)
Fatih/pool [![fatih/pool]()](https://github.com/fatih/pool/blob/master/pool.go#L6)

连接池 Pool 是一个为接口Conn实现的线程安全的连接池，目前主要以通道实现。它能够被用来管理和重用通用类连接实例

## 安装和使用

### 用以下命令安装包
```bash
go get github.com/wanggudlak/go-pool
```

### 使用示例

```go
// 创建连接实例的方法由用户自己实现，例如我们想要创建一个tcp连接实例
intanceConnFunc := func() (pool.Conn, error) { return net.Dial(network, address) }

// 创建一个连接池，用于管理连接实例。通过以下方法，我们可以创建一个初始连接实例有5个，总容量有30的连接池
p, err := pool.NewChannelPool(5, 30, intanceConnFunc)

// 现在你可以从连接池中取出一个连接实例来使用，如果连接池中没有可用的连接实例，将会创建通过intanceConnFunc创建一个新的连接实例并放入连接池
conn, err := p.Get()

// 通过以下方法可以根据标记unusable决定是关闭连接实例还是将连接实例重新放入连接池
conn.Close()

// 如果你想要关闭连接实例，而不不要放回连接池，可以使用下面的方法
if cw, ok := conn.(*ConnWrapper); ok {
    cw.MarkUnusable()
    cw.Close()
}

// 如果你要关闭连接池，并且释放所有的连接实例，可以使用下面的方法
p.Close()

// 获取当前连接池中的可用连接实例数量，可以使用下面的方法
availableConnCount := p.Len()
```

## 维护人员
* [wangleigudlak@gmail.com](https://github.com/wanggudlak)
