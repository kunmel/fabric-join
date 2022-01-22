package crossdb

type CrossDBProvider interface {
	GetDBHandler(id string) (CrossDB, error)
	Close()
}

type CrossDB interface {
	Test()
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte, sync bool) error
}