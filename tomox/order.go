package tomox

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// Signature struct
type Signature struct {
	V byte
	R common.Hash
	S common.Hash
}

type SignatureRecord struct {
	V byte   `json:"V" bson:"V"`
	R string `json:"R" bson:"R"`
	S string `json:"S" bson:"S"`
}

// Order: info that will be store in database
type Order struct {
	quantity        *big.Int       `json:"quantity,omitempty"`
	price           *big.Int       `json:"price,omitempty"`
	exchangeAddress common.Address `json:"exchangeAddress,omitempty"`
	userAddress     common.Address `json:"userAddress,omitempty"`
	baseToken       common.Address `json:"baseToken,omitempty"`
	quoteToken      common.Address `json:"quoteToken,omitempty"`
	status          string         `json:"status,omitempty"`
	side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	hash            common.Hash    `json:"hash,omitempty"`
	signature       *Signature     `json:"signature,omitempty"`
	filledAmount    *big.Int       `json:"filledAmount,omitempty"`
	nonce           *big.Int       `json:"nonce,omitempty"`
	makeFee         *big.Int       `json:"makeFee,omitempty"`
	takeFee         *big.Int       `json:"takeFee,omitempty"`
	pairName        string         `json:"pairName,omitempty"`
	createdAt       uint64         `json:"createdAt,omitempty"`
	updatedAt       uint64         `json:"updatedAt,omitempty"`
	orderID         uint64         `json:"orderID,omitempty"`
	// *OrderMeta
	nextOrder *Order     `json:"-"`
	prevOrder *Order     `json:"-"`
	orderList *OrderList `json:"-"`
}

type OrderBSON struct {
	Quantity        string           `json:"quantity,omitempty" bson:"quantity"`
	Price           string           `json:"price,omitempty" bson:"price"`
	ExchangeAddress string           `json:"exchangeAddress,omitempty" bson:"exchangeAddress"`
	UserAddress     string           `json:"userAddress,omitempty" bson:"userAddress"`
	BaseToken       string           `json:"baseToken,omitempty" bson:"baseToken"`
	QuoteToken      string           `json:"quoteToken,omitempty" bson:"quoteToken"`
	Status          string           `json:"status,omitempty" bson:"status"`
	Side            string           `json:"side,omitempty" bson:"side"`
	Type            string           `json:"type,omitempty" bson:"type"`
	Hash            string           `json:"hash,omitempty" bson:"hash"`
	Signature       *SignatureRecord `json:"signature,omitempty" bson:"signature"`
	FilledAmount    string           `json:"filledAmount,omitempty" bson:"filledAmount"`
	Nonce           string           `json:"nonce,omitempty" bson:"nonce"`
	MakeFee         string           `json:"makeFee,omitempty" bson:"makeFee"`
	TakeFee         string           `json:"takeFee,omitempty" bson:"takeFee"`
	PairName        string           `json:"pairName,omitempty" bson:"pairName"`
	CreatedAt       string           `json:"createdAt,omitempty" bson:"createdAt"`
	UpdatedAt       string           `json:"updatedAt,omitempty" bson:"updatedAt"`
	OrderID         string           `json:"orderID,omitempty" bson:"orderID"`
	NextOrder       string           `json:"nextOrder,omitempty" bson:"nextOrder"`
	PrevOrder       string           `json:"prevOrder,omitempty" bson:"prevOrder"`
	OrderList       string           `json:"orderList,omitempty" bson:"orderList"`
}

func (o *Order) NextOrder() *Order {
	return o.nextOrder
}

func (o *Order) PrevOrder() *Order {
	return o.prevOrder
}

// NewOrder : create new order with quote ( can be ethereum address )
func NewOrder(order *Order, orderList *OrderList) *Order {
	order.orderList = orderList
	return order
}

func (o *Order) UpdateQuantity(newQuantity *big.Int, newTimestamp uint64) {
	if newQuantity.Cmp(o.quantity) > 0 && o.orderList.tailOrder != o {
		o.orderList.MoveToTail(o)
	}
	o.orderList.volume = Sub(o.orderList.volume, Sub(o.quantity, newQuantity))
	log.Debug("Updated quantity", "old quantity", o.quantity, "new quantity", newQuantity)
	o.updatedAt = newTimestamp
	o.quantity = newQuantity
}
