package monome

import (
	"fmt"
	"io"
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
	Set(x, y, brightness uint8) error

	// Switches the light at x,y on or off
	// If on is true, it is a shortcut for  Set(x,y,15).
	// If on is false it is a shortcut for Set(x,y,0)
	Switch(x, y uint8, on bool) error

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

	// SwitchAll switches all lights on or off
	SwitchAll(on bool) error

	// Marquee shows the given string in a marquee-like manner (from left to right)
	Marquee(s string, dur time.Duration) error

	// Print prints the string one letter after another
	Print(s string, dur time.Duration) error

	// StartListering starts listening for button events. For errors the given errHandler is called
	StartListening(errHandler func(error))

	// StopListening stops listening for button events
	StopListening()

	// Rows returns the number of rows
	Rows() uint8

	// Cols returns the number of cols
	Cols() uint8

	// ReadMessage reads a message from the device and calls the handler if necessary
	// It should normally not be called and is just there to allow external implementations of Device
	ReadMessage() error
}

type monomeDevice interface {
	Rows() uint8
	Cols() uint8
	Set(x, y, brightness uint8) error

	// Switches the light at x,y on or off
	Switch(x, y uint8, on bool) error

	String() string

	ReadMessage() error
}

var _ monomeDevice = &m64{}
var _ monomeDevice = &m128{}
var _ monomeDevice = &testdevice{}

type monome struct {
	monomeDevice
	dev               io.Closer //    *usb.Device
	h                 Handler
	usbReader         io.Reader //  usb.Endpoint
	usbWriter         io.Writer // usb.Endpoint
	closed            bool
	mx                sync.RWMutex
	listeningStopped  chan bool
	maxpacketSizeRead uint16
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

func (m *monome) maxPacketSizeRead() uint16 {
	return m.maxpacketSizeRead
}

func (m *monome) Read(b []byte) (int, error) {
	var closed bool
	m.mx.RLock()
	closed = m.closed
	m.mx.RUnlock()
	if closed {
		return 0, ConnectionClosedError(m.String())
	}

	i, err := m.usbReader.Read(b)
	if err != nil {
		fmt.Printf("stopping read/write to device %s, because of reading error: %v\n", m.String(), err)
		m.mx.Lock()
		m.closed = true
		m.mx.Unlock()
		//m.doneChan <- true
		//close(m.doneChan)
	}
	return i, err
}

func (m *monome) Write(b []byte) (int, error) {
	var closed bool
	m.mx.RLock()
	closed = m.closed
	m.mx.RUnlock()
	if closed {
		return 0, ConnectionClosedError(m.String())
	}

	i, err := m.usbWriter.Write(b)
	if err != nil {
		fmt.Printf("stopping read/write to device %s, because of writing error: %v\n", m.String(), err)
		m.mx.Lock()
		m.closed = true
		m.mx.Unlock()
		//m.doneChan <- true
		//close(m.doneChan)
	}
	return i, err
}

func marquee(m Device, s string, dur time.Duration) error {
	var errs Errors
	s = strings.ToLower(s)
	s = "   " + s + " "
	err := m.SwitchAll(false)
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

func (m *monome) Marquee(s string, dur time.Duration) error {
	return marquee(m, s, dur)
}

func (m *monome) Print(s string, dur time.Duration) error {
	var errs Errors
	s = strings.ToLower(s)
	err := m.SwitchAll(false)

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
		err = m.SwitchAll(false)
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

func (m *monome) StartListening(errHandler func(error)) {
	go m.poll(errHandler, m)
}

func (m *monome) poll(errHandler func(error), d Device) {
	/*
		if m.ticker != nil {
			m.ticker.Stop()
		}
		m.ticker = time.NewTicker(m.pollInterval)
	*/
	ticker := time.NewTicker(m.pollInterval)
	tickChan := ticker.C

	if errHandler == nil {
		//go func() {
		for {
			select {
			case <-tickChan:
				var closed bool
				m.mx.RLock()
				closed = m.closed
				m.mx.RUnlock()
				if closed {
					return
				}
				err := d.ReadMessage()
				if err != nil {
					fmt.Printf("stop listening, because could not read from device %s: %v\n", m.String(), err)
					ticker.Stop()
					m.mx.Lock()
					m.closed = true
					m.mx.Unlock()
					//close(m.doneChan)
					return
				}
			case <-m.doneChan:
				//				fmt.Println("done channel called")
				ticker.Stop()
				m.listeningStopped <- true
				return
			}
		}
		//}()
	} else {
		//go func() {
		for {
			select {
			case <-tickChan:
				var closed bool
				m.mx.RLock()
				closed = m.closed
				m.mx.RUnlock()
				if closed {
					errHandler(ConnectionClosedError(m.String()))
					return
				}
				if err := d.ReadMessage(); err != nil {
					errHandler(err)
					fmt.Printf("stop listening, because could not read from device %s: %v\n", m.String(), err)
					ticker.Stop()
					m.mx.Lock()
					m.closed = true
					m.mx.Unlock()
					//close(m.doneChan)
					return
				}
			case <-m.doneChan:
				//				fmt.Println("done channel called")
				ticker.Stop()
				m.listeningStopped <- true
				return
			}
		}
		//}()
	}
}

func (m *monome) StopListening() {
	//	fmt.Println("stop listening called")
	if m.IsClosed() {
		return
	}
	m.doneChan <- true
	/*
		m.mx.Lock()
		m.closed = true
		//close(m.doneChan)
		m.mx.Unlock()
	*/
	<-m.listeningStopped
	//	fmt.Println("listening has stopped")
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
				m.Switch(_x, _y, false)
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

// SwitchAll switches all lights on or off.
func (m *monome) SwitchAll(on bool) error {
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

func (m *monome) IsClosed() bool {
	m.mx.RLock()
	closed := m.closed
	m.mx.RUnlock()
	return closed
}

func (m *monome) Close() (err error) {
	if m.IsClosed() {
		return nil
	}

	m.StopListening()
	m.mx.Lock()
	if !m.closed {
		m.closed = true
	}
	m.mx.Unlock()

	err = m.dev.Close()
	if err == nil {
		return
	}

	return CloseError{
		Device:       m.String(),
		WrappedError: err,
	}
}

func (m *monome) SetHandler(h Handler) {
	m.mx.Lock()
	m.h = h
	m.mx.Unlock()
}

func (m *monome) NumButtons() uint8 {
	return m.Rows() * m.Cols()
}

var defaultPollInterval = 4 * time.Millisecond

func (m *monome) Handle(d Device, x, y uint8, down bool) {
	if m.h != nil {
		m.h.Handle(d, x, y, down)
		return
	}
	action := "release"
	if down {
		action = "press"
	}
	fmt.Printf("unhandled key %s on device %s: x: %d, y: %d\n", action, d.String(), x, y)
}

// New returns a new monome device for the given usb.Device.
// Normally New should not be called directly, but
// Devices instead (which make use of New).
func New(dev *usb.Device, options ...Option) (d *monome, err error) {
	//printDevice(dev)
	var m = &monome{
		dev: dev,
		//pollInterval: 7 * time.Millisecond,
		pollInterval: defaultPollInterval,
	}
	m.doneChan = make(chan bool)
	m.listeningStopped = make(chan bool)

	for _, opt := range options {
		opt(m)
	}

	cfg := dev.Descriptor.Configs[0]
	iff := cfg.Interfaces[0]
	setup := iff.Setups[0]

	//	var t string = setup.Endpoints[0].Address

	m.usbReader, err = dev.OpenEndpoint(cfg.Config, iff.Number, setup.Number, setup.Endpoints[0].Address)

	if err != nil {
		var e ConnectError
		e.USBDevice = dev
		e.USBEndPoint.Purpose = "usbReader"
		e.USBEndPoint.Number = 0
		e.USBEndPoint.Config = cfg.Config
		e.USBEndPoint.Interface = iff.Number
		e.USBEndPoint.Setup = setup.Number
		e.USBEndPoint.Info = setup.Endpoints[0]
		e.WrappedError = err
		return nil, &e
	}
	m.maxpacketSizeRead = setup.Endpoints[0].MaxPacketSize

	//	m.maxPacketSizeRead = setup.Endpoints[0].MaxPacketSize

	m.usbWriter, err = dev.OpenEndpoint(cfg.Config, iff.Number, setup.Number, setup.Endpoints[1].Address)
	if err != nil {
		var e ConnectError
		e.USBDevice = dev
		e.USBEndPoint.Purpose = "usbWriter"
		e.USBEndPoint.Number = 1
		e.USBEndPoint.Config = cfg.Config
		e.USBEndPoint.Interface = iff.Number
		e.USBEndPoint.Setup = setup.Number
		e.USBEndPoint.Info = setup.Endpoints[1]
		e.WrappedError = err
		return nil, &e
	}

	var errs Errors

	_, err = m.usbWriter.Write([]byte{0x01, 0x00, 0x00})

	if err != nil {
		errs.Add(err)
		errs.Task = "initial write to find out the kind of monome"
		return nil, &errs
	}
	time.Sleep(time.Second)
	var b = make([]byte, int(m.maxpacketSizeRead))
	_, err = m.usbReader.Read(b)

	if err != nil {
		errs.Add(err)
		errs.Task = "initial read to find out the kind of monome"
		return nil, &errs
	}

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

	var e UnknownMonomeError
	e.Response = b
	e.USBDevice = dev
	e.USBReaderEndPoint = setup.Endpoints[0]
	e.USBWriterEndPoint = setup.Endpoints[1]
	return nil, &e

}

// serial = udev_device_get_property_value(d, "ID_SERIAL_SHORT");

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
