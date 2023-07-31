package isolated_device

type sSRIOVGpuDevice struct {
	*sBaseDevice

	pfName string
	virtfn int
}

func getSRIOVGpus() ([]*sSRIOVGpuDevice, error) {

}
