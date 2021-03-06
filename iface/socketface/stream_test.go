package socketface_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/socketface"
)

func TestStream(t *testing.T) {
	_, require := makeAR(t)

	fd, e := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	require.NoError(e)

	faceA, e := socketface.New(makeConnFromFd(fd[0]), socketfaceCfg)
	require.NoError(e)
	defer faceA.Close()
	faceB, e := socketface.New(makeConnFromFd(fd[1]), socketfaceCfg)
	require.NoError(e)
	defer faceB.Close()

	fixture := ifacetestfixture.New(t, faceA, faceB)
	fixture.RunTest()
	fixture.CheckCounters()
}

func checkStreamRedialing(t *testing.T, listener net.Listener, face *socketface.SocketFace) {
	assert, require := makeAR(t)

	var hasDownEvt, hasUpEvt bool
	defer iface.OnFaceDown(func(id iface.FaceId) {
		if id == face.GetFaceId() {
			hasDownEvt = true
		}
	}).Close()
	defer iface.OnFaceUp(func(id iface.FaceId) {
		if id == face.GetFaceId() {
			hasUpEvt = true
		}
	}).Close()

	accepted, e := listener.Accept()
	require.NoError(e)
	time.Sleep(100 * time.Millisecond)
	accepted.Close() // close initial connection

	accepted, e = listener.Accept() // face should redial
	require.NoError(e)
	time.Sleep(100 * time.Millisecond)
	accepted.Close()

	assert.True(hasDownEvt)
	assert.True(hasUpEvt)

	cnt := face.ReadExCounters().(socketface.ExCounters)
	assert.InDelta(1.5, float64(cnt.NRedials), 0.6) // redial counter should be 1 or 2
}

func TestTcp(t *testing.T) {
	assert, require := makeAR(t)

	addr, e := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	require.NoError(e)
	listener, e := net.ListenTCP("tcp", addr)
	require.NoError(e)
	defer listener.Close()
	*addr = *listener.Addr().(*net.TCPAddr)

	loc := iface.MustParseLocator(fmt.Sprintf(`{ "Scheme": "tcp", "Remote": "127.0.0.1:%d" }`, addr.Port)).(socketface.Locator)
	face, e := socketface.Create(loc, socketfaceCfg)
	require.NoError(e)
	defer face.Close()

	assert.Equal(iface.FaceKind_Socket, face.GetFaceId().GetKind())
	loc = face.GetLocator().(socketface.Locator)
	assert.Equal("tcp", loc.Scheme)
	assert.Equal(face.GetConn().LocalAddr().String(), loc.Local)
	assert.Equal(fmt.Sprintf("127.0.0.1:%d", addr.Port), loc.Remote)
	ifacetestfixture.CheckLocatorMarshal(t, loc)

	checkStreamRedialing(t, listener, face)
}

func TestUnix(t *testing.T) {
	assert, require := makeAR(t)

	tmpdir, e := ioutil.TempDir("", "socketface-test")
	require.NoError(e)
	defer os.RemoveAll(tmpdir)
	addr := path.Join(tmpdir, "unix.sock")
	listener, e := net.Listen("unix", addr)
	require.NoError(e)
	defer listener.Close()

	loc := iface.MustParseLocator(fmt.Sprintf(`{ "Scheme": "unix", "Remote": "%s" }`, addr)).(socketface.Locator)
	face, e := socketface.Create(loc, socketfaceCfg)
	require.NoError(e)
	defer face.Close()

	assert.Equal(iface.FaceKind_Socket, face.GetFaceId().GetKind())
	loc = face.GetLocator().(socketface.Locator)
	assert.Equal("unix", loc.Scheme)
	assert.Equal(addr, loc.Remote)
	ifacetestfixture.CheckLocatorMarshal(t, loc)

	checkStreamRedialing(t, listener, face)
}
