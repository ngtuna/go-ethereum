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
	priceTree *redblacktreeextended.RedBlackTreeExtended
	priceMap  map[string]*OrderList // Dictionary containing price : OrderList object
	orderMap  map[string]*Order     // Dictionary containing orderId : Order object
	volume    *big.Int              // Contains total quantity from all Orders in tree
	numOrders int                   // Contains count of Orders in tree
	depth     int                   // Number of different prices in tree (http://en.wikipedia.org/wiki/Order_book_(trading)#Book_depth)
	slot      *big.Int
	Key       []byte
	db        TomoXDao
}

func NewOrderTree(key []byte, db TomoXDao) *OrderTree {
	slot := new(big.Int).SetBytes(key)
	priceTree := &redblacktreeextended.RedBlackTreeExtended{rbt.NewWith(decimalComparator)}
	priceMap := make(map[string]*OrderList)
	orderMap := make(map[string]*Order)
	return &OrderTree{
		priceTree: priceTree,
		priceMap:  priceMap,
		orderMap:  orderMap,
		volume:    Zero(),
		numOrders: 0,
		depth:     0,
		Key:       key,
		slot:      slot,
		db:        db,
	}
}

func (ordertree *OrderTree) Length() int {
	return len(ordertree.orderMap)
}

func (ordertree *OrderTree) Order(orderId string) *Order {
	return ordertree.orderMap[orderId]
}

func (ordertree *OrderTree) PriceList(price *big.Int) *OrderList {
	return ordertree.priceMap[price.String()]
}

func (ordertree *OrderTree) CreatePrice(price *big.Int) {
	ordertree.depth = ordertree.depth + 1
	newList := NewOrderList(price, ordertree.db)

	// set key to the new orderlist
	newList.Key = ordertree.getKeyFromPrice(price)
	// set slot to the new orderlist
	newList.slot = new(big.Int).SetBytes(crypto.Keccak256(newList.Key))

	ordertree.priceTree.Put(price, newList)
	ordertree.priceMap[price.String()] = newList
}

func (ordertree *OrderTree) RemovePrice(price *big.Int) {
	ordertree.depth = ordertree.depth - 1
	ordertree.priceTree.Remove(price)
	delete(ordertree.priceMap, price.String())
}

func (ordertree *OrderTree) PriceExist(price *big.Int) bool {
	if _, ok := ordertree.priceMap[price.String()]; ok {
		return true
	}
	return false
}

func (ordertree *OrderTree) OrderExist(orderId string) bool {
	if ordertree.orderMap == nil {
		log.Error("Ordertree ordermap is corrupted")
	}
	if _, ok := ordertree.orderMap[orderId]; ok {
		return true
	}
	return false
}

func (ordertree *OrderTree) RemoveOrderById(orderId string) {
	ordertree.numOrders = ordertree.numOrders - 1
	order := ordertree.orderMap[orderId]
	ordertree.volume = Sub(ordertree.volume, order.Quantity)
	order.OrderList.RemoveOrder(order)
	if order.OrderList.Length() == 0 {
		ordertree.RemovePrice(order.Price)
	}
	delete(ordertree.orderMap, orderId)
}

func (ordertree *OrderTree) MaxPrice() *big.Int {
	if ordertree.depth > 0 {
		value, found := ordertree.priceTree.GetMax()
		if found {
			return value.(*OrderList).price
		}
		return Zero()

	} else {
		return Zero()
	}
}

func (ordertree *OrderTree) MinPrice() *big.Int {
	if ordertree.depth > 0 {
		value, found := ordertree.priceTree.GetMin()
		if found {
			return value.(*OrderList).price
		} else {
			return Zero()
		}

	} else {
		return Zero()
	}
}

func (ordertree *OrderTree) MaxPriceList() *OrderList {
	if ordertree.depth > 0 {
		price := ordertree.MaxPrice()
		return ordertree.priceMap[price.String()]
	}
	return nil

}

func (ordertree *OrderTree) MinPriceList() *OrderList {
	if ordertree.depth > 0 {
		price := ordertree.MinPrice()
		return ordertree.priceMap[price.String()]
	}
	return nil
}

func (ordertree *OrderTree) InsertOrder(quote *Order) error {
	orderID := quote.OrderID

	if ordertree.OrderExist(strconv.FormatUint(orderID, 10)) {
		ordertree.RemoveOrderById(strconv.FormatUint(orderID, 10))
	}
	ordertree.numOrders++

	price := quote.Price

	if !ordertree.PriceExist(price) {
		ordertree.CreatePrice(price)
	}

	orderlist := ordertree.priceMap[price.String()]
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
	ordertree.orderMap[strconv.FormatUint(orderID, 10)] = order
	ordertree.volume = Add(ordertree.volume, order.Quantity)
	err = ordertree.Save()
	if err != nil {
		return err
	}

	return nil
}

func (ordertree *OrderTree) UpdateOrder(quote *Order) {
	order := ordertree.orderMap[strconv.FormatUint(quote.OrderID, 10)]
	originalQuantity := order.Quantity
	price := quote.Price

	if price != order.Price {
		// Price changed. Remove order and update tree.
		orderList := ordertree.priceMap[order.Price.String()]
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
	ordertree.volume = Add(ordertree.volume, addedQuantity)
}

// next time this price will be big.Int
func (orderTree *OrderTree) getKeyFromPrice(price *big.Int) []byte {
	orderListKey := orderTree.getSlotFromPrice(price)
	return GetKeyFromBig(orderListKey)
}

func (orderTree *OrderTree) getSlotFromPrice(price *big.Int) *big.Int {
	return Add(orderTree.slot, price)
}

func (orderTree *OrderTree) Save() error {
	value, err := EncodeBytesItem(orderTree)
	if err != nil {
		log.Error("Can't save ordertree", "value", value, "err", err)
		return err
	}
	log.Debug("Save ordertree ", "key", orderTree.Key, "value", value, "orderTree", orderTree)
	return orderTree.db.Put(orderTree.Key, value)
}
