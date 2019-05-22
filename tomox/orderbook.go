package tomox

import (
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

const (
	Ask    = "SELL"
	Bid    = "BUY"
	Market = "market"
	Limit  = "limit"
	Cancel = "CANCELLED"

	// we use a big number as segment for storing order, order list from order tree slot.
	// as sequential id
	SlotSegment = common.AddressLength
)

type OrderBook struct {
	Db          TomoXDao
	Bids        *OrderTree
	Asks        *OrderTree
	Time        uint64
	NextOrderID uint64
	PairName    string
	Key         []byte
	Slot        *big.Int
}

// NewOrderBook : return new order book
func NewOrderBook(pairName string, db TomoXDao) *OrderBook {
	// do slot with hash to prevent collision

	// we convert to lower case, so even with name as contract address, it is still correct
	// without converting back from hex to bytes
	key := crypto.Keccak256([]byte(strings.ToLower(pairName)))
	slot := new(big.Int).SetBytes(key)

	// we just increase the segment at the most byte at address length level to avoid conflict
	// somehow it is like 2 hashes has the same common prefix and it is very difficult to resolve
	// the order id start at orderbook slot
	// the price of order tree start at order tree slot
	bidsKey := GetSegmentHash(key, 1, SlotSegment)
	asksKey := GetSegmentHash(key, 2, SlotSegment)

	bids := NewOrderTree(bidsKey, db)
	asks := NewOrderTree(asksKey, db)

	return &OrderBook{
		Bids:        bids,
		Asks:        asks,
		Time:        0,
		NextOrderID: 0,
		PairName:    strings.ToLower(pairName),
		Db:          db,
		Key:         key,
		Slot:        slot,
	}
}

func (orderBook *OrderBook) UpdateTime() {
	timestamp := uint64(time.Now().Unix())
	orderBook.Time = timestamp
}

func (orderBook *OrderBook) BestBid() (value *big.Int) {
	value = orderBook.Bids.MaxPrice()
	return
}

func (orderBook *OrderBook) BestAsk() (value *big.Int) {
	value = orderBook.Asks.MinPrice()
	return
}

func (orderBook *OrderBook) WorstBid() (value *big.Int) {
	value = orderBook.Bids.MinPrice()
	return
}

func (orderBook *OrderBook) WorstAsk() (value *big.Int) {
	value = orderBook.Asks.MaxPrice()
	return
}

func (orderBook *OrderBook) ProcessMarketOrder(quote *Order, verbose bool) []map[string]string {
	var trades []map[string]string
	quantity_to_trade := quote.Quantity
	side := quote.Side
	var new_trades []map[string]string

	if side == Bid {
		for quantity_to_trade.Cmp(Zero()) > 0 && orderBook.Asks.Length() > 0 {
			best_price_asks := orderBook.Asks.MinPriceList()
			quantity_to_trade, new_trades = orderBook.ProcessOrderList(Ask, best_price_asks, quantity_to_trade, quote, verbose)
			trades = append(trades, new_trades...)
		}
	} else if side == Ask {
		for quantity_to_trade.Cmp(Zero()) > 0 && orderBook.Bids.Length() > 0 {
			best_price_bids := orderBook.Bids.MaxPriceList()
			quantity_to_trade, new_trades = orderBook.ProcessOrderList(Bid, best_price_bids, quantity_to_trade, quote, verbose)
			trades = append(trades, new_trades...)
		}
	}
	return trades
}

func (orderBook *OrderBook) ProcessLimitOrder(quote *Order, verbose bool) ([]map[string]string, *Order) {
	var trades []map[string]string
	quantity_to_trade := quote.Quantity
	side := quote.Side
	price := quote.Price
	var new_trades []map[string]string

	order_in_book := &Order{}

	if side == Bid {
		minPrice := orderBook.Asks.MinPrice()
		for quantity_to_trade.Cmp(Zero()) > 0 && orderBook.Asks.Length() > 0 && price.Cmp(minPrice) >= 0 {
			best_price_asks := orderBook.Asks.MinPriceList()
			quantity_to_trade, new_trades = orderBook.ProcessOrderList(Ask, best_price_asks, quantity_to_trade, quote, verbose)
			trades = append(trades, new_trades...)
			minPrice = orderBook.Asks.MinPrice()
		}

		if quantity_to_trade.Cmp(Zero()) > 0 {
			quote.OrderID = orderBook.NextOrderID
			quote.Quantity = quantity_to_trade
			orderBook.Bids.InsertOrder(quote)
			order_in_book = quote
		}

	} else if side == Ask {
		maxPrice := orderBook.Bids.MaxPrice()
		for quantity_to_trade.Cmp(Zero()) > 0 && orderBook.Bids.Length() > 0 && price.Cmp(maxPrice) <= 0 {
			best_price_bids := orderBook.Bids.MaxPriceList()
			quantity_to_trade, new_trades = orderBook.ProcessOrderList(Bid, best_price_bids, quantity_to_trade, quote, verbose)
			trades = append(trades, new_trades...)
			maxPrice = orderBook.Bids.MaxPrice()
		}

		if quantity_to_trade.Cmp(Zero()) > 0 {
			quote.OrderID = orderBook.NextOrderID
			quote.Quantity = quantity_to_trade
			orderBook.Asks.InsertOrder(quote)
			order_in_book = quote
		}
	}
	return trades, order_in_book
}

func (orderBook *OrderBook) ProcessOrder(quote *Order, verbose bool) ([]map[string]string, *Order) {
	order_type := quote.Type
	order_in_book := &Order{}
	var trades []map[string]string

	orderBook.UpdateTime()
	quote.UpdatedAt = orderBook.Time
	orderBook.NextOrderID++

	if order_type == "market" {
		trades = orderBook.ProcessMarketOrder(quote, verbose)
	} else {
		trades, order_in_book = orderBook.ProcessLimitOrder(quote, verbose)
	}
	return trades, order_in_book
}

func (orderBook *OrderBook) ProcessOrderList(side string, orderList *OrderList, quantityStillToTrade *big.Int, quote *Order, verbose bool) (*big.Int, []map[string]string) {
	quantityToTrade := quantityStillToTrade
	var trades []map[string]string

	for orderList.Length() > 0 && quantityToTrade.Cmp(Zero()) > 0 {
		headOrder := orderList.HeadOrder()
		tradedPrice := headOrder.Price
		var newBookQuantity *big.Int
		var tradedQuantity *big.Int

		if quantityToTrade.Cmp(headOrder.Quantity) < 0 {
			tradedQuantity = quantityToTrade
			// Do the transaction
			newBookQuantity = Sub(headOrder.Quantity, quantityToTrade)
			headOrder.UpdateQuantity(newBookQuantity, headOrder.UpdatedAt)
			quantityToTrade = Zero()
		} else if quantityToTrade.Cmp(headOrder.Quantity) == 0 {
			tradedQuantity = quantityToTrade
			if side == Bid {
				orderBook.Bids.RemoveOrderById(strconv.FormatUint(headOrder.OrderID, 10))
			} else {
				orderBook.Asks.RemoveOrderById(strconv.FormatUint(headOrder.OrderID, 10))
			}
			quantityToTrade = Zero()

		} else {
			tradedQuantity = headOrder.Quantity
			if side == Bid {
				orderBook.Bids.RemoveOrderById(strconv.FormatUint(headOrder.OrderID, 10))
			} else {
				orderBook.Asks.RemoveOrderById(strconv.FormatUint(headOrder.OrderID, 10))
			}
		}

		if verbose {
			log.Debug("TRADE: ", "Time", orderBook.Time, "Price", tradedPrice.String(), "Quantity", tradedQuantity.String(), "TradeID", headOrder.ExchangeAddress.Hex(), "Matching TradeID", quote.ExchangeAddress.Hex())
		}

		transactionRecord := make(map[string]string)
		transactionRecord["timestamp"] = strconv.FormatUint(orderBook.Time, 10)
		transactionRecord["price"] = tradedPrice.String()
		transactionRecord["quantity"] = tradedQuantity.String()
		transactionRecord["time"] = strconv.FormatUint(orderBook.Time, 10)

		trades = append(trades, transactionRecord)
	}
	return quantityToTrade, trades
}

func (orderBook *OrderBook) CancelOrder(order *Order) {
	orderBook.UpdateTime()
	orderId := order.OrderID

	if order.Side == Bid {
		if orderBook.Bids.OrderExist(strconv.FormatUint(orderId, 10)) {
			orderBook.Bids.RemoveOrderById(strconv.FormatUint(orderId, 10))
		}
	} else {
		if orderBook.Asks.OrderExist(strconv.FormatUint(orderId, 10)) {
			orderBook.Asks.RemoveOrderById(strconv.FormatUint(orderId, 10))
		}
	}
}

func (orderBook *OrderBook) ModifyOrder(quoteUpdate *Order, orderId uint64) {
	orderBook.UpdateTime()

	side := quoteUpdate.Side
	quoteUpdate.OrderID = orderId
	quoteUpdate.UpdatedAt = orderBook.Time

	if side == Bid {
		if orderBook.Bids.OrderExist(strconv.FormatUint(orderId, 10)) {
			orderBook.Bids.UpdateOrder(quoteUpdate)
		}
	} else {
		if orderBook.Asks.OrderExist(strconv.FormatUint(orderId, 10)) {
			orderBook.Asks.UpdateOrder(quoteUpdate)
		}
	}
}

func (orderBook *OrderBook) VolumeAtPrice(side string, price *big.Int) *big.Int {
	if side == Bid {
		volume := Zero()
		if orderBook.Bids.PriceExist(price) {
			volume = orderBook.Bids.PriceList(price).Volume
		}

		return volume

	} else {
		volume := Zero()
		if orderBook.Asks.PriceExist(price) {
			volume = orderBook.Asks.PriceList(price).Volume
		}
		return volume
	}
}

//// commit everything by trigger db.Commit, later we can map custom encode and decode based on item
//func (orderBook *OrderBook) Commit() error {
//	return orderBook.db.Commit()
//}

func (orderBook *OrderBook) Save() error {
	err := orderBook.Asks.Save()
	if err != nil {
		return err
	}
	err = orderBook.Bids.Save()
	if err != nil {
		return err
	}

	value, err := EncodeBytesItem(orderBook)
	if err != nil {
		log.Error("Can't save orderbook", "value", value, "err", err)
		return err
	}
	log.Debug("Save orderbook ", "key", orderBook.Key, "value", value, "orderbook", orderBook)
	return orderBook.Db.Put(orderBook.Key, value)
}

func (orderBook *OrderBook) Restore() (*OrderBook, error) {
	val, err := orderBook.Db.Get(orderBook.Key, &OrderBook{})
	if err != nil {
		log.Error("Can't restore orderbook", "err", err)
		return nil, err
	}
	orderBook = val.(*OrderBook)
	log.Debug("orderbook restored", "orderbook", orderBook, "val", val, "val.(*OrderBook)", val.(*OrderBook))
	return orderBook, nil
}

func (orderBook *OrderBook) UpdateOrder(order *Order) {
	orderBook.ModifyOrder(order, order.OrderID)
}

// Save order pending into orderbook tree.
func (orderBook *OrderBook) SaveOrderPending(order *Order) error {
	zero := Zero()
	orderBook.UpdateTime()
	// if we do not use auto-increment orderid, we must set price slot to avoid conflict
	orderBook.NextOrderID++

	if order.Side == Bid {
		if order.Quantity.Cmp(zero) > 0 {
			order.OrderID = orderBook.NextOrderID
			if orderBook.Bids == nil {
				log.Error("orderbook bids is corrupted")
			}
			orderBook.Bids.InsertOrder(order)
		}
	} else {
		if order.Quantity.Cmp(zero) > 0 {
			order.OrderID = orderBook.NextOrderID
			if orderBook.Asks == nil {
				log.Error("orderbook asks is corrupted")
			}
			orderBook.Asks.InsertOrder(order)
		}
	}

	// save changes to orderbook
	err := orderBook.Save()
	if err != nil {
		return err
	}
	return nil
}

//
//func (orderBook *OrderBook) GetOrderIDFromBook(key []byte) uint64 {
//	orderSlot := new(big.Int).SetBytes(key)
//	return Sub(orderSlot, orderBook.slot).Uint64()
//}
//
//func (orderBook *OrderBook) GetOrderIDFromKey(key []byte) []byte {
//	orderSlot := new(big.Int).SetBytes(key)
//	return common.BigToHash(Add(orderBook.slot, orderSlot)).Bytes()
//}
//
//func (orderBook *OrderBook) GetOrder(key []byte) *Order {
//	if orderBook.db.IsEmptyKey(key) {
//		return nil
//	}
//	storedKey := orderBook.GetOrderIDFromKey(key)
//	orderItem := &OrderItem{}
//	val, err := orderBook.db.Get(storedKey, orderItem)
//	if err != nil {
//		log.Error("Key not found", "key", storedKey, "err", err)
//		return nil
//	}
//
//	order := &Order{
//		Item: val.(*OrderItem),
//		Key:  key,
//	}
//	return order
//}
//
//func (orderBook *OrderBook) String(startDepth int) string {
//	tabs := strings.Repeat("\t", startDepth)
//	return fmt.Sprintf("%s{\n\t%sName: %s\n\t%sTimestamp: %d\n\t%sNextOrderID: %d\n\t%sBids: %s\n\t%sAsks: %s\n%s}\n",
//		tabs,
//		tabs, orderBook.Item.Name, tabs, orderBook.Item.Timestamp, tabs, orderBook.Item.NextOrderID,
//		tabs, orderBook.Bids.String(startDepth+1), tabs, orderBook.Asks.String(startDepth+1),
//		tabs)
//}
//
//// processMarketOrder : process the market order
//func (orderBook *OrderBook) processMarketOrder(order *OrderItem, verbose bool) []map[string]string {
//	var trades []map[string]string
//	quantityToTrade := order.Quantity
//	side := order.Side
//	var newTrades []map[string]string
//	// speedup the comparison, do not assign because it is pointer
//	zero := Zero()
//	if side == Bid {
//		for quantityToTrade.Cmp(zero) > 0 && orderBook.Asks.NotEmpty() {
//			bestPriceAsks := orderBook.Asks.MinPriceList()
//			quantityToTrade, newTrades = orderBook.processOrderList(Ask, bestPriceAsks, quantityToTrade, order, verbose)
//			trades = append(trades, newTrades...)
//		}
//	} else {
//		for quantityToTrade.Cmp(zero) > 0 && orderBook.Bids.NotEmpty() {
//			bestPriceBids := orderBook.Bids.MaxPriceList()
//			quantityToTrade, newTrades = orderBook.processOrderList(Bid, bestPriceBids, quantityToTrade, order, verbose)
//			trades = append(trades, newTrades...)
//		}
//	}
//	return trades
//}
//
//// processLimitOrder : process the limit order, can change the quote
//// If not care for performance, we should make a copy of quote to prevent further reference problem
//func (orderBook *OrderBook) processLimitOrder(order *OrderItem, verbose bool) ([]map[string]string, *OrderItem) {
//	var trades []map[string]string
//	quantityToTrade := order.Quantity
//	side := order.Side
//	price := order.Price
//
//	var newTrades []map[string]string
//	var orderInBook *OrderItem
//	// speedup the comparison, do not assign because it is pointer
//	zero := Zero()
//
//	if side == Bid {
//		minPrice := orderBook.Asks.MinPrice()
//		for quantityToTrade.Cmp(zero) > 0 && orderBook.Asks.NotEmpty() && price.Cmp(minPrice) >= 0 {
//			bestPriceAsks := orderBook.Asks.MinPriceList()
//			quantityToTrade, newTrades = orderBook.processOrderList(Ask, bestPriceAsks, quantityToTrade, order, verbose)
//			trades = append(trades, newTrades...)
//			minPrice = orderBook.Asks.MinPrice()
//		}
//
//		if quantityToTrade.Cmp(zero) > 0 {
//			order.OrderID = orderBook.Item.NextOrderID
//			order.Quantity = quantityToTrade
//			orderBook.Bids.InsertOrder(order)
//			orderInBook = order
//		}
//
//	} else {
//		maxPrice := orderBook.Bids.MaxPrice()
//		for quantityToTrade.Cmp(zero) > 0 && orderBook.Bids.NotEmpty() && price.Cmp(maxPrice) <= 0 {
//			bestPriceBids := orderBook.Bids.MaxPriceList()
//			quantityToTrade, newTrades = orderBook.processOrderList(Bid, bestPriceBids, quantityToTrade, order, verbose)
//			trades = append(trades, newTrades...)
//			maxPrice = orderBook.Bids.MaxPrice()
//		}
//
//		if quantityToTrade.Cmp(zero) > 0 {
//			order.OrderID = orderBook.Item.NextOrderID
//			order.Quantity = quantityToTrade
//			orderBook.Asks.InsertOrder(order)
//			orderInBook = order
//		}
//	}
//	return trades, orderInBook
//}
//
//// ProcessOrder : process the order
//func (orderBook *OrderBook) ProcessOrder(order *OrderItem, verbose bool) ([]map[string]string, *OrderItem) {
//	orderType := order.Type
//	var orderInBook *OrderItem
//	var trades []map[string]string
//
//	//orderBook.UpdateTime()
//	//// if we do not use auto-increment orderid, we must set price slot to avoid conflict
//	//orderBook.Item.NextOrderID++
//
//	if orderType == Market {
//		trades = orderBook.processMarketOrder(order, verbose)
//	} else {
//		trades, orderInBook = orderBook.processLimitOrder(order, verbose)
//	}
//
//	// update orderBook
//	orderBook.Save()
//
//	return trades, orderInBook
//}
//
//// processOrderList : process the order list
//func (orderBook *OrderBook) processOrderList(side string, orderList *OrderList, quantityStillToTrade *big.Int, order *OrderItem, verbose bool) (*big.Int, []map[string]string) {
//	quantityToTrade := CloneBigInt(quantityStillToTrade)
//	var trades []map[string]string
//	// speedup the comparison, do not assign because it is pointer
//	zero := Zero()
//	for orderList.Item.Length > 0 && quantityToTrade.Cmp(zero) > 0 {
//
//		headOrder := orderList.GetOrder(orderList.Item.HeadOrder)
//		if headOrder == nil {
//			panic("headOrder is null")
//		}
//
//		tradedPrice := CloneBigInt(headOrder.Item.Price)
//
//		var newBookQuantity *big.Int
//		var tradedQuantity *big.Int
//
//		if IsStrictlySmallerThan(quantityToTrade, headOrder.Item.Quantity) {
//			tradedQuantity = CloneBigInt(quantityToTrade)
//			// Do the transaction
//			newBookQuantity = Sub(headOrder.Item.Quantity, quantityToTrade)
//			headOrder.UpdateQuantity(orderList, newBookQuantity, headOrder.Item.UpdatedAt)
//			quantityToTrade = Zero()
//
//		} else if IsEqual(quantityToTrade, headOrder.Item.Quantity) {
//			tradedQuantity = CloneBigInt(quantityToTrade)
//			if side == Bid {
//				orderBook.Bids.RemoveOrder(headOrder)
//			} else {
//				orderBook.Asks.RemoveOrder(headOrder)
//			}
//			quantityToTrade = Zero()
//
//		} else {
//			tradedQuantity = CloneBigInt(headOrder.Item.Quantity)
//			if side == Bid {
//				orderBook.Bids.RemoveOrder(headOrder)
//			} else {
//				orderBook.Asks.RemoveOrderFromOrderList(headOrder, orderList)
//			}
//		}
//
//		if verbose {
//			log.Info("TRADE", "Timestamp", orderBook.Item.Timestamp, "Price", tradedPrice, "Quantity", tradedQuantity, "TradeID", headOrder.Item.ExchangeAddress.Hex(), "Matching TradeID", order.ExchangeAddress.Hex())
//		}
//
//		transactionRecord := make(map[string]string)
//		transactionRecord["timestamp"] = strconv.FormatUint(orderBook.Item.Timestamp, 10)
//		transactionRecord["price"] = tradedPrice.String()
//		transactionRecord["quantity"] = tradedQuantity.String()
//
//		trades = append(trades, transactionRecord)
//	}
//	return quantityToTrade, trades
//}
//
//// CancelOrder : cancel the order, just need ID, side and price, of course order must belong
//// to a price point as well
//func (orderBook *OrderBook) CancelOrder(order *OrderItem) error {
//	orderBook.UpdateTime()
//	key := GetKeyFromBig(big.NewInt(int64(order.OrderID)))
//	var err error
//	if order.Side == Bid {
//		orderInDB := orderBook.Bids.GetOrder(key, order.Price)
//		if orderInDB == nil || orderInDB.Item.Hash != order.Hash {
//			return fmt.Errorf("Can't cancel order as it doesn't exist - order: %v", order)
//		}
//		orderInDB.Item.Status = Cancel
//		_, err = orderBook.Bids.RemoveOrder(orderInDB)
//		if err != nil {
//			return err
//		}
//	} else {
//		orderInDB := orderBook.Asks.GetOrder(key, order.Price)
//		if orderInDB == nil || orderInDB.Item.Hash != order.Hash {
//			return fmt.Errorf("Can't cancel order as it doesn't exist - order: %v", order)
//		}
//		orderInDB.Item.Status = Cancel
//		_, err = orderBook.Asks.RemoveOrder(orderInDB)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
