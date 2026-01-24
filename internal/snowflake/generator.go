package snowflake

import (
	"sync"
	"time"
)

const (
	epoch        = int64(1767225600000)
	nodeBits     = 10
	sequenceBits = 12
	maxNode      = -1 ^ (-1 << nodeBits)
	maxSequence  = -1 ^ (-1 << sequenceBits)
	timeShift    = nodeBits + sequenceBits
	nodeShift    = sequenceBits
)

type Generator struct {
	mu       sync.Mutex
	nodeID   int64
	lastTime int64
	sequence int64
}

func NewGenerator(nodeID int64) *Generator {
	if nodeID < 0 || nodeID > maxNode {
		panic("nodeID must be between 0 and 1023")
	}
	return &Generator{nodeID: nodeID}
}

func (g *Generator) Generate() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli() - epoch
	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			for now <= g.lastTime {
				now = time.Now().UnixMilli() - epoch
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastTime = now
	return (now << timeShift) | (g.nodeID << nodeShift) | g.sequence
}
