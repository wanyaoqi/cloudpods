package guestman

/*
 -device virtio-serial,id=virtio-serial0,addr=0x3,bus=pci.0 \
 -device VGA,id=video0,addr=0x2,bus=pci.0 \
 -device virtio-scsi-pci,id=scsi,addr=0x4,bus=pci.0 \
 -netdev type=tap,id=vnet22-1009,ifname=vnet22-1009,vhost=on,vhostforce=off,script=/opt/cloud/workspace/servers/a3c11dc4-061a-4fe5-8d24-38c62633682b/if-up-brvpc-vnet22-1009.sh,downscript=/opt/cloud/workspace/servers/a3c11dc4-061a-4fe5-8d24-38c62633682b/if-down-brvpc-vnet22-1009.sh \
 -device virtio-net-pci,id=netdev-vnet22-1009,netdev=vnet22-1009,mac=00:24:15:86:18:96$(nic_speed 1000)$(nic_mtu "brvpc"),addr=0x5,bus=pci.0 \
 -device qemu-xhci,id=usb,addr=0x6,bus=pci.0 \
 -object rng-random,id=rng0,filename=/dev/urandom -device virtio-rng-pci,rng=rng0,max-bytes=1024,period=1000,addr=0x7,bus=pci.0
*/

// https://libvirt.org/pci-addresses.html

const (
	DEVICE_TYPE_NETDEV = 0
	DEVICE_TYPE_DISK
	DEVICE_TYPE_USB
	DEVICE_TYPE_GPU
)

// %04x:%02x:%02x.%x, domain, bus, device, function
type PCIDeviceAddress struct {
	Controller string // pcie.x / pci.x / scsi.x / ide.x

	Domain   int
	Bus      int
	Device   int
	Function int

	MultiFunction bool
}

func (s *SKVMGuestInstance) GetPciAddress() {

}
