package tomox

import (
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"encoding/json"
)

type OrderTreeStore struct {
	PriceTreeByte []byte
	PriceMapByte []byte
	OrderMapByte []byte
	//PriceMap  map[string]*OrderList // Dictionary containing price : OrderList object
	//OrderMap  map[string]*Order     // Dictionary containing orderId : Order object
	Volume    *big.Int              // Contains total quantity from all Orders in tree
	NumOrders uint64                // Contains count of Orders in tree
	Depth     uint64                // Number of different prices in tree (http://en.wikipedia.org/wiki/Order_book_(trading)#Book_Depth)
	Slot      *big.Int
	Key       []byte
}

// Order: info that will be store in database
type OrderStore struct {
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
}

type OrderListStore struct {
	HOrder *Order `rlp:"nil"`
	TOrder *Order `rlp:"nil"`
	Len    uint64
	Volume    *big.Int
	LastOrder *Order `rlp:"nil"`
	Price     *big.Int
	Key       []byte
	Slot      *big.Int
}

func prepareOrderTreeToStore(ot *OrderTree) (*OrderTreeStore, error) {
	otStore := &OrderTreeStore{
		Volume: ot.Volume,
		NumOrders: ot.NumOrders,
		Depth: ot.Depth,
		Slot: ot.Slot,
		Key: ot.Key,
	}
	data, err := ot.PriceTree.ToJSON()
	if err != nil {
		return nil, err
	}
	otStore.PriceTreeByte = data
	//PriceMap: ot.PriceMap,
	//OrderMap: ot.OrderMap,
	data, err = json.Marshal(ot.PriceMap)
	if err != nil {
		return nil, err
	}
	otStore.PriceMapByte = data
	data, err = json.Marshal(ot.OrderMap)
	if err != nil {
		return nil, err
	}
	otStore.OrderMapByte = data
	return otStore, nil
}

func prepareOrderToStore(o *Order) (*OrderStore, error) {
	return &OrderStore{
		Quantity: o.Quantity,
		Price: o.Price,
		ExchangeAddress: o.ExchangeAddress,
		UserAddress: o.UserAddress,
		BaseToken: o.BaseToken,
		QuoteToken: o.QuoteToken,
		Status: o.Status,
		Side: o.Side,
		Type: o.Type,
		Hash: o.Hash,
		Signature: o.Signature,
		FilledAmount: o.FilledAmount,
		Nonce: o.Nonce,
		MakeFee: o.MakeFee,
		TakeFee: o.TakeFee,
		PairName: o.PairName,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
		OrderID: o.OrderID,
		// *OrderMeta
		NextOrder: o.NextOrder,
		PrevOrder: o.PrevOrder,
		OrderList: o.OrderList,
		Key: o.Key,
	}, nil
}

func prepareOrderListToStore(ol *OrderList) (*OrderListStore, error) {
	return &OrderListStore{
		HOrder: ol.HOrder,
		TOrder: ol.TOrder,
		Len: ol.Len,
		Volume: ol.Volume,
		LastOrder: ol.LastOrder,
		Price: ol.Price,
		Key: ol.Key,
		Slot: ol.Slot,
	}, nil
}

func EncodeBytesItem(val interface{}) ([]byte, error) {
	switch val.(type) {
	case *Order:
		o := val.(*Order)
		oStore, err := prepareOrderToStore(o)
		if err != nil {
			return nil, err
		}
		return rlp.EncodeToBytes(oStore)
	case *OrderList:
		ol := val.(*OrderList)
		olStore, err := prepareOrderListToStore(ol)
		if err != nil {
			return nil, err
		}
		return rlp.EncodeToBytes(olStore)
	case *OrderTree:
		ot := val.(*OrderTree)
		otStore, err := prepareOrderTreeToStore(ot)
		if err != nil {
			return nil, err
		}
		return rlp.EncodeToBytes(otStore)
	case *OrderBook:
		return rlp.EncodeToBytes(val.(*OrderBook))
	default:
		return rlp.EncodeToBytes(val)
	}
}

func restoreOrderTree(ot *OrderTree, out *OrderTreeStore) (error) {
	//ot := &OrderTree{
	//	//PriceMap: out.PriceMap,
	//	//OrderMap: out.OrderMap,
	//	Volume: out.Volume,
	//	NumOrders: out.NumOrders,
	//	Depth: out.Depth,
	//	Slot: out.Slot,
	//	Key: out.Key,
	//}
	ot.Volume  = out.Volume
	ot.NumOrders = out.NumOrders
	ot.Depth = out.Depth
	ot.Slot = out.Slot
	ot.Key = out.Key

	data := out.PriceTreeByte
	err := ot.PriceTree.FromJSON(data)
	if err != nil {
		return err
	}
	data = out.PriceMapByte
	err = json.Unmarshal(data, ot.PriceMap)
	if err != nil {
		return err
	}
	data = out.OrderMapByte
	err = json.Unmarshal(data, ot.OrderMap)
	if err != nil {
		return err
	}

	return nil
}

func restoreOrder(o *Order, out *OrderStore) error {
	o.Quantity= out.Quantity
	o.Price = out.Price
	o.ExchangeAddress = out.ExchangeAddress
	o.UserAddress = out.UserAddress
	o.BaseToken = out.BaseToken
	o.QuoteToken = out.QuoteToken
	o.Status = out.Status
	o.Side = out.Side
	o.Type = out.Type
	o.Hash = out.Hash
	o.Signature = out.Signature
	o.FilledAmount = out.FilledAmount
	o.Nonce = out.Nonce
	o.MakeFee = out.MakeFee
	o.TakeFee = out.TakeFee
	o.PairName = out.PairName
	o.CreatedAt = out.CreatedAt
	o.UpdatedAt = out.UpdatedAt
	o.OrderID = out.OrderID
	o.NextOrder = out.NextOrder
	o.PrevOrder = out.PrevOrder
	o.OrderList = out.OrderList
	o.Key = out.Key
	return nil
}

func restoreOrderList(ol *OrderList, out *OrderListStore) error {
	ol.HOrder = out.HOrder
	ol.TOrder = out.TOrder
	ol.Len = out.Len
	ol.Volume = out.Volume
	ol.LastOrder = out.LastOrder
	ol.Price = out.Price
	ol.Key = out.Key
	ol.Slot = out.Slot
	return nil
}

func DecodeBytesItem(bytes []byte, val interface{}) (interface{}, error) {

	switch val.(type) {
	case *Order:
		out := &OrderStore{}
		o := val.(*Order)
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		err = restoreOrder(o, out)
		if err != nil {
			return nil, err
		}
		return o, nil
	case *OrderList:
		out := &OrderListStore{}
		ol := val.(*OrderList)
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		err = restoreOrderList(ol, out)
		if err != nil {
			return nil, err
		}
		return ol, nil
	case *OrderTree:
		out := &OrderTreeStore{}
		ot := val.(*OrderTree)
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		err = restoreOrderTree(ot, out)
		if err != nil {
			return nil, err
		}
		return ot, nil
	case *OrderBook:
		var out OrderBook
		err := rlp.DecodeBytes(bytes, &out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	default:
		return nil, errors.New("type is not supported")
	}

}
