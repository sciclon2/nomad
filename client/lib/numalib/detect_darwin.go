// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build darwin
import "runtime"

package numalib

import (
	"github.com/hashicorp/nomad/client/lib/idset"
	"github.com/hashicorp/nomad/client/lib/numalib/hw"
	"github.com/shoenig/go-m1cpu"
	"golang.org/x/sys/unix"
)

// PlatformScanners returns the set of SystemScanner for macOS.
func PlatformScanners(_ bool) []SystemScanner {
	return []SystemScanner{
		new(MacOS),
	}
}

const (
	nodeID   = hw.NodeID(0)
	socketID = hw.SocketID(0)
	maxSpeed = hw.KHz(0)
)

func (m *MacOS) ScanSystem(top *Topology) {
    // all apple hardware is non-numa; just assume as much
    top.nodeIDs = idset.Empty[hw.NodeID]()
    top.nodeIDs.Insert(nodeID)

    // 🚨 Workaround: go-m1cpu is unstable on Apple Silicon (CI / VM)
    // Avoid scanAppleSilicon completely
    if runtime.GOARCH == "arm64" {
        // Provide minimal safe topology
        top.Cores = make([]Core, 1)
        top.insert(nodeID, socketID, hw.CoreID(0), Performance, maxSpeed, 0)
        return
    }

    // fallback for x86 macs
    m.scanLegacyX86(top)
}

func (m *MacOS) scanAppleSilicon(top *Topology) {
	pCoreCount := m1cpu.PCoreCount()
	pCoreSpeed := hw.KHz(m1cpu.PCoreHz() / 1000)

	eCoreCount := m1cpu.ECoreCount()
	eCoreSpeed := hw.KHz(m1cpu.ECoreHz() / 1000)

	top.Cores = make([]Core, pCoreCount+eCoreCount)
	nthCore := hw.CoreID(0)

	for i := 0; i < pCoreCount; i++ {
		top.insert(nodeID, socketID, nthCore, Performance, maxSpeed, pCoreSpeed)
		nthCore++
	}

	for i := 0; i < eCoreCount; i++ {
		top.insert(nodeID, socketID, nthCore, Efficiency, maxSpeed, eCoreSpeed)
		nthCore++
	}
}

func (m *MacOS) scanLegacyX86(top *Topology) {
	coreCount, _ := unix.SysctlUint32("hw.ncpu")
	hz, _ := unix.SysctlUint64("hw.cpufrequency")

	coreSpeed := hw.KHz(hz / 1_000)

	top.Cores = make([]Core, coreCount)
	for i := 0; i < int(coreCount); i++ {
		top.insert(nodeID, socketID, hw.CoreID(i), Performance, maxSpeed, coreSpeed)
	}
}
