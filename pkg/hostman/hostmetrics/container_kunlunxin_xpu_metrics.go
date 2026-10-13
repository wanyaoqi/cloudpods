package hostmetrics

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

/*
==============XPUSMI LOG==============

Timestamp                                 : Sat Oct 10 07:46:41 2026
Driver Version                            : 5.19.0.0
XPU-RT Version                            : 10.2

Attached XPUs                             : 8
XPU 00000000:03:00.0
    Product Name                          : P800 OAM
    Product Brand                         : KUNLUNXIN
    Product Architecture                  : KL3
    Serial Number                         : 02K15K6252V00115
    XPU UUID                              : GPU-a0ac042a-5a14-5481-bf7e-dd8e523792cd
    Minor Number                          : 0
    OAM Id                                : 3
    XPU Part Number                       : B00100300110312
    Firmware Version
        PBL Version                       : 1.0
        PCIE Version                      : 2.14
        SBL Version                       : 1.48
        ALL Version                       : 1.0.2.14.1.48
        CPLD Version                      : 5.4
    PCI
        Bus                               : 0x03
        Device                            : 0x00
        Function                          : 0x0
        Domain                            : 0x0000
        Device Id                         : 0x36881D22
        Bus Id                            : 00000000:03:00.0
        Sub System Id                     : 0x00010001
        XPU Link Info
            PCIe Generation
                Max                       : 4
                Current                   : 4
            Link Width
                Max                       : 16x
                Current                   : 16x
    Memory Usage
        Total                             : 98304 MiB
        Reserved                          : 0 MiB
        Used                              : 14 MiB
        Free                              : 98290 MiB
   L3 Usage
       Total                             : 96 MiB
       Reserved                          : 0 MiB
       Used                              : 0 MiB
       Free                              : 96 MiB
   Utilization
       Xpu                               : 0 %
   Ecc Mode
       Current                           : Enabled
       Pending                           : Enabled
   ECC Errors
       Volatile
           DRAM Correctable              : 0
           DRAM Uncorrectable            : 0
       Aggregate
           DRAM Correctable              : 0
           DRAM Uncorrectable            : 0
   Temperature
       XPU Current Temp                  : 36 C
   Power Readings
       Enforced Power Limit              : 400.00 W
       Power Draw                        : 87.00 W
   Clocks
       Cluster                           : 1450 MHz
       CDNN                              : 1450 MHz
   Processes
       XPU instance ID                   : N/A
       Compute instance ID               : N/A
       Process ID                        : 2501352
           Type                          : C
           Name                          : ./test_cuda
           Used XPU Memory               : 14 MiB
       XPU instance ID                   : N/A
       Compute instance ID               : N/A
       Process ID                        : 2501354
           Type                          : C
           Name                          : ./test_cuda
           Used XPU Memory               : 12 MiB

XPU 00000000:05:00.0
    Product Name                          : P800 OAM
    Product Brand                         : KUNLUNXIN
    Product Architecture                  : KL3
    Serial Number                         : 02K15K6253V001FY
    XPU UUID                              : GPU-1490b1c8-8410-557a-874a-45d4bf2e1456
    Minor Number                          : 1
    OAM Id                                : 2
    XPU Part Number                       : B00100300110312
    Firmware Version
        PBL Version                       : 1.0
        PCIE Version                      : 2.14
        SBL Version                       : 1.48
        ALL Version                       : 1.0.2.14.1.48
        CPLD Version                      : 5.4
    PCI
        Bus                               : 0x05
        Device                            : 0x00
        Function                          : 0x0
        Domain                            : 0x0000
        Device Id                         : 0x36881D22
        Bus Id                            : 00000000:05:00.0
        Sub System Id                     : 0x00010001
*/

type KunlunxinXpuProcessMetrics struct {
	Pid     string // Process ID
	PciAddr string
	MemUtil float64
	Mem     float64
	Idx     int
}

func GetKunlunxinXpuProcessMetrics() ([]KunlunxinXpuProcessMetrics, error) {
	cmd := "xpu-smi -q"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}

	return parseKunlinxinXpuProcessMetrics(bytes.NewReader(output)), nil
}

func util(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return used / total * 100
}

// parseNum 从 "98304 MiB" / "14 MiB" / "1.48" / "16x" / "N/A" 中取出首个数。
func parseNum(s string) float64 {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0 // N/A 等
	}
	return v
}

// devCtx 设备上下文：只用于给进程补齐 PciAddr/Idx/MemUtil，不对外暴露。
type devCtx struct {
	pciAddr  string
	idx      int
	memTotal float64
}

// procCtx 进程上下文：Process ID / Name / Used XPU Memory 三行拼出来的中间态。
type procCtx struct {
	pid string
	mem float64
}

// 设备头：形如 "XPU 00000000:03:00.0"
var deviceHeaderRe = regexp.MustCompile(`^XPU\s+([0-9a-fA-F]{4,8}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F])$`)

// key : value，按第一个冒号切分；"Bus Id : 00000000:03:00.0" 也能正确拆开
var kvRe = regexp.MustCompile(`^([^:]+?)\s*:\s*(.*)$`)

func parseKunlinxinXpuProcessMetrics(r io.Reader) []KunlunxinXpuProcessMetrics {
	var procs []KunlunxinXpuProcessMetrics

	var (
		curDev  *devCtx
		curProc *procCtx
		section string
	)

	emitProc := func() {
		if curProc == nil || curDev == nil {
			curProc = nil
			return
		}
		if curProc.pid != "" && curProc.pid != "N/A" {
			procs = append(procs, KunlunxinXpuProcessMetrics{
				Pid:     curProc.pid,
				PciAddr: curDev.pciAddr,
				Idx:     curDev.idx,
				Mem:     curProc.mem,
				MemUtil: util(curProc.mem, curDev.memTotal),
			})
		}
		curProc = nil
	}

	flushDev := func() {
		emitProc()
		curDev = nil
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		trimmed := strings.TrimSpace(strings.TrimRight(sc.Text(), " \t\r\n"))
		if trimmed == "" {
			continue
		}

		if m := deviceHeaderRe.FindStringSubmatch(trimmed); m != nil {
			flushDev()
			curDev = &devCtx{pciAddr: m[1]}
			section = ""
			continue
		}
		if curDev == nil {
			continue
		}

		if !strings.Contains(trimmed, ":") {
			section = trimmed
			continue
		}

		m := kvRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		key, val := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])

		switch key {
		case "Minor Number":
			curDev.idx = int(parseNum(val))
		case "Process ID":
			emitProc() // 上一个进程结束
			curProc = &procCtx{pid: val}
		case "Used XPU Memory":
			if curProc != nil {
				curProc.mem = parseNum(val)
			}
		}

		if section == "Memory Usage" {
			switch key {
			case "Total":
				curDev.memTotal = parseNum(val)
			}
		}
	}
	flushDev()

	return procs
}
