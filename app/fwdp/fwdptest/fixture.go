package fwdptest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/strategy/strategy_elf"
)

const STEP_DELAY = 50 * time.Millisecond
const nFwds = 2

type Fixture struct {
	require *require.Assertions

	DataPlane *fwdp.DataPlane
	Ndt       *ndt.Ndt
	Fib       *fib.Fib
}

func NewFixture(t *testing.T) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.require = require.New(t)

	faceCfg := createface.GetDefaultConfig()
	faceCfg.EnableEth = false
	faceCfg.EnableSock = false
	faceCfg.EnableMock = true
	faceCfg.Apply()

	var dpCfg fwdp.Config

	dpCfg.Crypto.InputCapacity = 64
	dpCfg.Crypto.OpPoolCapacity = 1023

	dpCfg.Ndt.PrefixLen = 2
	dpCfg.Ndt.IndexBits = 16
	dpCfg.Ndt.SampleFreq = 8

	dpCfg.Fib.MaxEntries = 65535
	dpCfg.Fib.NBuckets = 256
	dpCfg.Fib.StartDepth = 8

	dpCfg.Pcct.MaxEntries = 65535
	dpCfg.Pcct.CsCapMd = 16384
	dpCfg.Pcct.CsCapMi = 16384

	dpCfg.LatencySampleFreq = 0

	theDp, e := fwdp.New(dpCfg)
	fixture.require.NoError(e)
	fixture.DataPlane = theDp
	fixture.Ndt = theDp.GetNdt()
	fixture.Fib = theDp.GetFib()

	e = theDp.Launch()
	fixture.require.NoError(e)

	return fixture
}

func (fixture *Fixture) Close() error {
	fixture.DataPlane.Close()
	strategycode.DestroyAll()
	return nil
}

func (fixture *Fixture) CreateFace() *mockface.MockFace {
	face, e := createface.Create(mockface.NewLocator())
	fixture.require.NoError(e)
	return face.(*mockface.MockFace)
}

func (fixture *Fixture) SetFibEntry(name string, strategy string, nexthops ...iface.FaceId) {
	var entry fib.Entry
	e := entry.SetName(ndn.MustParseName(name))
	fixture.require.NoError(e)

	e = entry.SetNexthops(nexthops)
	fixture.require.NoError(e)

	entry.SetStrategy(fixture.makeStrategy(strategy))

	_, e = fixture.Fib.Insert(&entry)
	fixture.require.NoError(e)
}

func (fixture *Fixture) ReadFibCounters(name string) fib.EntryCounters {
	return fixture.Fib.ReadEntryCounters(ndn.MustParseName(name))
}

func (fixture *Fixture) makeStrategy(shortname string) strategycode.StrategyCode {
	if sc := strategycode.Find(shortname); sc != nil {
		return sc
	}

	elf, e := strategy_elf.Load(shortname)
	fixture.require.NoError(e)

	sc, e := strategycode.Load(shortname, elf)
	fixture.require.NoError(e)

	return sc
}

// Read a counter from all FwFwds and compute the sum.
func (fixture *Fixture) SumCounter(getCounter func(dp *fwdp.DataPlane, i int) uint64) (n uint64) {
	for i := 0; i < nFwds; i++ {
		n += getCounter(fixture.DataPlane, i)
	}
	return n
}
