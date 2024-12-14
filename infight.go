package mqtt

import (
	"github.com/golang-io/mqtt/packet"
	"sync"
)

type InFight struct {
	mu   *sync.RWMutex
	maps map[uint16]*packet.PUBLISH
}

func newInFight() *InFight {
	return &InFight{
		mu:   new(sync.RWMutex),
		maps: make(map[uint16]*packet.PUBLISH),
	}
}

func (i *InFight) Get(id uint16) (*packet.PUBLISH, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	pkt, ok := i.maps[id]
	if ok {
		delete(i.maps, id)
	}
	//log.Printf("[InFight] GET: id=%d, kind=%s", pkt.ID(), Kind[pkt.PacketKind()])
	return pkt, ok
}

func (i *InFight) Put(pkt *packet.PUBLISH) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	//log.Printf("[InFight] PUT: id=%d, kind=%s", pkt.ID(), Kind[pkt.PacketKind()])
	i.maps[pkt.PacketID] = pkt
	return true
}
