package peer

import (
	"encoding/binary"
	"net"
	"strconv"
	"sync"

	"github.com/iotaledger/goshimmer/packages/events"
	"github.com/iotaledger/goshimmer/packages/identity"
	"github.com/iotaledger/goshimmer/packages/network"
	"github.com/iotaledger/goshimmer/plugins/autopeering/protocol/types"
	"github.com/iotaledger/goshimmer/plugins/autopeering/types/salt"
	"github.com/pkg/errors"
)

type Peer struct {
	Identity     *identity.Identity
	Address      net.IP
	PeeringPort  uint16
	GossipPort   uint16
	Salt         *salt.Salt
	Conn         *network.ManagedConnection
	connectMutex sync.Mutex
}

func Unmarshal(data []byte) (*Peer, error) {
	if len(data) < MARSHALLED_TOTAL_SIZE {
		return nil, errors.New("size of marshalled peer is too small")
	}

	peer := &Peer{
		Identity: identity.NewIdentity(data[MARSHALLED_PUBLIC_KEY_START:MARSHALLED_PUBLIC_KEY_END]),
	}

	switch data[MARSHALLED_ADDRESS_TYPE_START] {
	case types.ADDRESS_TYPE_IPV4:
		peer.Address = net.IP(data[MARSHALLED_ADDRESS_START:MARSHALLED_ADDRESS_END]).To4()
	case types.ADDRESS_TYPE_IPV6:
		peer.Address = net.IP(data[MARSHALLED_ADDRESS_START:MARSHALLED_ADDRESS_END]).To16()
	}

	peer.PeeringPort = binary.BigEndian.Uint16(data[MARSHALLED_PEERING_PORT_START:MARSHALLED_PEERING_PORT_END])
	peer.GossipPort = binary.BigEndian.Uint16(data[MARSHALLED_GOSSIP_PORT_START:MARSHALLED_GOSSIP_PORT_END])

	if unmarshalledSalt, err := salt.Unmarshal(data[MARSHALLED_SALT_START:MARSHALLED_SALT_END]); err != nil {
		return nil, err
	} else {
		peer.Salt = unmarshalledSalt
	}

	return peer, nil
}

// sends data and
func (peer *Peer) Send(data []byte, protocol types.ProtocolType, responseExpected bool) (bool, error) {
	conn, dialed, err := peer.Connect(protocol)
	if err != nil {
		return false, err
	}

	if _, err := conn.Write(data); err != nil {
		return false, err
	}

	if dialed && !responseExpected {
		conn.Close()
	}

	return dialed, nil
}

func (peer *Peer) ConnectTCP() (*network.ManagedConnection, bool, error) {
	if peer.Conn == nil {
		peer.connectMutex.Lock()
		defer peer.connectMutex.Unlock()

		if peer.Conn == nil {
			conn, err := net.Dial("tcp", peer.Address.String()+":"+strconv.Itoa(int(peer.PeeringPort)))
			if err != nil {
				return nil, false, errors.New("error when connecting to " + peer.String() + ": " + err.Error())
			} else {
				peer.Conn = network.NewManagedConnection(conn)

				peer.Conn.Events.Close.Attach(events.NewClosure(func() {
					peer.Conn = nil
				}))

				return peer.Conn, true, nil
			}
		}
	}

	return peer.Conn, false, nil
}

func (peer *Peer) ConnectUDP() (*network.ManagedConnection, bool, error) {
	conn, err := net.Dial("udp", peer.Address.String()+":"+strconv.Itoa(int(peer.PeeringPort)))
	if err != nil {
		return nil, false, errors.New("error when connecting to " + peer.Address.String() + ": " + err.Error())
	}

	return network.NewManagedConnection(conn), true, nil
}

func (peer *Peer) Connect(protocol types.ProtocolType) (*network.ManagedConnection, bool, error) {
	switch protocol {
	case types.PROTOCOL_TYPE_TCP:
		return peer.ConnectTCP()
	case types.PROTOCOL_TYPE_UDP:
		return peer.ConnectUDP()
	default:
		return nil, false, errors.New("unsupported peering protocol in peer " + peer.Address.String())
	}
}

func (peer *Peer) Marshal() []byte {
	result := make([]byte, MARSHALLED_TOTAL_SIZE)

	copy(result[MARSHALLED_PUBLIC_KEY_START:MARSHALLED_PUBLIC_KEY_END],
		peer.Identity.PublicKey[:MARSHALLED_PUBLIC_KEY_SIZE])

	switch len(peer.Address) {
	case net.IPv4len:
		result[MARSHALLED_ADDRESS_TYPE_START] = types.ADDRESS_TYPE_IPV4
	case net.IPv6len:
		result[MARSHALLED_ADDRESS_TYPE_START] = types.ADDRESS_TYPE_IPV6
	default:
		panic("invalid address in peer")
	}

	copy(result[MARSHALLED_ADDRESS_START:MARSHALLED_ADDRESS_END], peer.Address.To16())

	binary.BigEndian.PutUint16(result[MARSHALLED_PEERING_PORT_START:MARSHALLED_PEERING_PORT_END], peer.PeeringPort)
	binary.BigEndian.PutUint16(result[MARSHALLED_GOSSIP_PORT_START:MARSHALLED_GOSSIP_PORT_END], peer.GossipPort)

	copy(result[MARSHALLED_SALT_START:MARSHALLED_SALT_END], peer.Salt.Marshal())

	return result
}

func (peer *Peer) String() string {
	if peer.Identity != nil {
		return peer.Address.String() + ":" + strconv.Itoa(int(peer.PeeringPort)) + " / " + peer.Identity.StringIdentifier
	} else {
		return peer.Address.String() + ":" + strconv.Itoa(int(peer.PeeringPort))
	}
}
