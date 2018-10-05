package monome

import (
	"fmt"

	"github.com/karalabe/gousb/usb"
)

// Devices returns all monome devices that could be found.
func Devices(options ...Option) ([]Device, error) {
	return find("", options...)
}

func find(which string, options ...Option) ([]Device, error) {
	ctx, err := usb.NewContext()

	if err != nil {
		return nil, err
	}

	//	ctx.Debug(4)

	devs, err2 := ctx.ListDevices(func(desc *usb.Descriptor) bool {
		return usb.Class(desc.Class) == 0 && desc.Vendor.String() == "0403" && desc.Product.String() == "6001"
	})

	if err2 != nil {
		return nil, err
	}

	var ms []Device

	for _, dev := range devs {
		m, errM := newMonome(dev, options...)
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
