package monome

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/karalabe/gousb/usb"
	"github.com/karalabe/gousb/usbid"
)

// Handler responds to a pressing or releasing action on a button
type Handler interface {

	// Handle is the callback that is called if a button is pressed (down=true)
	// or released (down=false)
	Handle(d Device, x, y uint8, down bool)
}

// HandlerFunc is a function that acts as a Handler
type HandlerFunc func(d Device, x, y uint8, down bool)

func (h HandlerFunc) Handle(d Device, x, y uint8, down bool) {
	h(d, x, y, down)
}

// Device is a monome device
type Device interface {
	// Set sets the button at position x,y to the given brightness
	// If the connection has been closed, nothing is sent
	// From the monome docs about brightness levels:
	// [0, 3] - off
	// [4, 7] - low intensity
	// [8, 11] - medium intensity
	// [12, 15] - high intensity
	// June 2012 devices allow the full 16 intensity levels.
	Set(x, y, brightness uint8)

	// On is a shortcut for Set(x,y,15)
	On(x, y uint8)

	// Off is a shortcut for Set(x,y,0)
	Off(x, y uint8)

	// Close closes the connection to the monome
	Close() error

	// IsClosed returns wether the connection is closed
	IsClosed() bool

	// NumButtons returns the available number of buttons
	NumButtons() uint8

	// String returns an identifier as a string (name)
	String() string

	// SetHandler set the active handler for the device
	SetHandler(Handler)

	// AllOff switches all lights off
	AllOff()

	// AllOn switches all lights on
	AllOn()

	// Marquee shows the given string in a marquee-like manner (from left to right)
	Marquee(s string, dur time.Duration)

	// Print prints the string one letter after another
	Print(s string, dur time.Duration)

	// StartListering starts listening for button events. For errors the given errHandler is called
	StartListening(errHandler func(error))

	// StopListening stops listening for button events
	StopListening()

	// Rows returns the number of rows
	Rows() uint8

	// Cols returns the number of cols
	Cols() uint8

	// Read reads a message from the device and calls the handler if necessary
	// It should normally not be called and is just there to allow external implementations of Device
	Read() error
}

type monomeDevice interface {
	Rows() uint8
	Cols() uint8
	Set(x, y, brightness uint8)
	On(x, y uint8)
	Off(x, y uint8)
	Read() error
	String() string
}

var _ monomeDevice = &m64{}
var _ monomeDevice = &m128{}

type monome struct {
	monomeDevice
	dev               *usb.Device
	h                 Handler
	usbReader         usb.Endpoint
	usbWriter         usb.Endpoint
	closed            bool
	mx                sync.RWMutex
	maxPacketSizeRead uint16
	//	ticker            *time.Ticker
	pollInterval time.Duration
	/*
		usbConfig         uint8
		usbIffNumber      uint8
		usbSetupNumber    uint8
		usbReaderAddress  uint8
		usbWriterAddress  uint8
	*/
	doneChan chan bool
}

func marquee(m Device, s string, dur time.Duration) {
	s = strings.ToLower(s)
	m.AllOff()

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
				if on {
					m.On(uint8(row), targetCol)
				} else {
					m.Off(uint8(row), targetCol)
				}
			}
			targetCol++
		}
		//time.Sleep(time.Millisecond * 60)
		time.Sleep(dur)
	}
}

func (m *monome) Marquee(s string, dur time.Duration) {
	marquee(m, s, dur)
}

func (m *monome) Print(s string, dur time.Duration) {
	s = strings.ToLower(s)
	m.AllOff()
	for _, l := range s {
		lt, has := Letters[l]
		if !has {
			continue
		}
		for pt, v := range lt {
			if v {
				m.On(pt[0], pt[1])
			}
		}
		time.Sleep(dur)
		m.AllOff()
		time.Sleep(dur / 2)
	}

}

func (m *monome) StartListening(errHandler func(error)) {
	m.poll(errHandler, m)
}

func (m *monome) poll(errHandler func(error), d Device) {
	/*
		if m.ticker != nil {
			m.ticker.Stop()
		}
		m.ticker = time.NewTicker(m.pollInterval)
	*/
	m.doneChan = make(chan bool)
	tickChan := time.NewTicker(m.pollInterval).C

	if errHandler == nil {
		go func() {
			for {
				select {
				case <-tickChan:
					d.Read()
				case <-m.doneChan:
					return
				}
			}
		}()
	} else {
		go func() {
			for {
				select {
				case <-tickChan:
					if err := d.Read(); err != nil {
						errHandler(err)
					}
				case <-m.doneChan:
					return
				}
			}
		}()
	}
}

func (m *monome) StopListening() {
	m.doneChan <- true
}

func (m *monome) Flash() {
	m.worm()
	//	time.Sleep(time.Millisecond * 100)
	//	m.AllOn()
	//	time.Sleep(time.Millisecond * 300)
	//	m.AllOff()
}

func (m *monome) worm() {
	rows := m.Rows()
	cols := m.Cols()

	var wg sync.WaitGroup
	var mx sync.Mutex
	var flip bool

	var i time.Duration = 0

	for x := uint8(0); x < rows; x++ {
		y := 0
		if flip {
			y = int(cols) - 1
		}
		for y >= 0 && y < int(cols) {

			time.Sleep(time.Millisecond * 4)
			mx.Lock()
			wg.Add(1)
			//m.On(x, uint8(y))
			m.Set(x, uint8(y), 4+x)
			mx.Unlock()
			//			println(x, y)

			go func(_x, _y uint8) {
				time.Sleep(time.Millisecond * (i*i*i*7 + 47))
				mx.Lock()
				m.Off(_x, _y)
				wg.Done()
				mx.Unlock()
			}(x, uint8(y))

			if flip {
				y--
			} else {
				y++
			}
		}
		flip = !flip
	}

	wg.Wait()
}

func (m *monome) AllOff() {
	rows := m.Rows()
	cols := m.Cols()
	for x := uint8(0); x < rows; x++ {
		for y := uint8(0); y < cols; y++ {
			m.Off(x, y)
		}
	}
}

func (m *monome) AllOn() {
	rows := m.Rows()
	cols := m.Cols()
	for x := uint8(0); x < rows; x++ {
		for y := uint8(0); y < cols; y++ {
			m.On(x, y)
		}
	}
}

func (m *monome) IsClosed() bool {
	m.mx.RLock()
	closed := m.closed
	m.mx.RUnlock()
	return closed
}

func (m *monome) Close() (err error) {
	m.mx.Lock()
	if !m.closed {
		m.closed = true
		err = m.dev.Close()
	}
	m.mx.Unlock()
	return err
}

func (m *monome) SetHandler(h Handler) {
	m.mx.Lock()
	m.h = h
	m.mx.Unlock()
}

func (m *monome) NumButtons() uint8 {
	return m.Rows() * m.Cols()
}

// newMonome returns a new monome device for the given usb.Device.
// Normally New should not be called directly, but
// Monome64, Monome128 or All instead (which make use of New).
func newMonome(dev *usb.Device, options ...Option) (d *monome, err error) {
	//printDevice(dev)
	var m = &monome{
		dev: dev,
		//pollInterval: 7 * time.Millisecond,
		pollInterval: 4 * time.Millisecond,
	}

	for _, opt := range options {
		opt(m)
	}

	cfg := dev.Descriptor.Configs[0]
	iff := cfg.Interfaces[0]
	setup := iff.Setups[0]

	m.usbReader, err = dev.OpenEndpoint(cfg.Config, iff.Number, setup.Number, setup.Endpoints[0].Address)

	if err != nil {
		return nil, err
	}
	m.maxPacketSizeRead = setup.Endpoints[0].MaxPacketSize

	//	m.maxPacketSizeRead = setup.Endpoints[0].MaxPacketSize

	m.usbWriter, err = dev.OpenEndpoint(cfg.Config, iff.Number, setup.Number, setup.Endpoints[1].Address)
	if err != nil {
		return nil, err
	}

	m.usbWriter.Write([]byte{0x01, 0x00, 0x00})
	time.Sleep(time.Second)
	var b = make([]byte, m.maxPacketSizeRead)
	ln, err := m.usbReader.Read(b)

	if err != nil {
		return nil, err
	}

	_ = ln

	//	fmt.Printf("% X (%s) len: %v\n", b[:ln], string(b[:ln]), ln)

	//	monome16x8 = "m1000293" -> 0xF4365 oder 1000293
	//
	//	monome8x8  = "m64-0348" -> 0x15C

	if string(b[3:13]) == "monome 128" {
		m.monomeDevice = &m128{m}
		//		m.Flash()
		return m, nil
	}

	if b[0] == 0x31 {
		//		fmt.Println("monome64")
		m.monomeDevice = &m64{m}
		//		m.Flash()
		return m, nil
	}
	return nil, fmt.Errorf("unknown % X (%s)\n", b, string(b))

}

// serial = udev_device_get_property_value(d, "ID_SERIAL_SHORT");

func printDevice(dev *usb.Device) {
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
