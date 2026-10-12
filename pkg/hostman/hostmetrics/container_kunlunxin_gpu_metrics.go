package hostmetrics

import (
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

type KunlunxinGpuProcessMetrics struct {
	Pid     string // Process ID
	PciAddr string
	MemUtil float64
	Mem     float64
	Enc     float64
	Dec     float64
	SmUtil  float64
	Idx     int
}

func GetKunlunxinGpuProcessMetrics() ([]KunlunxinGpuProcessMetrics, error) {
	cmd := "xpu-smi -q"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}
	return parseKunlinxinGpuProcessMetrics(string(output)), nil
}

func parseKunlinxinGpuProcessMetrics(output string) []KunlunxinGpuProcessMetrics {
	lines := strings.Split(output, "\n")
	var i = 0
	for i < len(lines) {
		if !strings.HasPrefix(lines[i], "XPU ") {

		}
	}
}
