package monome

import (
	"fmt"

	"github.com/karalabe/gousb/usb"
)

// Devices returns all monome devices that could be found.
func Devices(options ...Option) ([]Device, error) {
	return find("", options...)
}

const (
	VENDOR_ID  = "0403"
	PRODUCT_ID = "6001"
)

// USBDevices returns all USB devices for the given vendor and product id.
// If they are empty strins, the defaults are used, which is VENDOR_ID and PRODUCT_ID
func USBDevices(vendor_id, product_id string) ([]*usb.Device, error) {
	return findUSBDevices(vendor_id, product_id)
}

func findUSBDevices(vendor_id, product_id string) ([]*usb.Device, error) {
	if vendor_id == "" {
		vendor_id = VENDOR_ID
	}
	if product_id == "" {
		product_id = PRODUCT_ID
	}
	ctx, err := usb.NewContext()

	if err != nil {
		return nil, USBContextError(err.Error())
	}

	defer ctx.Close()

	return ctx.ListDevices(func(desc *usb.Descriptor) bool {
		return usb.Class(desc.Class) == 0 && desc.Vendor.String() == vendor_id && desc.Product.String() == product_id
	})
}

func find(which string, options ...Option) ([]Device, error) {
	ctx, err := usb.NewContext()

	if err != nil {
		return nil, USBContextError(err.Error())
	}

	//	ctx.Debug(4)

	devs, err2 := ctx.ListDevices(func(desc *usb.Descriptor) bool {
		return usb.Class(desc.Class) == 0 && desc.Vendor.String() == VENDOR_ID && desc.Product.String() == PRODUCT_ID
	})

	if err2 != nil {
		return nil, USBContextError(err2.Error())
	}

	var ms []Device

	for _, dev := range devs {
		m, errM := New(dev, options...)
		if errM != nil {
			return ms, errM
		}

		//if which == "" || m.String() == which {
		fmt.Printf("found: %s (%d buttons)\n", m, m.NumButtons())
		m.Flash()
		ms = append(ms, m)
		//}
		/*
			else {
				//	dev.Close()
			}
		*/
	}

	return ms, nil
}
