package appinit

import (
	"fmt"
	"log"
	"math"
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

type MempoolConfig struct {
	Capacity     int
	CacheSize    int
	PrivSize     uint16
	DataRoomSize uint16
}

var mempoolCfgs = make(map[string]*MempoolConfig)
var mempools = make(map[string]dpdk.PktmbufPool)

// Register template for mempool creation.
func RegisterMempool(key string, cfg MempoolConfig) {
	if _, ok := mempoolCfgs[key]; ok {
		log.Panicf("RegisterPktmbufPool(%s) duplicate", key)
	}

	if strings.ContainsRune(key, '#') {
		log.Panicf("RegisterPktmbufPool(%s) key cannot contain '#'", key)
	}

	mempoolCfgs[key] = &cfg
}

// Modify mempool capacity in template.
func ConfigureMempool(key string, capacity int, cacheSize int) {
	cfg, ok := mempoolCfgs[key]
	if !ok {
		log.Panicf("ConfigurePktmbufPool(%s) unregistered", key)
	}

	cfg.Capacity = capacity
	cfg.CacheSize = cacheSize
}

// Get or create a mempool on specified NumaSocket.
func MakePktmbufPool(key string, socket dpdk.NumaSocket) dpdk.PktmbufPool {
	cfg, ok := mempoolCfgs[key]
	if !ok {
		log.Panicf("MakePktmbufPool(%s) unregistered", key)
	}

	if cfg.Capacity <= 0 {
		Exitf(EXIT_BAD_CONFIG, "MakePktmbufPool(%s) bad config: capacity must be positive")
	}
	if ((cfg.Capacity + 1) & cfg.Capacity) != 0 {
		log.Printf("MakePktmbufPool(%s) nonoptimal config: capacity is not 2^q-1")
	}
	maxCacheSize := int(math.Min(float64(MEMPOOL_MAX_CACHE_SIZE), float64(cfg.Capacity)/1.5))
	if cfg.CacheSize < 0 || cfg.CacheSize > maxCacheSize {
		Exitf(EXIT_BAD_CONFIG, "MakePktmbufPool(%s) bad config: cache size must be between 0 and %d",
			maxCacheSize)
	}
	if cfg.CacheSize > 0 && cfg.Capacity%cfg.CacheSize != 0 {
		log.Printf("MakePktmbufPool(%s) nonoptimal config: capacity is not a multiply of cacheSize")
	}

	name := fmt.Sprintf("%s#%d", key, socket)
	if mp, ok := mempools[name]; ok {
		return mp
	}

	mp, e := dpdk.NewPktmbufPool(name, cfg.Capacity, cfg.CacheSize,
		cfg.PrivSize, cfg.DataRoomSize, socket)
	if e != nil {
		Exitf(EXIT_MEMPOOL_INIT_ERROR, "MakePktmbufPool(%s,%d): %v", key, socket, e)
	}
	mempools[name] = mp
	return mp
}

const (
	MP_IND   = "__IND"   // mempool for indirect mbufs
	MP_ETHRX = "__ETHRX" // mempool for incoming Ethernet frames
	MP_ETHTX = "__ETHTX" // mempool for outgoing Ethernet and NDNLP headers
)

func init() {
	RegisterMempool(MP_IND,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataRoomSize: 0,
		})
	RegisterMempool(MP_ETHRX,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     ndn.SizeofPacketPriv(),
			DataRoomSize: 9014, // MTU+sizeof(ether_hdr)
		})
	RegisterMempool(MP_ETHTX,
		MempoolConfig{
			Capacity:     65535,
			CacheSize:    255,
			PrivSize:     0,
			DataRoomSize: ethface.SizeofHeaderMempoolDataRoom(),
		})
}
