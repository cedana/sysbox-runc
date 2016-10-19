// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/docker/go-units"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

func u64Ptr(i uint64) *uint64 { return &i }
func u16Ptr(i uint16) *uint16 { return &i }

var updateCommand = cli.Command{
	Name:      "update",
	Usage:     "update container resource constraints",
	ArgsUsage: `<container-id>`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "resources, r",
			Value: "",
			Usage: `path to the file containing the resources to update or '-' to read from the standard input

The accepted format is as follow (unchanged values can be omitted):

{
  "memory": {
    "limit": 0,
    "reservation": 0,
    "swap": 0,
    "kernel": 0,
    "kernelTCP": 0
  },
  "cpu": {
    "shares": 0,
    "quota": 0,
    "period": 0,
    "realtimeRuntime": 0,
    "realtimePeriod": 0,
    "cpus": "",
    "mems": ""
  },
  "blockIO": {
    "blkioWeight": 0
  }
}

Note: if data is to be read from a file or the standard input, all
other options are ignored.
`,
		},

		cli.IntFlag{
			Name:  "blkio-weight",
			Usage: "Specifies per cgroup weight, range is from 10 to 1000",
		},
		cli.StringFlag{
			Name:  "cpu-period",
			Usage: "CPU CFS period to be used for hardcapping (in usecs). 0 to use system default",
		},
		cli.StringFlag{
			Name:  "cpu-quota",
			Usage: "CPU CFS hardcap limit (in usecs). Allowed cpu time in a given period",
		},
		cli.StringFlag{
			Name:  "cpu-share",
			Usage: "CPU shares (relative weight vs. other containers)",
		},
		cli.StringFlag{
			Name:  "cpu-rt-period",
			Usage: "CPU realtime period to be used for hardcapping (in usecs). 0 to use system default",
		},
		cli.StringFlag{
			Name:  "cpu-rt-runtime",
			Usage: "CPU realtime hardcap limit (in usecs). Allowed cpu time in a given period",
		},
		cli.StringFlag{
			Name:  "cpuset-cpus",
			Usage: "CPU(s) to use",
		},
		cli.StringFlag{
			Name:  "cpuset-mems",
			Usage: "Memory node(s) to use",
		},
		cli.StringFlag{
			Name:  "kernel-memory",
			Usage: "Kernel memory limit (in bytes)",
		},
		cli.StringFlag{
			Name:  "kernel-memory-tcp",
			Usage: "Kernel memory limit (in bytes) for tcp buffer",
		},
		cli.StringFlag{
			Name:  "memory",
			Usage: "Memory limit (in bytes)",
		},
		cli.StringFlag{
			Name:  "memory-reservation",
			Usage: "Memory reservation or soft_limit (in bytes)",
		},
		cli.StringFlag{
			Name:  "memory-swap",
			Usage: "Total memory usage (memory + swap); set '-1' to enable unlimited swap",
		},
	},
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}
		container, err := getContainer(context)
		if err != nil {
			return err
		}

		r := specs.LinuxResources{
			Memory: &specs.LinuxMemory{
				Limit:       u64Ptr(0),
				Reservation: u64Ptr(0),
				Swap:        u64Ptr(0),
				Kernel:      u64Ptr(0),
				KernelTCP:   u64Ptr(0),
			},
			CPU: &specs.LinuxCPU{
				Shares:          u64Ptr(0),
				Quota:           u64Ptr(0),
				Period:          u64Ptr(0),
				RealtimeRuntime: u64Ptr(0),
				RealtimePeriod:  u64Ptr(0),
				Cpus:            sPtr(""),
				Mems:            sPtr(""),
			},
			BlockIO: &specs.LinuxBlockIO{
				Weight: u16Ptr(0),
			},
		}

		config := container.Config()

		if in := context.String("resources"); in != "" {
			var (
				f   *os.File
				err error
			)
			switch in {
			case "-":
				f = os.Stdin
			default:
				f, err = os.Open(in)
				if err != nil {
					return err
				}
			}
			err = json.NewDecoder(f).Decode(&r)
			if err != nil {
				return err
			}
		} else {
			if val := context.Int("blkio-weight"); val != 0 {
				r.BlockIO.Weight = u16Ptr(uint16(val))
			}
			if val := context.String("cpuset-cpus"); val != "" {
				r.CPU.Cpus = &val
			}
			if val := context.String("cpuset-mems"); val != "" {
				r.CPU.Mems = &val
			}

			for _, pair := range []struct {
				opt  string
				dest *uint64
			}{

				{"cpu-period", r.CPU.Period},
				{"cpu-quota", r.CPU.Quota},
				{"cpu-rt-period", r.CPU.RealtimePeriod},
				{"cpu-rt-runtime", r.CPU.RealtimeRuntime},
				{"cpu-share", r.CPU.Shares},
			} {
				if val := context.String(pair.opt); val != "" {
					var err error
					*pair.dest, err = strconv.ParseUint(val, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %s", pair.opt, err)
					}
				}
			}
			for _, pair := range []struct {
				opt  string
				dest *uint64
			}{
				{"memory", r.Memory.Limit},
				{"memory-swap", r.Memory.Swap},
				{"kernel-memory", r.Memory.Kernel},
				{"kernel-memory-tcp", r.Memory.KernelTCP},
				{"memory-reservation", r.Memory.Reservation},
			} {
				if val := context.String(pair.opt); val != "" {
					var v int64

					if val != "-1" {
						v, err = units.RAMInBytes(val)
						if err != nil {
							return fmt.Errorf("invalid value for %s: %s", pair.opt, err)
						}
					} else {
						v = -1
					}
					*pair.dest = uint64(v)
				}
			}
		}

		// Update the value
		config.Cgroups.Resources.BlkioWeight = *r.BlockIO.Weight
		config.Cgroups.Resources.CpuPeriod = int64(*r.CPU.Period)
		config.Cgroups.Resources.CpuQuota = int64(*r.CPU.Quota)
		config.Cgroups.Resources.CpuShares = int64(*r.CPU.Shares)
		config.Cgroups.Resources.CpuRtPeriod = int64(*r.CPU.RealtimePeriod)
		config.Cgroups.Resources.CpuRtRuntime = int64(*r.CPU.RealtimeRuntime)
		config.Cgroups.Resources.CpusetCpus = *r.CPU.Cpus
		config.Cgroups.Resources.CpusetMems = *r.CPU.Mems
		config.Cgroups.Resources.KernelMemory = int64(*r.Memory.Kernel)
		config.Cgroups.Resources.KernelMemoryTCP = int64(*r.Memory.KernelTCP)
		config.Cgroups.Resources.Memory = int64(*r.Memory.Limit)
		config.Cgroups.Resources.MemoryReservation = int64(*r.Memory.Reservation)
		config.Cgroups.Resources.MemorySwap = int64(*r.Memory.Swap)

		if err := container.Set(config); err != nil {
			return err
		}
		return nil
	},
}
