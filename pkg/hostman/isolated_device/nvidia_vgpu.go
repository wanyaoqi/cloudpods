package isolated_device

type sNVIDIAVgpuDevice struct {
	// base device is pyhsical GPU device
	*sBaseDevice
}

func NewNvidiaVgpuDevice(dev *PCIDevice, devType string) *sNVIDIAVgpuDevice {
	return &sNVIDIAVgpuDevice{
		sBaseDevice: newBaseDevice(dev, devType),
	}
}

func getNvidiaVGpus(gpuPF string) ([]*sNVIDIAVgpuDevice, error) {
	mdevDeviceDir := fmt.Sprintf("/sys/class/mdev_bus/0000:%s", gpuPF)
	if !fileutils2.Exists(mdevDeviceDir) {
		return nil, errors.Errorf("unknown device %s", gpuPF)
	}
	// regutils.MatchUUID(self.HostId)
	files, err := ioutil.ReadDir(mdevDeviceDir)
	if err != nil {
		return nil, errors.Wrap(err, "read mdev device path")
	}
	nvidiaVgpus := make([]*sNVIDIAVgpuDevice, 0)
	for i := range files {
		if regutils.MatchUUID(files[i].Name()) {
			// mdev
		}
	}
}
