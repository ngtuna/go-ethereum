package tomox

import (
	"container/heap"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
)

// An OrderPending is something we manage in a priority queue.
type OrderPending struct {
	nonce     *big.Int       // order nonce
	timestamp uint64         // The priority of order in the queue.
	hash      common.Hash    // order hash
	address   common.Address // order's owner
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of order in the queue.
}

type OrderByAddress struct {
	nonce *big.Int
	hash  common.Hash
}

func (o OrderByAddress) Nonce() uint64 { return o.nonce.Uint64() }

type OrdersByAddress []OrderByAddress

// A PriorityQueue implements heap.Interface and holds OrderPendings.
type PriorityQueue []*OrderPending

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].timestamp <= pq[j].timestamp
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*OrderPending)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0: n-1]
	return item
}

// update modifies the priority and value of an OrderPending in the queue.
func (pq *PriorityQueue) update(item *OrderPending, nonce *big.Int) {
	item.nonce = nonce
	heap.Fix(pq, item.index)
}
