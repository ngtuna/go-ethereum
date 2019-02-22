// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the Whisper protocol Message element.

package whisperv6

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	mrand "math/rand"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// MessageParams specifies the exact way a message should be wrapped
// into an Envelope.
type MessageParams struct {
	TTL      uint32
	Topic    TopicType
	WorkTime uint32
	PoW      float64
	Payload  []byte
	Padding  []byte
}

// SentMessage represents an end-user data packet to transmit through the
// Whisper protocol. These are wrapped into Envelopes that need not be
// understood by intermediate nodes, just forwarded.
type sentMessage struct {
	Raw []byte
}

// ReceivedMessage represents a data packet to be received through the
// Whisper protocol and successfully decrypted.
type ReceivedMessage struct {
	Raw []byte

	Payload   []byte
	Padding   []byte
	Signature []byte
	Salt      []byte

	PoW   float64          // Proof of work as described in the Whisper spec
	Sent  uint32           // Time when the message was posted into the network
	TTL   uint32           // Maximum time to live allowed for the message
	Topic TopicType

	EnvelopeHash common.Hash // Message envelope hash to act as a unique id
}

func isMessageSigned(flags byte) bool {
	return (flags & signatureFlag) != 0
}

// NewSentMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewSentMessage(params *MessageParams) (*sentMessage, error) {
	const payloadSizeFieldMaxSize = 4
	msg := sentMessage{}
	msg.Raw = make([]byte, 1,
		flagsLength+payloadSizeFieldMaxSize+len(params.Payload)+len(params.Padding)+signatureLength+padSizeLimit)
	msg.Raw[0] = 0 // set all the flags to zero
	msg.addPayloadSizeField(params.Payload)
	msg.Raw = append(msg.Raw, params.Payload...)
	err := msg.appendPadding(params)
	return &msg, err
}

// addPayloadSizeField appends the auxiliary field containing the size of payload
func (msg *sentMessage) addPayloadSizeField(payload []byte) {
	fieldSize := getSizeOfPayloadSizeField(payload)
	field := make([]byte, 4)
	binary.LittleEndian.PutUint32(field, uint32(len(payload)))
	field = field[:fieldSize]
	msg.Raw = append(msg.Raw, field...)
	msg.Raw[0] |= byte(fieldSize)
}

// getSizeOfPayloadSizeField returns the number of bytes necessary to encode the size of payload
func getSizeOfPayloadSizeField(payload []byte) int {
	s := 1
	for i := len(payload); i >= 256; i /= 256 {
		s++
	}
	return s
}

// appendPadding appends the padding specified in params.
// If no padding is provided in params, then random padding is generated.
func (msg *sentMessage) appendPadding(params *MessageParams) error {
	if len(params.Padding) != 0 {
		// padding data was provided by the Dapp, just use it as is
		msg.Raw = append(msg.Raw, params.Padding...)
		return nil
	}

	rawSize := flagsLength + getSizeOfPayloadSizeField(params.Payload) + len(params.Payload)
	odd := rawSize % padSizeLimit
	paddingSize := padSizeLimit - odd
	pad := make([]byte, paddingSize)
	_, err := crand.Read(pad)
	if err != nil {
		return err
	}
	if !validateDataIntegrity(pad, paddingSize) {
		return errors.New("failed to generate random padding of size " + strconv.Itoa(paddingSize))
	}
	msg.Raw = append(msg.Raw, pad...)
	return nil
}

// generateSecureRandomData generates random data where extra security is required.
// The purpose of this function is to prevent some bugs in software or in hardware
// from delivering not-very-random data. This is especially useful for AES nonce,
// where true randomness does not really matter, but it is very important to have
// a unique nonce for every message.
func generateSecureRandomData(length int) ([]byte, error) {
	x := make([]byte, length)
	y := make([]byte, length)
	res := make([]byte, length)

	_, err := crand.Read(x)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(x, length) {
		return nil, errors.New("crypto/rand failed to generate secure random data")
	}
	_, err = mrand.Read(y)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(y, length) {
		return nil, errors.New("math/rand failed to generate secure random data")
	}
	for i := 0; i < length; i++ {
		res[i] = x[i] ^ y[i]
	}
	if !validateDataIntegrity(res, length) {
		return nil, errors.New("failed to generate secure random data")
	}
	return res, nil
}

// Wrap bundles the message into an Envelope to transmit over the network.
func (msg *sentMessage) Wrap(options *MessageParams) (envelope *Envelope, err error) {
	if options.TTL == 0 {
		options.TTL = DefaultTTL
	}

	envelope = NewEnvelope(options.TTL, options.Topic, msg)
	if err = envelope.Seal(options); err != nil {
		return nil, err
	}
	return envelope, nil
}

// ValidateAndParse checks the message validity and extracts the fields in case of success.
func (msg *ReceivedMessage) ValidateAndParse() bool {
	end := len(msg.Raw)
	if end < 1 {
		return false
	}

	if isMessageSigned(msg.Raw[0]) {
		end -= signatureLength
		if end <= 1 {
			return false
		}
		msg.Signature = msg.Raw[end : end+signatureLength]
	}

	beg := 1
	payloadSize := 0
	sizeOfPayloadSizeField := int(msg.Raw[0] & SizeMask) // number of bytes indicating the size of payload
	if sizeOfPayloadSizeField != 0 {
		payloadSize = int(bytesToUintLittleEndian(msg.Raw[beg : beg+sizeOfPayloadSizeField]))
		if payloadSize+1 > end {
			return false
		}
		beg += sizeOfPayloadSizeField
		msg.Payload = msg.Raw[beg : beg+payloadSize]
	}

	beg += payloadSize
	msg.Padding = msg.Raw[beg:end]
	return true
}

// hash calculates the SHA3 checksum of the message flags, payload size field, payload and padding.
func (msg *ReceivedMessage) hash() []byte {
	if isMessageSigned(msg.Raw[0]) {
		sz := len(msg.Raw) - signatureLength
		return crypto.Keccak256(msg.Raw[:sz])
	}
	return crypto.Keccak256(msg.Raw)
}
