package pool

type Pool interface {
	Get() (Conn, error)
	Close()
	Len() int
}
