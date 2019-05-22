package tomox

import (
	"math/big"

	"github.com/HuKeping/rbtree"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OrderList struct {
	HOrder *Order
	TOrder *Order
	Len    int
	Volume    *big.Int
	LastOrder *Order
	Price     *big.Int
	Key       []byte
	Slot      *big.Int
	Db        TomoXDao
}

func NewOrderList(Price *big.Int, db TomoXDao) *OrderList {
	return &OrderList{HOrder: nil, TOrder: nil, Len: 0, Volume: Zero(),
		LastOrder: nil, Price: Price, Db: db}
}

func (orderlist *OrderList) Less(than rbtree.Item) bool {
	return orderlist.Price.Cmp(than.(*OrderList).Price) < 0
}

func (orderlist *OrderList) Length() int {
	return orderlist.Len
}

func (orderlist *OrderList) HeadOrder() *Order {
	return orderlist.HOrder
}

func (orderlist *OrderList) AppendOrder(order *Order) {
	if orderlist.Length() == 0 {
		order.NextOrder = nil
		order.PrevOrder = nil
		orderlist.HOrder = order
		orderlist.TOrder = order
	} else {
		order.PrevOrder = orderlist.TOrder
		order.NextOrder = nil
		orderlist.TOrder.NextOrder = order
		orderlist.TOrder = order
	}
	orderlist.Len = orderlist.Len + 1
	orderlist.Volume = Add(orderlist.Volume, order.Quantity)
}

func (orderlist *OrderList) RemoveOrder(order *Order) {
	orderlist.Volume = Sub(orderlist.Volume, order.Quantity)
	orderlist.Len = orderlist.Len - 1
	if orderlist.Len == 0 {
		return
	}

	nextOrder := order.NextOrder
	prevOrder := order.PrevOrder

	if nextOrder != nil && prevOrder != nil {
		nextOrder.PrevOrder = prevOrder
		prevOrder.NextOrder = nextOrder
	} else if nextOrder != nil {
		nextOrder.PrevOrder = nil
		orderlist.HOrder = nextOrder
	} else if prevOrder != nil {
		prevOrder.NextOrder = nil
		orderlist.TOrder = prevOrder
	}
}

func (orderlist *OrderList) MoveToTail(order *Order) {
	if order.PrevOrder != nil { // This Order is not the first Order in the OrderList
		order.PrevOrder.NextOrder = order.NextOrder // Link the previous Order to the next Order, then move the Order to tail
	} else { // This Order is the first Order in the OrderList
		orderlist.HOrder = order.NextOrder // Make next order the first
	}
	order.NextOrder.PrevOrder = order.PrevOrder

	// Move Order to the last position. Link up the previous last position Order.
	orderlist.TOrder.NextOrder = order
	orderlist.TOrder = order
}

func (orderList *OrderList) SaveOrder(order *Order) error {
	key := orderList.GetOrderID(order)
	value, err := EncodeBytesItem(order)
	if err != nil {
		log.Error("Can't save order", "value", value, "err", err)
		return err
	}
	log.Debug("Save order ", "key", key, "value", value, "order", order)
	return orderList.Db.Put(key, value)
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
	return common.BigToHash(Add(orderList.Slot, orderSlot)).Bytes()
}

func (orderList *OrderList) Save() error {
	value, err := EncodeBytesItem(orderList)
	if err != nil {
		log.Error("Can't save orderlist", "value", value, "err", err)
		return err
	}
	log.Debug("Save orderlist ", "key", orderList.Key, "value", value, "orderList", orderList)
	return orderList.Db.Put(orderList.Key, value)
}
