// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package measurements

import "yunion.io/x/onecloud/pkg/apis/monitor"

// procstat metrics come from telegraf [[inputs.procstat]].
// See hostman/system_service/telegraf.go for enabled fields (fieldpass).
var procstat = SMeasurement{
	Context: []SMonitorContext{
		{
			Name:         "procstat",
			DisplayName:  "Process statistics",
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
		{
			Name:        "cpu_usage",
			DisplayName: "CPU usage percent",
			Unit:        monitor.METRIC_UNIT_PERCENT,
		},
		{
			Name:        "memory_rss",
			DisplayName: "Resident memory size",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "read_bytes",
			DisplayName: "Bytes read",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "write_bytes",
			DisplayName: "Bytes written",
			Unit:        monitor.METRIC_UNIT_BYTE,
		},
		{
			Name:        "num_threads",
			DisplayName: "Number of threads",
			Unit:        monitor.METRIC_UNIT_COUNT,
		},
	},
}

