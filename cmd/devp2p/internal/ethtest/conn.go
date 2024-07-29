// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/X7-Group/x7chain/crypto"
	"github.com/X7-Group/x7chain/eth/protocols/snap"
	"github.com/X7-Group/x7chain/eth/protocols/x7c"
	"github.com/X7-Group/x7chain/p2p"
	"github.com/X7-Group/x7chain/p2p/rlpx"
	"github.com/X7-Group/x7chain/rlp"
	"github.com/davecgh/go-spew/spew"
)

var (
	pretty = spew.ConfigState{
		Indent:                  "  ",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
		SortKeys:                true,
	}
	timeout = 2 * time.Second
)

// dial attempts to dial the given node and perform a handshake, returning the
// created Conn if successful.
func (s *Suite) dial() (*Conn, error) {
	key, _ := crypto.GenerateKey()
	return s.dialAs(key)
}

// dialAs attempts to dial a given node and perform a handshake using the given
// private key.
func (s *Suite) dialAs(key *ecdsa.PrivateKey) (*Conn, error) {
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", s.Dest.IP(), s.Dest.TCP()))
	if err != nil {
		return nil, err
	}
	conn := Conn{Conn: rlpx.NewConn(fd, s.Dest.Pubkey())}
	conn.ourKey = key
	_, err = conn.Handshake(conn.ourKey)
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.caps = []p2p.Cap{
		{Name: "x7c", Version: 67},
		{Name: "x7c", Version: 68},
	}
	conn.ourHighestProtoVersion = 68
	return &conn, nil
}

// dialSnap creates a connection with snap/1 capability.
func (s *Suite) dialSnap() (*Conn, error) {
	conn, err := s.dial()
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}
	conn.caps = append(conn.caps, p2p.Cap{Name: "snap", Version: 1})
	conn.ourHighestSnapProtoVersion = 1
	return conn, nil
}

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey                     *ecdsa.PrivateKey
	negotiatedProtoVersion     uint
	negotiatedSnapProtoVersion uint
	ourHighestProtoVersion     uint
	ourHighestSnapProtoVersion uint
	caps                       []p2p.Cap
}

// Read reads a packet from the connection.
func (c *Conn) Read() (uint64, []byte, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	code, data, _, err := c.Conn.Read()
	if err != nil {
		return 0, nil, err
	}
	return code, data, nil
}

// ReadMsg attempts to read a devp2p message with a specific code.
func (c *Conn) ReadMsg(proto Proto, code uint64, msg any) error {
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		got, data, err := c.Read()
		if err != nil {
			return err
		}
		if protoOffset(proto)+code == got {
			return rlp.DecodeBytes(data, msg)
		}
	}
}

// Write writes a x7c packet to the connection.
func (c *Conn) Write(proto Proto, code uint64, msg any) error {
	c.SetWriteDeadline(time.Now().Add(timeout))
	payload, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(protoOffset(proto)+code, payload)
	return err
}

// ReadEth reads an x7c sub-protocol wire message.
func (c *Conn) ReadEth() (any, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		code, data, _, err := c.Conn.Read()
		if err != nil {
			return nil, err
		}
		if code == pingMsg {
			c.Write(baseProto, pongMsg, []byte{})
			continue
		}
		if getProto(code) != ethProto {
			// Read until x7c message.
			continue
		}
		code -= baseProtoLen

		var msg any
		switch int(code) {
		case x7c.StatusMsg:
			msg = new(x7c.StatusPacket)
		case x7c.GetBlockHeadersMsg:
			msg = new(x7c.GetBlockHeadersPacket)
		case x7c.BlockHeadersMsg:
			msg = new(x7c.BlockHeadersPacket)
		case x7c.GetBlockBodiesMsg:
			msg = new(x7c.GetBlockBodiesPacket)
		case x7c.BlockBodiesMsg:
			msg = new(x7c.BlockBodiesPacket)
		case x7c.NewBlockMsg:
			msg = new(x7c.NewBlockPacket)
		case x7c.NewBlockHashesMsg:
			msg = new(x7c.NewBlockHashesPacket)
		case x7c.TransactionsMsg:
			msg = new(x7c.TransactionsPacket)
		case x7c.NewPooledTransactionHashesMsg:
			msg = new(x7c.NewPooledTransactionHashesPacket68)
		case x7c.GetPooledTransactionsMsg:
			msg = new(x7c.GetPooledTransactionsPacket)
		case x7c.PooledTransactionsMsg:
			msg = new(x7c.PooledTransactionsPacket)
		default:
			panic(fmt.Sprintf("unhandled x7c msg code %d", code))
		}
		if err := rlp.DecodeBytes(data, msg); err != nil {
			return nil, fmt.Errorf("unable to decode x7c msg: %v", err)
		}
		return msg, nil
	}
}

// ReadSnap reads a snap/1 response with the given id from the connection.
func (c *Conn) ReadSnap() (any, error) {
	c.SetReadDeadline(time.Now().Add(timeout))
	for {
		code, data, _, err := c.Conn.Read()
		if err != nil {
			return nil, err
		}
		if getProto(code) != snapProto {
			// Read until snap message.
			continue
		}
		code -= baseProtoLen + ethProtoLen

		var msg any
		switch int(code) {
		case snap.GetAccountRangeMsg:
			msg = new(snap.GetAccountRangePacket)
		case snap.AccountRangeMsg:
			msg = new(snap.AccountRangePacket)
		case snap.GetStorageRangesMsg:
			msg = new(snap.GetStorageRangesPacket)
		case snap.StorageRangesMsg:
			msg = new(snap.StorageRangesPacket)
		case snap.GetByteCodesMsg:
			msg = new(snap.GetByteCodesPacket)
		case snap.ByteCodesMsg:
			msg = new(snap.ByteCodesPacket)
		case snap.GetTrieNodesMsg:
			msg = new(snap.GetTrieNodesPacket)
		case snap.TrieNodesMsg:
			msg = new(snap.TrieNodesPacket)
		default:
			panic(fmt.Errorf("unhandled snap code: %d", code))
		}
		if err := rlp.DecodeBytes(data, msg); err != nil {
			return nil, fmt.Errorf("could not rlp decode message: %v", err)
		}
		return msg, nil
	}
}

// peer performs both the protocol handshake and the status message
// exchange with the node in order to peer with it.
func (c *Conn) peer(chain *Chain, status *x7c.StatusPacket) error {
	if err := c.handshake(); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	if err := c.statusExchange(chain, status); err != nil {
		return fmt.Errorf("status exchange failed: %v", err)
	}
	return nil
}

// handshake performs a protocol handshake with the node.
func (c *Conn) handshake() error {
	// Write hello to client.
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &protoHandshake{
		Version: 5,
		Caps:    c.caps,
		ID:      pub0,
	}
	if err := c.Write(baseProto, handshakeMsg, ourHandshake); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}
	// Read hello from client.
	code, data, err := c.Read()
	if err != nil {
		return fmt.Errorf("erroring reading handshake: %v", err)
	}
	switch code {
	case handshakeMsg:
		msg := new(protoHandshake)
		if err := rlp.DecodeBytes(data, &msg); err != nil {
			return fmt.Errorf("error decoding handshake msg: %v", err)
		}
		// Set snappy if version is at least 5.
		if msg.Version >= 5 {
			c.SetSnappy(true)
		}
		c.negotiateEthProtocol(msg.Caps)
		if c.negotiatedProtoVersion == 0 {
			return fmt.Errorf("could not negotiate x7c protocol (remote caps: %v, local x7c version: %v)", msg.Caps, c.ourHighestProtoVersion)
		}
		// If we require snap, verify that it was negotiated.
		if c.ourHighestSnapProtoVersion != c.negotiatedSnapProtoVersion {
			return fmt.Errorf("could not negotiate snap protocol (remote caps: %v, local snap version: %v)", msg.Caps, c.ourHighestSnapProtoVersion)
		}
		return nil
	default:
		return fmt.Errorf("bad handshake: got msg code %d", code)
	}
}

// negotiateEthProtocol sets the Conn's x7c protocol version to highest
// advertised capability from peer.
func (c *Conn) negotiateEthProtocol(caps []p2p.Cap) {
	var highestEthVersion uint
	var highestSnapVersion uint
	for _, capability := range caps {
		switch capability.Name {
		case "x7c":
			if capability.Version > highestEthVersion && capability.Version <= c.ourHighestProtoVersion {
				highestEthVersion = capability.Version
			}
		case "snap":
			if capability.Version > highestSnapVersion && capability.Version <= c.ourHighestSnapProtoVersion {
				highestSnapVersion = capability.Version
			}
		}
	}
	c.negotiatedProtoVersion = highestEthVersion
	c.negotiatedSnapProtoVersion = highestSnapVersion
}

// statusExchange performs a `Status` message exchange with the given node.
func (c *Conn) statusExchange(chain *Chain, status *x7c.StatusPacket) error {
loop:
	for {
		code, data, err := c.Read()
		if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}
		switch code {
		case x7c.StatusMsg + protoOffset(ethProto):
			msg := new(x7c.StatusPacket)
			if err := rlp.DecodeBytes(data, &msg); err != nil {
				return fmt.Errorf("error decoding status packet: %w", err)
			}
			if have, want := msg.Head, chain.blocks[chain.Len()-1].Hash(); have != want {
				return fmt.Errorf("wrong head block in status, want:  %#x (block %d) have %#x",
					want, chain.blocks[chain.Len()-1].NumberU64(), have)
			}
			if have, want := msg.TD.Cmp(chain.TD()), 0; have != want {
				return fmt.Errorf("wrong TD in status: have %v want %v", have, want)
			}
			if have, want := msg.ForkID, chain.ForkID(); !reflect.DeepEqual(have, want) {
				return fmt.Errorf("wrong fork ID in status: have %v, want %v", have, want)
			}
			if have, want := msg.ProtocolVersion, c.ourHighestProtoVersion; have != uint32(want) {
				return fmt.Errorf("wrong protocol version: have %v, want %v", have, want)
			}
			break loop
		case discMsg:
			var msg []p2p.DiscReason
			if rlp.DecodeBytes(data, &msg); len(msg) == 0 {
				return errors.New("invalid disconnect message")
			}
			return fmt.Errorf("disconnect received: %v", pretty.Sdump(msg))
		case pingMsg:
			// TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
			c.Write(baseProto, pongMsg, nil)
		default:
			return fmt.Errorf("bad status message: code %d", code)
		}
	}
	// make sure x7c protocol version is set for negotiation
	if c.negotiatedProtoVersion == 0 {
		return errors.New("x7c protocol version must be set in Conn")
	}
	if status == nil {
		// default status message
		status = &x7c.StatusPacket{
			ProtocolVersion: uint32(c.negotiatedProtoVersion),
			NetworkID:       chain.config.ChainID.Uint64(),
			TD:              chain.TD(),
			Head:            chain.blocks[chain.Len()-1].Hash(),
			Genesis:         chain.blocks[0].Hash(),
			ForkID:          chain.ForkID(),
		}
	}
	if err := c.Write(ethProto, x7c.StatusMsg, status); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}
	return nil
}