package tomox

import (
	"github.com/ethereum/go-ethereum/rlp"
)

func EncodeBytesItem(val interface{}) ([]byte, error) {

	switch val.(type) {
	case *Order:
		return rlp.EncodeToBytes(val.(*Order))
	case *OrderList:
		return rlp.EncodeToBytes(val.(*OrderList))
	case *OrderTree:
		return rlp.EncodeToBytes(val.(*OrderTree))
	case *OrderBook:
		return rlp.EncodeToBytes(val.(*OrderBook))
	default:
		return rlp.EncodeToBytes(val)
	}
}

func DecodeBytesItem(bytes []byte, val interface{}) error {

	switch val.(type) {
	case *Order:
		return rlp.DecodeBytes(bytes, val.(*Order))
	case *OrderList:
		return rlp.DecodeBytes(bytes, val.(*OrderList))
	case *OrderTree:
		return rlp.DecodeBytes(bytes, val.(*OrderTree))
	case *OrderBook:
		return rlp.DecodeBytes(bytes, val.(*OrderBook))
	default:
		return rlp.DecodeBytes(bytes, val)
	}

}
