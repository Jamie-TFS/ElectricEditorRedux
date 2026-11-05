//go:build windows

package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procReadProcessMemory  = kernel32.NewProc("ReadProcessMemory")
	procWriteProcessMemory = kernel32.NewProc("WriteProcessMemory")
	procVirtualQueryEx     = kernel32.NewProc("VirtualQueryEx")
	procModule32FirstW     = kernel32.NewProc("Module32FirstW")
	procGetExitCodeProcess = kernel32.NewProc("GetExitCodeProcess")
)

const (
	processVMRead      = 0x0010
	processVMWrite     = 0x0020
	processVMOperation = 0x0008
	processQueryInfo   = 0x0400
	snapProcess        = 0x00000002
	snapModule         = 0x00000008
	snapModule32       = 0x00000010
	memCommit          = 0x1000
	memPrivate         = 0x20000
	pageReadWrite      = 0x04
	stillActive        = 259
)

type memoryBasicInformation struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	PartitionID       uint16
	_                 uint16
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
	_                 uint32
}

type moduleEntry32 struct {
	Size         uint32
	ModuleID     uint32
	ProcessID    uint32
	GlblcntUsage uint32
	ProccntUsage uint32
	ModBaseAddr  uintptr
	ModBaseSize  uint32
	HModule      uintptr
	SzModule     [256]uint16
	SzExePath    [260]uint16
}

// Process is an attached Tekken 8 instance.
type Process struct {
	h          syscall.Handle
	PID        uint32
	Name       string
	ExePath    string
	base, size uint64
}

var ErrNotRunning = errors.New("Tekken 8 is not running")

func utf16z(s []uint16) string {
	for i, c := range s {
		if c == 0 {
			return syscall.UTF16ToString(s[:i])
		}
	}
	return syscall.UTF16ToString(s)
}

// Attach finds Polaris*.exe and opens it for reading and writing.
func Attach() (*Process, error) {
	snap, err := syscall.CreateToolhelp32Snapshot(snapProcess, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(snap)

	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	var pid uint32
	var name string
	for e := syscall.Process32First(snap, &pe); e == nil; e = syscall.Process32Next(snap, &pe) {
		n := utf16z(pe.ExeFile[:])
		ln := strings.ToLower(n)
		if strings.HasPrefix(ln, "polaris") && strings.HasSuffix(ln, ".exe") {
			pid, name = pe.ProcessID, n
			break
		}
	}
	if pid == 0 {
		return nil, ErrNotRunning
	}

	h, err := syscall.OpenProcess(processVMRead|processVMWrite|processVMOperation|processQueryInfo, false, pid)
	if err != nil {
		return nil, fmt.Errorf("can't open Tekken (%v) - try running Electric Editor as administrator", err)
	}
	msnap, err := syscall.CreateToolhelp32Snapshot(snapModule|snapModule32, pid)
	if err != nil {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("module snapshot: %v", err)
	}
	defer syscall.CloseHandle(msnap)
	var me moduleEntry32
	me.Size = uint32(unsafe.Sizeof(me))
	if r, _, e := procModule32FirstW.Call(uintptr(msnap), uintptr(unsafe.Pointer(&me))); r == 0 {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("Module32FirstW: %v", e)
	}
	return &Process{h: h, PID: pid, Name: name, ExePath: utf16z(me.SzExePath[:]), base: uint64(me.ModBaseAddr), size: uint64(me.ModBaseSize)}, nil
}

func (p *Process) Close() { syscall.CloseHandle(p.h) }

func (p *Process) Alive() bool {
	var code uint32
	r, _, _ := procGetExitCodeProcess.Call(uintptr(p.h), uintptr(unsafe.Pointer(&code)))
	return r != 0 && code == stillActive
}

func (p *Process) read(addr uint64, buf []byte) bool {
	if len(buf) == 0 {
		return true
	}
	var n uintptr
	r, _, _ := procReadProcessMemory.Call(uintptr(p.h), uintptr(addr),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)))
	return r != 0 && int(n) == len(buf)
}

func (p *Process) write(addr uint64, buf []byte) error {
	var n uintptr
	r, _, err := procWriteProcessMemory.Call(uintptr(p.h), uintptr(addr),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)))
	if r == 0 || int(n) != len(buf) {
		return fmt.Errorf("write failed: %v", err)
	}
	return nil
}

func (p *Process) u64(addr uint64) uint64 {
	var b [8]byte
	if !p.read(addr, b[:]) {
		return 0
	}
	return binary.LittleEndian.Uint64(b[:])
}

func (p *Process) u32(addr uint64) uint32 {
	var b [4]byte
	if !p.read(addr, b[:]) {
		return 0
	}
	return binary.LittleEndian.Uint32(b[:])
}

func (p *Process) u16(addr uint64) uint16 {
	var b [2]byte
	if !p.read(addr, b[:]) {
		return 0
	}
	return binary.LittleEndian.Uint16(b[:])
}

type region struct{ base, size uint64 }

// heapRegions lists committed private read/write memory (where UE allocates).
func (p *Process) heapRegions() []region {
	var out []region
	var addr uint64
	for addr < 0x7FFFFFFF0000 {
		var mbi memoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(uintptr(p.h), uintptr(addr), uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
		if r == 0 {
			break
		}
		next := uint64(mbi.BaseAddress) + uint64(mbi.RegionSize)
		if mbi.State == memCommit && mbi.Type == memPrivate && mbi.Protect == pageReadWrite {
			out = append(out, region{uint64(mbi.BaseAddress), uint64(mbi.RegionSize)})
		}
		if next <= addr {
			break
		}
		addr = next
	}
	return out
}
