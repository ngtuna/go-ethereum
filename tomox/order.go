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
	Quantity        *big.Int       `json:"quantity,omitempty"`
	Price           *big.Int       `json:"price,omitempty"`
	ExchangeAddress common.Address `json:"exchangeAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	BaseToken       common.Address `json:"baseToken,omitempty"`
	QuoteToken      common.Address `json:"quoteToken,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	Hash            common.Hash    `json:"hash,omitempty"`
	Signature       *Signature     `json:"signature,omitempty"`
	FilledAmount    *big.Int       `json:"filledAmount,omitempty"`
	Nonce           *big.Int       `json:"nonce,omitempty"`
	MakeFee         *big.Int       `json:"makeFee,omitempty"`
	TakeFee         *big.Int       `json:"takeFee,omitempty"`
	PairName        string         `json:"pairName,omitempty"`
	CreatedAt       uint64         `json:"createdAt,omitempty"`
	UpdatedAt       uint64         `json:"updatedAt,omitempty"`
	OrderID         uint64         `json:"orderID,omitempty"`
	// *OrderMeta
	NextOrder *Order     `json:"-"`
	PrevOrder *Order     `json:"-"`
	OrderList *OrderList `json:"-"`
	Key  []byte `json:"orderID"`
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

// NewOrder : create new order with quote ( can be ethereum address )
func NewOrder(order *Order, orderList *OrderList) *Order {
	order.OrderList = orderList
	return order
}

func (o *Order) UpdateQuantity(newQuantity *big.Int, newTimestamp uint64) {
	if newQuantity.Cmp(o.Quantity) > 0 && o.OrderList.tailOrder != o {
		o.OrderList.MoveToTail(o)
	}
	o.OrderList.volume = Sub(o.OrderList.volume, Sub(o.Quantity, newQuantity))
	log.Debug("Updated quantity", "old quantity", o.Quantity, "new quantity", newQuantity)
	o.UpdatedAt = newTimestamp
	o.Quantity = newQuantity
}
