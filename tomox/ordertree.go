package tomox

import (
	"math/big"
	"strconv"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/maurodelazeri/orderbook/extend"
)

type Comparator func(a, b interface{}) int
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
	volume    *big.Int       // Contains total quantity from all Orders in tree
	numOrders int                   // Contains count of Orders in tree
	depth     int                   // Number of different prices in tree (http://en.wikipedia.org/wiki/Order_book_(trading)#Book_depth)
	slot      *big.Int
	Key       []byte
}

func NewOrderTree() *OrderTree {
	priceTree := &redblacktreeextended.RedBlackTreeExtended{rbt.NewWith(decimalComparator)}
	priceMap := make(map[string]*OrderList)
	orderMap := make(map[string]*Order)
	return &OrderTree{
		priceTree: priceTree,
		priceMap: priceMap,
		orderMap: orderMap,
		volume: Zero(),
		numOrders: 0,
		depth: 0,
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
	newList := NewOrderList(price)
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
	if _, ok := ordertree.orderMap[orderId]; ok {
		return true
	}
	return false
}

func (ordertree *OrderTree) RemoveOrderById(orderId string) {
	ordertree.numOrders = ordertree.numOrders - 1
	order := ordertree.orderMap[orderId]
	ordertree.volume = Sub(ordertree.volume, order.quantity)
	order.orderList.RemoveOrder(order)
	if order.orderList.Length() == 0 {
		ordertree.RemovePrice(order.price)
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

func (ordertree *OrderTree) InsertOrder(quote *Order) {
	orderID := quote.orderID

	if ordertree.OrderExist(strconv.FormatUint(orderID, 10)) {
		ordertree.RemoveOrderById(strconv.FormatUint(orderID, 10))
	}
	ordertree.numOrders++

	price := quote.price

	if !ordertree.PriceExist(price) {
		ordertree.CreatePrice(price)
	}

	order := NewOrder(quote, ordertree.priceMap[price.String()])
	ordertree.priceMap[price.String()].AppendOrder(order)
	ordertree.orderMap[strconv.FormatUint(orderID, 10)] = order
	ordertree.volume = Add(ordertree.volume, order.quantity)
}

func (ordertree *OrderTree) UpdateOrder(quote *Order) {
	order := ordertree.orderMap[strconv.FormatUint(quote.orderID, 10)]
	originalQuantity := order.quantity
	price := quote.price

	if price != order.price {
		// Price changed. Remove order and update tree.
		orderList := ordertree.priceMap[order.price.String()]
		orderList.RemoveOrder(order)
		if orderList.Length() == 0 {
			ordertree.RemovePrice(price)
		}
		ordertree.InsertOrder(quote)
	} else {
		quantity := quote.quantity
		timestamp := quote.updatedAt
		order.UpdateQuantity(quantity, timestamp)
	}
	addedQuantity := Sub(order.quantity, originalQuantity)
	ordertree.volume = Add(ordertree.volume, addedQuantity)
}
