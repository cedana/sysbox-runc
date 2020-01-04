//
// Copyright: (C) 2019 Nestybox Inc.  All rights reserved.
//

// Exposes functions for sysbox-runc to interact with sysbox-fs

package sysbox

import (
	"fmt"
	"time"

	"github.com/nestybox/sysbox-ipc/sysboxFsGrpc"
)

// FsRegInfo contains info about a sys container registered with sysbox-fs
type FsRegInfo struct {
	Rootfs        string
	Pid           int
	Uid           int
	Gid           int
	IdSize        int
	ProcRoPaths   []string
	ProcMaskPaths []string
}

type Fs struct {
	Active bool
	Id     string // container-id
	Reg    bool   // indicates if sys container was registered with sysbox-fs
}

func NewFs(id string, enable bool) *Fs {
	return &Fs{
		Active: enable,
		Id:     id,
	}
}

func (fs *Fs) Enabled() bool {
	return fs.Active
}

// Registers container info with with sysbox-fs
func (fs *Fs) Register(info *FsRegInfo) error {
	if fs.Reg {
		return fmt.Errorf("container %v already registered", fs.Id)
	}
	data := &sysboxFsGrpc.ContainerData{
		Id:            fs.Id,
		InitPid:       int32(info.Pid),
		Rootfs:        info.Rootfs,
		UidFirst:      int32(info.Uid),
		UidSize:       int32(info.IdSize),
		GidFirst:      int32(info.Gid),
		GidSize:       int32(info.IdSize),
		ProcRoPaths:   info.ProcRoPaths,
		ProcMaskPaths: info.ProcMaskPaths,
	}
	if err := sysboxFsGrpc.SendContainerRegistration(data); err != nil {
		return fmt.Errorf("failed to register with sysbox-fs: %v", err)
	}
	fs.Reg = true
	return nil
}

// Sends container creation time to sysbox-fs
func (fs *Fs) SendCreationTime(t time.Time) error {
	if !fs.Reg {
		return fmt.Errorf("must register container %v before", fs.Id)
	}
	data := &sysboxFsGrpc.ContainerData{
		Id:    fs.Id,
		Ctime: t,
	}
	if err := sysboxFsGrpc.SendContainerUpdate(data); err != nil {
		return fmt.Errorf("failed to send creation time to sysbox-fs: %v", err)
	}
	return nil
}

// Sends the seccomp-notification fd to sysbox-fs (tracer) to setup syscall
// trapping and waits for its response (ack).
func (fs *Fs) SendSeccompFd(id string, seccompFd int32) error {
	// TODO: implement this function
	return nil
}

// Unregisters the container with sysbox-fs
func (fs *Fs) Unregister() error {
	if fs.Reg {
		data := &sysboxFsGrpc.ContainerData{
			Id: fs.Id,
		}
		if err := sysboxFsGrpc.SendContainerUnregistration(data); err != nil {
			return fmt.Errorf("failed to unregister with sysbox-fs: %v", err)
		}
		fs.Reg = false
	}
	return nil
}
