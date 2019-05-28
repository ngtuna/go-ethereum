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
	Quantity        *big.Int
	Price           *big.Int
	ExchangeAddress common.Address
	UserAddress     common.Address
	BaseToken       common.Address
	QuoteToken      common.Address
	Status          string
	Side            string
	Type            string
	Hash            common.Hash
	Signature       *Signature
	FilledAmount    *big.Int
	Nonce           *big.Int
	MakeFee         *big.Int
	TakeFee         *big.Int
	PairName        string
	CreatedAt       uint64
	UpdatedAt       uint64
	OrderID         uint64
	// *OrderMeta
	NextOrder *Order     `rlp:"nil"`
	PrevOrder *Order     `rlp:"nil"`
	OrderList *OrderList `rlp:"nil"`
	Key       []byte
	Db        TomoXDao
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
	Key             string           `json:"key" bson:"key"`
}

// NewOrder : create new order with quote ( can be ethereum address )
func NewOrder(order *Order, orderList *OrderList) *Order {
	order.OrderList = orderList
	return order
}

func (o *Order) UpdateQuantity(newQuantity *big.Int, newTimestamp uint64) {
	if newQuantity.Cmp(o.Quantity) > 0 && o.OrderList.TOrder != o {
		o.OrderList.MoveToTail(o)
	}
	o.OrderList.Volume = Sub(o.OrderList.Volume, Sub(o.Quantity, newQuantity))
	log.Debug("Updated quantity", "old quantity", o.Quantity, "new quantity", newQuantity)
	o.UpdatedAt = newTimestamp
	o.Quantity = newQuantity
}
