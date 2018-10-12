package monome

import (
	"fmt"
	"strings"
	"time"

	"github.com/karalabe/gousb/usb"
	"github.com/karalabe/gousb/usbid"
)

func PrintUSBDevice(dev *usb.Device) {
	desc := dev.Descriptor
	fmt.Printf(
		"Bus: %v\nAddress: %v\nVendorID: %v\nProductID: %v\nclass: %v\nsubclass: %v\nprotocol: %v\nClass: %s\nDescribe: %s\n Spec: %#v\n Device: %#v\n\n",
		desc.Bus,
		desc.Address,
		desc.Vendor,
		desc.Product,
		desc.Class,
		desc.SubClass,
		desc.Protocol,
		//"Class: %s Address: %v Device: %s (%v) Spec: %s Describe: %s Protocol: %v\n",
		usbid.Classify(desc),
		usbid.Describe(desc),
		desc.Spec.String(),
		desc.Device.String(),
	)

	for _, cfg := range desc.Configs {
		fmt.Printf("\t%s\n", cfg.String())
		//		fmt.Printf("\t\tExtra: %#v\n", cfg.Extra)

		for _, iff := range cfg.Interfaces {
			fmt.Printf("\t\tinterface: %s (%v)\n", iff.String(), iff.Number)

			for _, st := range iff.Setups {
				fmt.Printf("\t\t\tsetup: %s (%v) IfClass: %d class: %v IfSubclass: %v subclass: %v, protocol: %v, alternate: %v Extra: %v\n",
					st.String(),
					st.Number,
					st.IfClass,
					usbid.Classify(st.IfClass),
					st.IfSubClass,
					usbid.Classify(st.IfSubClass),
					st.IfProtocol,
					st.Alternate,
					//					st.Extra,
				)

				for _, ep := range st.Endpoints {
					fmt.Printf("\t\t\t\tEndpoint: %s (%v) direction: %v, address: %v, attributes: %v, MaxIsoPacket: %v, MaxPacketSize: %v, PollInterval: %v, RefreshRate: %v, SynchAddress: %v, Extra: %v\n",
						ep.String(),
						ep.Number(),
						ep.Direction(),
						ep.Address,
						ep.Attributes,
						ep.MaxIsoPacket,
						ep.MaxPacketSize,
						ep.PollInterval,
						ep.RefreshRate,
						ep.SynchAddress,
						//						ep.Extra,
					)
				}

			}

		}
	}
}

// NumButtons returns the available number of buttons
func NumButtons(dev Device) uint8 {
	return dev.Rows() * dev.Cols()
}

// SwitchAll switches all lights on or off
func SwitchAll(m Device, on bool) error {
	var errs Errors
	rows := m.Rows()
	cols := m.Cols()
	for x := uint8(0); x < rows; x++ {
		for y := uint8(0); y < cols; y++ {
			errs.Add(m.Switch(x, y, on))
		}
	}
	if errs.Len() == 0 {
		return nil
	}

	if on {
		errs.Task = "switch all on"
	} else {
		errs.Task = "switch all off"
	}
	return &errs
}

// Greeter prints the name of the device on the device, followed by a flash
func Greeter(dev Device) {
	Marquee(dev, dev.String(), time.Millisecond*80)
	time.Sleep(time.Millisecond * 20)
	SwitchAll(dev, true)
	time.Sleep(time.Millisecond * 300)
	SwitchAll(dev, false)
}

// Marquee shows the given string in a marquee-like manner (from left to right)
func Marquee(m Device, s string, dur time.Duration) error {
	var errs Errors
	s = strings.ToLower(s)
	s = "   " + s + " "
	err := SwitchAll(m, false)
	if err != nil {
		e := err.(*Errors)
		e.Task = fmt.Sprintf("blank (switch all off) before marquee on device %s", m.String())
		errs.Add(e)
		return &errs
	}

	// cols is the linear row of all letters, where each letter has some cols
	//var cols = make([][8]bool, len(s)*8)
	var cols [][8]bool

	for _, l := range s {
		lt, has := Letters[l]
		if !has {
			continue
		}

		letter := make([][8]bool, LetterWidth[l])

		for pt, v := range lt {
			if v {
				letter[pt[1]][pt[0]] = true
			}
		}

		for _, col := range letter {
			cols = append(cols, col)
		}
	}

	// i is the starting point
	width := int(m.Cols())
	for i := 0; i < len(cols); i++ {
		var targetCol uint8 = 0
		for j := i; j < (i+width) && j < len(cols); j++ {

			for row, on := range cols[j] {
				//err = m.Switch(uint8(row), targetCol, on)
				if on {
					dist := width/2 - int(targetCol)
					if dist < 0 {
						dist *= (-1)
					}
					//err = m.Set(uint8(row), targetCol, uint8(15-dist))
					err = m.Set(uint8(row), targetCol, uint8(targetCol+1))
				} else {
					err = m.Set(uint8(row), targetCol, 0)
				}
				if err != nil {
					e := err.(Error)
					what := "off"
					if on {
						what = "on"
					}
					e.Task = fmt.Sprintf("switch %s %d/%d on device %s while marqueing", what, uint8(row), targetCol, m.String())
					errs.Add(e)
					return &errs
				}
			}
			targetCol++
		}
		//time.Sleep(time.Millisecond * 60)
		time.Sleep(dur)
	}

	return nil
}

// Print prints the string one letter after another
func Print(m Device, s string, dur time.Duration) error {
	var errs Errors
	s = strings.ToLower(s)
	err := SwitchAll(m, false)

	if err != nil {
		e := err.(*Errors)
		e.Task = fmt.Sprintf("blank (switch all off) before printing on device %s", m.String())
		errs.Add(e)
		return &errs
	}

	for _, l := range s {
		lt, has := Letters[l]
		if !has {
			continue
		}
		for pt, v := range lt {
			if v {
				err = m.Switch(pt[0], pt[1], true)
				if err != nil {
					e := err.(Error)
					e.Task = fmt.Sprintf("switch on %d/%d on device %s to print letter %q", pt[0], pt[1], m.String(), string(l))
					errs.Add(e)
					return &errs
				}
			}
		}
		time.Sleep(dur)
		err = SwitchAll(m, false)
		if err != nil {
			e := err.(*Errors)
			e.Task = fmt.Sprintf("blank (switch all off) after printing letter %q on device %s", string(l), m.String())
			errs.Add(e)
			return &errs
		}
		time.Sleep(dur / 2)
	}

	return nil

}

// Connections returns all connections that could be made to attached monome devices.
func Connections(options ...Option) ([]Connection, error) {
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
		return nil, USBAccessError
		//		return nil, USBContextError(err.Error())
	}

	defer ctx.Close()

	return ctx.ListDevices(func(desc *usb.Descriptor) bool {
		return usb.Class(desc.Class) == 0 && desc.Vendor.String() == vendor_id && desc.Product.String() == product_id
	})
}

func find(which string, options ...Option) ([]Connection, error) {
	ctx, err := usb.NewContext()

	if err != nil {
		return nil, USBAccessError
		//return nil, USBContextError(err.Error())
	}

	//	ctx.Debug(4)

	devs, err2 := ctx.ListDevices(func(desc *usb.Descriptor) bool {
		return usb.Class(desc.Class) == 0 && desc.Vendor.String() == VENDOR_ID && desc.Product.String() == PRODUCT_ID
	})

	if err2 != nil {
		return nil, USBAccessError
	}

	var ms []Connection

	var errs Errors

	for _, dev := range devs {
		m, errM := Connect(dev, options...)
		if errM != nil {
			errs.Add(errM)
			continue
		}

		//		fmt.Printf("opened: %s (%d buttons)\n", m, NumButtons(m))
		m.Flash()
		ms = append(ms, m)
	}

	if errs.Len() == 0 {
		return ms, nil
	}

	return ms, &errs
}
