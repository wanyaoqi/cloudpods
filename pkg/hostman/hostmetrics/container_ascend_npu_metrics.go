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

// ContainerAscendNpuMetrics 单个昇腾 NPU 进程指标。
type ContainerAscendNpuMetrics struct {
	Pid     string  // Process id
	PciAddr string  // Bus-Id，如 0000:C1:00.0
	MemUtil float64 // 显存占用率，百分比 0~100（Mem / HBM 总量 * 100）
	Mem     float64 // Process memory(MB)
	Idx     int     // NPU 编号（npu-smi 表格第一列）
}

// ascendDevCtx 设备上下文：只用于给进程补 PciAddr 和 MemUtil，不对外暴露。
type ascendDevCtx struct {
	pciAddr  string
	memTotal float64
}

// ascendRawProc 进程表解析出的中间态，最后统一用设备上下文补全。
type ascendRawProc struct {
	idx int
	pid string
	mem float64
}

var (
	// 设备表每块的第一行第一格："0     910B2" → 取 NPU 编号
	ascendDevIdxRe = regexp.MustCompile(`^(\d+)\s+\S+`)
	// 设备表每块的第二行 Bus-Id 列："0000:C1:00.0"
	ascendBusIdRe = regexp.MustCompile(`^[0-9a-fA-F]{4,8}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.\d+$`)
	// 最后一个 "used / total" 组合，即 HBM-Usage(MB) 列
	ascendPairRe = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*/\s*(\d+(?:\.\d+)?)\s*$`)
)

const (
	ascendDevHdr     = "Bus-Id"          // 设备表表头特征
	ascendProcHdr    = "Process id"      // 进程表表头特征
	ascendNoProcLine = "No running proc" // 空卡提示行
)

// GetContainerAscendNpuMetrics 执行 npu-smi 并解析进程指标。
func GetContainerAscendNpuMetrics() ([]ContainerAscendNpuMetrics, error) {
	cmd := "npu-smi info"
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Execute %s failed: %s", cmd, output)
	}
	return parseContainerAscendNpuMetrics(bytes.NewReader(output)), nil
}

// parseContainerAscendNpuMetrics 解析 npu-smi info 的表格输出。
//
// 输出分两张表：
//  1. 设备表：每个 NPU 两行 —— 第一行给编号（0 910B2），第二行给 Bus-Id 与 HBM 总量；
//  2. 进程表：| NPU  Chip | Process id | Process name | Process memory(MB) | ... |
//
// 进程先按原始形态收集，扫描结束后再用设备表补全 PciAddr / MemUtil，
// 这样即使两张表顺序调整也不会解析错。
func parseContainerAscendNpuMetrics(r io.Reader) []ContainerAscendNpuMetrics {
	devs := make(map[int]*ascendDevCtx)
	var raw []ascendRawProc

	var (
		inProcTable bool
		curIdx      int // 最近一个设备名行的 NPU 编号，供紧随其后的 Chip 行使用
	)

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// 跳过空行、+---+ / +===+ 分隔线以及表格外的说明文字
		if line == "" || !strings.HasPrefix(line, "|") {
			continue
		}

		if strings.Contains(line, ascendProcHdr) {
			inProcTable = true
			continue
		}
		if strings.Contains(line, ascendDevHdr) {
			inProcTable = false
			continue
		}
		if strings.Contains(line, ascendNoProcLine) {
			continue
		}

		cells := splitAscendRow(line)
		if len(cells) == 0 {
			continue
		}

		if inProcTable {
			// | 0   0 | 1011595 | python3 | 2163 | NA |
			if len(cells) < 4 {
				continue
			}
			idx, ok := ascendAtoi(cells[0])
			if !ok {
				continue
			}
			pid := cells[1]
			if pid == "" || pid == "NA" {
				continue
			}
			raw = append(raw, ascendRawProc{idx: idx, pid: pid, mem: parseNum(cells[3])})
			continue
		}

		// 设备表第一行："0     910B2" → 记录编号，并准备好设备上下文
		if m := ascendDevIdxRe.FindStringSubmatch(cells[0]); m != nil {
			curIdx = int(parseNum(m[1]))
			if devs[curIdx] == nil {
				devs[curIdx] = &ascendDevCtx{}
			}
			continue
		}

		// 设备表第二行：Chip / Bus-Id / AICore / Memory-Usage / HBM-Usage
		if len(cells) >= 2 && ascendBusIdRe.MatchString(cells[1]) {
			dev := devs[curIdx]
			if dev == nil { // 没抓到编号行时的兜底
				dev = &ascendDevCtx{}
				devs[curIdx] = dev
			}
			dev.pciAddr = cells[1]
			if m := ascendPairRe.FindStringSubmatch(cells[len(cells)-1]); m != nil {
				dev.memTotal = parseNum(m[2]) // "5505 / 65536" 的分母
			}
		}
	}

	procs := make([]ContainerAscendNpuMetrics, 0, len(raw))
	for _, p := range raw {
		m := ContainerAscendNpuMetrics{Pid: p.pid, Idx: p.idx, Mem: p.mem}
		if dev, ok := devs[p.idx]; ok {
			m.PciAddr = dev.pciAddr
			m.MemUtil = util(p.mem, dev.memTotal)
		}
		procs = append(procs, m)
	}
	return procs
}

// splitAscendRow 按 | 切表格行，去掉首尾空单元格并 TrimSpace。
func splitAscendRow(line string) []string {
	cells := make([]string, 0, 8)
	for _, p := range strings.Split(line, "|") {
		if p = strings.TrimSpace(p); p != "" {
			cells = append(cells, p)
		}
	}
	return cells
}

// ascendAtoi 取单元格里第一个整数，如 "0       0" → 0。
func ascendAtoi(s string) (int, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, false
	}
	return n, true
}

/*
+------------------------------------------------------------------------------------------------------------------+
| npu-smi 26.1.1                              Version: 26.1.1                                                      |
+---------------------------+---------------+----------------------------------------------------------------------+
| NPU   Name                | Health        | Power(W)             Temp(C)                 Hugepages-Usage(page)   |
| Chip                      | Bus-Id        | AICore(%)            Memory-Usage(MB)        HBM-Usage(MB)           |
+===========================+===============+======================================================================+
| 0     910B2               | OK            | 126.4                44                      0    / 0                |
| 0                         | 0000:C1:00.0  | 0                    0    / 0                5505 / 65536            |
+===========================+===============+======================================================================+
| 1     910B2               | OK            | 92.6                 40                      0    / 0                |
| 0                         | 0000:C2:00.0  | 0                    0    / 0                3412 / 65536            |
+===========================+===============+======================================================================+
| 2     910B2               | OK            | 91.1                 39                      0    / 0                |
| 0                         | 0000:81:00.0  | 0                    0    / 0                3411 / 65536            |
+===========================+===============+======================================================================+
| 3     910B2               | OK            | 94.1                 39                      0    / 0                |
| 0                         | 0000:82:00.0  | 0                    0    / 0                3412 / 65536            |
+===========================+===============+======================================================================+
| 4     910B2               | OK            | 100.7                44                      0    / 0                |
| 0                         | 0000:01:00.0  | 0                    0    / 0                3412 / 65536            |
+===========================+===============+======================================================================+
| 5     910B2               | OK            | 94.6                 43                      0    / 0                |
| 0                         | 0000:02:00.0  | 0                    0    / 0                3411 / 65536            |
+===========================+===============+======================================================================+
| 6     910B2               | OK            | 99.7                 44                      0    / 0                |
| 0                         | 0000:41:00.0  | 0                    0    / 0                3412 / 65536            |
+===========================+===============+======================================================================+
| 7     910B2               | OK            | 99.6                 43                      0    / 0                |
| 0                         | 0000:42:00.0  | 0                    0    / 0                3411 / 65536            |
+===========================+===============+======================================================================+
+---------------------------+---------------+----------------------------------------------------------------------+
| NPU     Chip              | Process id    | Process name       | Process memory(MB)    | Process id in container |
+===========================+===============+======================================================================+
| 0       0                 | 1011595       | python3            | 2163                  | NA                      |
| 0       0                 | 1011606       | python3            | 2163                  | NA                      |
+===========================+===============+======================================================================+
| No running processes found in NPU 1                                                                              |
+===========================+===============+======================================================================+
| 2       0                 | 1013240       | python3            | 1139                  | NA                      |
+===========================+===============+======================================================================+
| No running processes found in NPU 3                                                                              |
+===========================+===============+======================================================================+
| No running processes found in NPU 4                                                                              |
+===========================+===============+======================================================================+
| No running processes found in NPU 5                                                                              |
+===========================+===============+======================================================================+
| No running processes found in NPU 6                                                                              |
+===========================+===============+======================================================================+
| No running processes found in NPU 7                                                                              |
+===========================+===============+======================================================================+
*/
