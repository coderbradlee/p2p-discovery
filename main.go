package main

import (
	"./logger"
	util "./utils"
	"bufio"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

var cfg *util.Config

func log_init() {
	logger.SetConsole(cfg.Log.Console)
	logger.SetRollingFile(cfg.Log.Dir, cfg.Log.Name, cfg.Log.Num, cfg.Log.Size, logger.KB)
	//ALL，DEBUG，INFO，WARN，ERROR，FATAL，OFF
	logger.SetLevel(logger.ERROR)
	if cfg.Log.Level == "info" {
		logger.SetLevel(logger.INFO)
	} else if cfg.Log.Level == "error" {
		logger.SetLevel(logger.ERROR)
	}
}
func init() {
	cfg = &util.Config{}

	if !util.LoadConfig("seeker.toml", cfg) {
		return
	}
	log_init()
	initialize()
}

func test() {
		pm, _ := newTestProtocolManagerMust( downloader.FullSync, downloader.MaxHashFetch+15, nil, nil)
		peer, _ := newTestPeer("peer", 63, pm, true)
		defer peer.close()
	
		// Create a "random" unknown hash for testing
		var unknown common.Hash
		for i := range unknown {
			unknown[i] = byte(i)
		}
		// Create a batch of tests for various scenarios
		limit := uint64(downloader.MaxHeaderFetch)
		tests := []struct {
			query  *getBlockHeadersData // The query to execute for header retrieval
			expect []common.Hash        // The hashes of the block whose headers are expected
		}{
			
			{
				&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Amount: 3},
				[]common.Hash{
					pm.blockchain.GetBlockByNumber(limit / 2).Hash(),
					pm.blockchain.GetBlockByNumber(limit/2 + 1).Hash(),
					pm.blockchain.GetBlockByNumber(limit/2 + 2).Hash(),
				},
			}, 
			// Multiple headers with skip lists should be retrievable
			{
				&getBlockHeadersData{Origin: hashOrNumber{Number: limit / 2}, Skip: 3, Amount: 3},
				[]common.Hash{
					pm.blockchain.GetBlockByNumber(limit / 2).Hash(),
					pm.blockchain.GetBlockByNumber(limit/2 + 4).Hash(),
					pm.blockchain.GetBlockByNumber(limit/2 + 8).Hash(),
				}
			}
		}
		// Run each of the tests and verify the results against the chain
		for i, tt := range tests {
			// Collect the headers to expect in the response
			headers := []*types.Header{}
			for _, hash := range tt.expect {
				headers = append(headers, pm.blockchain.GetBlockByHash(hash).Header())
			}
			// Send the hash request and verify the response
			p2p.Send(peer.app, 0x03, tt.query)
			if err := p2p.ExpectMsg(peer.app, 0x04, headers); err != nil {
				t.Errorf("test %d: headers mismatch: %v", i, err)
			}
			// If the test used number origins, repeat with hashes as the too
			if tt.query.Origin.Hash == (common.Hash{}) {
				if origin := pm.blockchain.GetBlockByNumber(tt.query.Origin.Number); origin != nil {
					tt.query.Origin.Hash, tt.query.Origin.Number = origin.Hash(), 0
	
					p2p.Send(peer.app, 0x03, tt.query)
					if err := p2p.ExpectMsg(peer.app, 0x04, headers); err != nil {
						t.Errorf("test %d: headers mismatch: %v", i, err)
					}
				}
			}
		}
}
func main() {
	test()
}
