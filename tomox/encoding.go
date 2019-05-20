package tomox

import (
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
	"github.com/ethereum/go-ethereum/log"
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

func DecodeBytesItem(bytes []byte, val interface{}) (interface{}, error) {

	switch val.(type) {
	case *Order:
		out := &Order{}
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		return out, nil
	case *OrderList:
		out := &OrderList{}
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		return out, nil
	case *OrderTree:
		out := &OrderTree{}
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		return out, nil
	case *OrderBook:
		out := &OrderBook{}
		err := rlp.DecodeBytes(bytes, out)
		if err != nil {
			return nil, err
		}
		log.Debug("decode debug", "out", out)
		return out, nil
	default:
		return nil, errors.New("type is not supported")
	}

}
