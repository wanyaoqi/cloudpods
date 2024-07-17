package main

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/hostutils/hardware"
)

func main() {
	topo, err := hardware.GetTopology()
	if err != nil {
		log.Fatalln(err)
	}
	cpuInfo, err := hardware.GetCPU()
	if err != nil {
		log.Fatalln(err)
	}
	log.Infof("topo %v cpuinfo %v", topo.String(), cpuInfo.String())
}
