package tomox

import (
	"math/big"
	"strconv"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/maurodelazeri/orderbook/extend"
)

func decimalComparator(a, b interface{}) int {
	aAsserted := a.(*big.Int)
	bAsserted := b.(*big.Int)
	switch {
	case aAsserted.Cmp(bAsserted) > 0:
		return 1
	case aAsserted.Cmp(bAsserted) < 0:
		return -1
	default:
		return 0
	}
}

type OrderTree struct {
	PriceTree *redblacktreeextended.RedBlackTreeExtended
	PriceMap  map[string]*OrderList // Dictionary containing price : OrderList object
	OrderMap  map[string]*Order     // Dictionary containing orderId : Order object
	Volume    *big.Int              // Contains total quantity from all Orders in tree
	NumOrders int                   // Contains count of Orders in tree
	Depth     int                   // Number of different prices in tree (http://en.wikipedia.org/wiki/Order_book_(trading)#Book_Depth)
	Slot      *big.Int
	Key       []byte
	Db        TomoXDao
}

func NewOrderTree(key []byte, db TomoXDao) *OrderTree {
	Slot := new(big.Int).SetBytes(key)
	PriceTree := &redblacktreeextended.RedBlackTreeExtended{rbt.NewWith(decimalComparator)}
	PriceMap := make(map[string]*OrderList)
	OrderMap := make(map[string]*Order)
	return &OrderTree{
		PriceTree: PriceTree,
		PriceMap:  PriceMap,
		OrderMap:  OrderMap,
		Volume:    Zero(),
		NumOrders: 0,
		Depth:     0,
		Key:       key,
		Slot:      Slot,
		Db:        db,
	}
}

func (ordertree *OrderTree) Length() int {
	return len(ordertree.OrderMap)
}

func (ordertree *OrderTree) Order(orderId string) *Order {
	return ordertree.OrderMap[orderId]
}

func (ordertree *OrderTree) PriceList(price *big.Int) *OrderList {
	return ordertree.PriceMap[price.String()]
}

func (ordertree *OrderTree) CreatePrice(price *big.Int) {
	ordertree.Depth = ordertree.Depth + 1
	newList := NewOrderList(price, ordertree.Db)

	// set key to the new orderlist
	newList.Key = ordertree.getKeyFromPrice(price)
	// set Slot to the new orderlist
	newList.Slot = new(big.Int).SetBytes(crypto.Keccak256(newList.Key))

	ordertree.PriceTree.Put(price, newList)
	ordertree.PriceMap[price.String()] = newList
}

func (ordertree *OrderTree) RemovePrice(price *big.Int) {
	ordertree.Depth = ordertree.Depth - 1
	ordertree.PriceTree.Remove(price)
	delete(ordertree.PriceMap, price.String())
}

func (ordertree *OrderTree) PriceExist(price *big.Int) bool {
	if _, ok := ordertree.PriceMap[price.String()]; ok {
		return true
	}
	return false
}

func (ordertree *OrderTree) OrderExist(orderId string) bool {
	if ordertree.OrderMap == nil {
		log.Error("Ordertree ordermap is corrupted")
	}
	if _, ok := ordertree.OrderMap[orderId]; ok {
		return true
	}
	return false
}

func (ordertree *OrderTree) RemoveOrderById(orderId string) {
	ordertree.NumOrders = ordertree.NumOrders - 1
	order := ordertree.OrderMap[orderId]
	ordertree.Volume = Sub(ordertree.Volume, order.Quantity)
	order.OrderList.RemoveOrder(order)
	if order.OrderList.Length() == 0 {
		ordertree.RemovePrice(order.Price)
	}
	delete(ordertree.OrderMap, orderId)
}

func (ordertree *OrderTree) MaxPrice() *big.Int {
	if ordertree.Depth > 0 {
		value, found := ordertree.PriceTree.GetMax()
		if found {
			return value.(*OrderList).Price
		}
		return Zero()

	} else {
		return Zero()
	}
}

func (ordertree *OrderTree) MinPrice() *big.Int {
	if ordertree.Depth > 0 {
		value, found := ordertree.PriceTree.GetMin()
		if found {
			return value.(*OrderList).Price
		} else {
			return Zero()
		}

	} else {
		return Zero()
	}
}

func (ordertree *OrderTree) MaxPriceList() *OrderList {
	if ordertree.Depth > 0 {
		price := ordertree.MaxPrice()
		return ordertree.PriceMap[price.String()]
	}
	return nil

}

func (ordertree *OrderTree) MinPriceList() *OrderList {
	if ordertree.Depth > 0 {
		price := ordertree.MinPrice()
		return ordertree.PriceMap[price.String()]
	}
	return nil
}

func (ordertree *OrderTree) InsertOrder(quote *Order) error {
	orderID := quote.OrderID

	if ordertree.OrderExist(strconv.FormatUint(orderID, 10)) {
		ordertree.RemoveOrderById(strconv.FormatUint(orderID, 10))
	}
	ordertree.NumOrders++

	price := quote.Price

	if !ordertree.PriceExist(price) {
		ordertree.CreatePrice(price)
	}

	orderlist := ordertree.PriceMap[price.String()]
	order := NewOrder(quote, orderlist)

	// set order.Key
	order.Key = GetKeyFromBig(new(big.Int).SetUint64(order.OrderID))
	orderlist.AppendOrder(order)

	// save order to DB
	err := orderlist.SaveOrder(order)
	if err != nil {
		return err
	}

	// save orderlist to DB
	err = orderlist.Save()
	if err != nil {
		return err
	}

	// save ordertree to DB
	ordertree.OrderMap[strconv.FormatUint(orderID, 10)] = order
	ordertree.Volume = Add(ordertree.Volume, order.Quantity)
	err = ordertree.Save()
	if err != nil {
		return err
	}

	return nil
}

func (ordertree *OrderTree) UpdateOrder(quote *Order) {
	order := ordertree.OrderMap[strconv.FormatUint(quote.OrderID, 10)]
	originalQuantity := order.Quantity
	price := quote.Price

	if price != order.Price {
		// Price changed. Remove order and update tree.
		orderList := ordertree.PriceMap[order.Price.String()]
		orderList.RemoveOrder(order)
		if orderList.Length() == 0 {
			ordertree.RemovePrice(price)
		}
		ordertree.InsertOrder(quote)
	} else {
		quantity := quote.Quantity
		timestamp := quote.UpdatedAt
		order.UpdateQuantity(quantity, timestamp)
	}
	addedQuantity := Sub(order.Quantity, originalQuantity)
	ordertree.Volume = Add(ordertree.Volume, addedQuantity)
}

// next time this price will be big.Int
func (orderTree *OrderTree) getKeyFromPrice(price *big.Int) []byte {
	orderListKey := orderTree.getSlotFromPrice(price)
	return GetKeyFromBig(orderListKey)
}

func (orderTree *OrderTree) getSlotFromPrice(price *big.Int) *big.Int {
	return Add(orderTree.Slot, price)
}

func (orderTree *OrderTree) Save() error {
	value, err := EncodeBytesItem(orderTree)
	if err != nil {
		log.Error("Can't save ordertree", "value", value, "err", err)
		return err
	}
	log.Debug("Save ordertree ", "key", orderTree.Key, "value", value, "orderTree", orderTree)
	return orderTree.Db.Put(orderTree.Key, value)
}
