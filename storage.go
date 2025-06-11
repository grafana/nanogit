package nanogit

import (
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

type PackfileStorage interface {
	Get(key hash.Hash) (*protocol.PackfileObject, bool)
	GetAllKeys() []hash.Hash
	Add(objs ...*protocol.PackfileObject)
	AddMap(objs map[string]*protocol.PackfileObject)
	Delete(key hash.Hash)
}
