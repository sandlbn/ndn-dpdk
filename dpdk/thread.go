package dpdk

/*
#include "thread.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// An application thread.
type IThread interface {
	SetLCore(lc LCore) // Assign an LCore.
	GetLCore() LCore   // Return assigned LCore.
	IsRunning() bool

	Launch() error // Launch the thread.
	Stop() error   // Stop the thread.
	Close() error  // Release data structures.
}

// Base class of an application thread.
type ThreadBase struct {
	lc LCore
}

func (t *ThreadBase) ResetThreadBase() {
	t.lc = LCORE_INVALID
}

func (t *ThreadBase) SetLCore(lc LCore) {
	if t.IsRunning() {
		panic("cannot change lcore while running")
	}
	t.lc = lc
}

func (t *ThreadBase) GetLCore() LCore {
	return t.lc
}

func (t *ThreadBase) IsRunning() bool {
	return t.lc != LCORE_INVALID && t.lc.GetState() != LCORE_STATE_WAIT
}

func (t *ThreadBase) MustHaveLCore() {
	if t.lc == LCORE_INVALID {
		panic("lcore unassigned")
	}
}

func (t *ThreadBase) GetNumaSocket() NumaSocket {
	t.MustHaveLCore()
	return t.lc.GetNumaSocket()
}

func (t *ThreadBase) LaunchImpl(f func() int) error {
	t.MustHaveLCore()
	if t.IsRunning() {
		return fmt.Errorf("lcore %d is running", t.lc)
	}
	if ok := t.lc.RemoteLaunch(f); !ok {
		return fmt.Errorf("unable to launch on %d", t.lc)
	}
	return nil
}

func (t *ThreadBase) StopImpl(stop IStop) error {
	t.MustHaveLCore()
	if !t.IsRunning() {
		return nil
	}
	stop.BeforeWait()
	exitCode := t.lc.Wait()
	stop.AfterWait()
	if exitCode != 0 {
		return fmt.Errorf("exit code %d", exitCode)
	}
	return nil
}

var ErrCloseRunningThread = errors.New("cannot close a running thread")

// Thread stop helper.
type IStop interface {
	BeforeWait() // What to do before lcore.Wait().
	AfterWait()  // What to do after lcore.Wait().
}

// Stop a thread by waiting for it indefinitely.
type StopWait struct{}

func (stop StopWait) BeforeWait() {}
func (stop StopWait) AfterWait()  {}

// Stop a thread by setting a boolean flag.
type StopFlag struct {
	c *C.ThreadStopFlag
}

func NewStopFlag(c unsafe.Pointer) (stop StopFlag) {
	stop.c = (*C.ThreadStopFlag)(c)
	return stop
}

func InitStopFlag(c unsafe.Pointer) (stop StopFlag) {
	stop = NewStopFlag(c)
	stop.Init()
	return stop
}

func (stop StopFlag) Init() {
	C.ThreadStopFlag_Init(stop.c)
}

func (stop StopFlag) BeforeWait() {
	C.ThreadStopFlag_RequestStop(stop.c)
}

func (stop StopFlag) AfterWait() {
	C.ThreadStopFlag_FinishStop(stop.c)
}

// Stop a thread by sending to a channel.
type StopChan chan bool

func NewStopChan() (stop StopChan) {
	return make(StopChan)
}

func (stop StopChan) Continue() bool {
	select {
	case <-stop:
		return false
	default:
		return true
	}
}

func (stop StopChan) BeforeWait() {
	stop <- true
}

func (stop StopChan) AfterWait() {
}
