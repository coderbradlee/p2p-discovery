package main

import (
	"./logger"
	"./utils"
	"fmt"
	"crypto/ecdsa"
	// "flag"
	// "fmt"
	// "net"
	// "os"

	// "github.com/ethereum/go-ethereum/cmd/utils"
	// "github.com/ethereum/go-ethereum/crypto"
	// "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	// "github.com/ethereum/go-ethereum/p2p/nat"
	// "github.com/ethereum/go-ethereum/p2p/netutil"
)

var cfg *config.Config

func log_init() {
	logger.SetConsole(cfg.Log.Console)
	logger.SetRollingFile(cfg.Log.Dir, cfg.Log.Name, cfg.Log.Num, cfg.Log.Size, logger.KB)
	//ALL，DEBUG，INFO，WARN，ERROR，FATAL，OFF
	logger.SetLevel(logger.ERROR)
	if configuration.Log.Level == "info" {
		logger.SetLevel(logger.INFO)
	} else if configuration.Log.Level == "error" {
		logger.SetLevel(logger.ERROR)
	}
}
func init() {
	cfg = &config.Config{}

	if !utils.LoadConfig("seeker.toml", cfg) {
		return
	}
	log_init()
}

type testTransport struct {
	id discover.NodeID
	*rlpx
	closeErr error
}
func newTestTransport(id discover.NodeID, fd net.Conn) p2p.transport {
	wrapped := newRLPX(fd).(*rlpx)
	wrapped.rw = newRLPXFrameRW(fd, secrets{
		MAC:        zero16,
		AES:        zero16,
		IngressMAC: sha3.NewKeccak256(),
		EgressMAC:  sha3.NewKeccak256(),
	})
	return &testTransport{id: id, rlpx: wrapped}
}

func startServer(id discover.NodeID, pf func(*Peer)) *p2p.Server {
	config := Config{
		Name:       "test",
		MaxPeers:   10,
		ListenAddr: "0.0.0.0:30303",
		PrivateKey: newkey(),
	}
	server := &Server{
		Config:       config,
		newPeerHook:  pf,
		newTransport: func(fd net.Conn) transport { return newTestTransport(id, fd) },
	}
	if err := server.Start(); err != nil {
		fmt.Printf("Could not start server: %v", err)
	}
	return server
}
func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
func randomID() (id discover.NodeID) {
	for i := range id {
		id[i] = byte(rand.Intn(255))
	}
	return id
}
func test() {
	listener, err := net.Listen("tcp", "0.0.0.0:30303")
	if err != nil {
		fmt.Printf("could not setup listener: %v", err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("accept error:", err)
			return
		}
		accepted <- conn
	}()

	// start the server
	connected := make(chan *Peer)
	remid := randomID()
	srv := startServer(remid, func(p *Peer) { connected <- p })
	defer close(connected)
	defer srv.Stop()

	// tell the server to connect
	tcpAddr := listener.Addr().(*net.TCPAddr)
	srv.AddPeer(&discover.Node{ID: remid, IP: tcpAddr.IP, TCP: uint16(tcpAddr.Port)})

	select {
	case conn := <-accepted:
		defer conn.Close()

		select {
		case peer := <-connected:
			if peer.ID() != remid {
				fmt.Printf("peer has wrong id")
			}
			if peer.Name() != "test" {
				fmt.Printf("peer has wrong name")
			}
			if peer.RemoteAddr().String() != conn.LocalAddr().String() {
				fmt.Printf("peer started with wrong conn: got %v, want %v",
					peer.RemoteAddr(), conn.LocalAddr())
			}
			peers := srv.Peers()
			if !reflect.DeepEqual(peers, []*Peer{peer}) {
				fmt.Printf("Peers mismatch: got %v, want %v", peers, []*Peer{peer})
			}
		case <-time.After(1 * time.Second):
			fmt.Printf("server did not launch peer within one second")
		}

	case <-time.After(1 * time.Second):
		fmt.Printf("server did not connect within one second")
	}

}
func main() {
	test()
}
