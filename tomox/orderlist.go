package tomox

import (
	"math/big"

	"github.com/HuKeping/rbtree"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OrderList struct {
	headOrder *Order
	tailOrder *Order
	length    int
	volume    *big.Int
	lastOrder *Order
	price     *big.Int
	Key       []byte
	slot      *big.Int
	db        TomoXDao
}

func NewOrderList(price *big.Int, db TomoXDao) *OrderList {
	return &OrderList{headOrder: nil, tailOrder: nil, length: 0, volume: Zero(),
		lastOrder: nil, price: price, db: db}
}

func (orderlist *OrderList) Less(than rbtree.Item) bool {
	return orderlist.price.Cmp(than.(*OrderList).price) < 0
}

func (orderlist *OrderList) Length() int {
	return orderlist.length
}

func (orderlist *OrderList) HeadOrder() *Order {
	return orderlist.headOrder
}

func (orderlist *OrderList) AppendOrder(order *Order) {
	if orderlist.Length() == 0 {
		order.NextOrder = nil
		order.PrevOrder = nil
		orderlist.headOrder = order
		orderlist.tailOrder = order
	} else {
		order.PrevOrder = orderlist.tailOrder
		order.NextOrder = nil
		orderlist.tailOrder.NextOrder = order
		orderlist.tailOrder = order
	}
	orderlist.length = orderlist.length + 1
	orderlist.volume = Add(orderlist.volume, order.Quantity)
}

func (orderlist *OrderList) RemoveOrder(order *Order) {
	orderlist.volume = Sub(orderlist.volume, order.Quantity)
	orderlist.length = orderlist.length - 1
	if orderlist.length == 0 {
		return
	}

	nextOrder := order.NextOrder
	prevOrder := order.PrevOrder

	if nextOrder != nil && prevOrder != nil {
		nextOrder.PrevOrder = prevOrder
		prevOrder.NextOrder = nextOrder
	} else if nextOrder != nil {
		nextOrder.PrevOrder = nil
		orderlist.headOrder = nextOrder
	} else if prevOrder != nil {
		prevOrder.NextOrder = nil
		orderlist.tailOrder = prevOrder
	}
}

func (orderlist *OrderList) MoveToTail(order *Order) {
	if order.PrevOrder != nil { // This Order is not the first Order in the OrderList
		order.PrevOrder.NextOrder = order.NextOrder // Link the previous Order to the next Order, then move the Order to tail
	} else { // This Order is the first Order in the OrderList
		orderlist.headOrder = order.NextOrder // Make next order the first
	}
	order.NextOrder.PrevOrder = order.PrevOrder

	// Move Order to the last position. Link up the previous last position Order.
	orderlist.tailOrder.NextOrder = order
	orderlist.tailOrder = order
}

func (orderList *OrderList) SaveOrder(order *Order) error {
	key := orderList.GetOrderID(order)
	value, err := EncodeBytesItem(order)
	if err != nil {
		log.Error("Can't save order", "value", value, "err", err)
		return err
	}
	log.Debug("Save order ", "key", key, "value", value, "order", order)
	return orderList.db.Put(key, value)
}

// GetOrderID return the real slot key of order in this linked list
func (orderList *OrderList) GetOrderID(order *Order) []byte {
	return orderList.GetOrderIDFromKey(order.Key)
}

// If we allow the same orderid belongs to many pricelist, we must use slot
// otherwise just use 1 db for storing all orders of all pricelists
// currently we use auto increase ment id so no need slot
func (orderList *OrderList) GetOrderIDFromKey(key []byte) []byte {
	orderSlot := new(big.Int).SetBytes(key)
	return common.BigToHash(Add(orderList.slot, orderSlot)).Bytes()
}

func (orderList *OrderList) Save() error {
	value, err := EncodeBytesItem(orderList)
	if err != nil {
		log.Error("Can't save orderlist", "value", value, "err", err)
		return err
	}
	log.Debug("Save orderlist ", "key", orderList.Key, "value", value, "orderList", orderList)
	return orderList.db.Put(orderList.Key, value)
}
