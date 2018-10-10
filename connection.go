package monome

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/karalabe/gousb/usb"
)

// Handler responds to a pressing or releasing action on a button
type Handler interface {

	// Handle is the callback that is called if a button is pressed (down=true)
	// or released (down=false)
	Handle(d Connection, x, y uint8, down bool)
}

// HandlerFunc is a function that acts as a Handler
type HandlerFunc func(d Connection, x, y uint8, down bool)

func (h HandlerFunc) Handle(d Connection, x, y uint8, down bool) {
	h(d, x, y, down)
}

// Connection is a connection to a monome device
type Connection interface {
	// Close closes the connection to the monome
	Close() error

	// IsClosed returns wether the connection is closed
	IsClosed() bool

	// SetHandler set the active handler for the device
	SetHandler(Handler)

	// StartListering starts listening for button events. For errors the given errHandler is called
	StartListening(errHandler func(error))

	// StopListening stops listening for button events
	StopListening()

	Device
}

type connection struct {
	Device
	dev               io.Closer //    *usb.Device
	h                 Handler
	usbReader         io.Reader //  usb.Endpoint
	usbWriter         io.Writer // usb.Endpoint
	closed            bool
	mx                sync.RWMutex
	listeningStopped  chan bool
	maxpacketSizeRead uint16
	pollInterval      time.Duration
	doneChan          chan bool
}

type monomeConnection interface {
	io.ReadWriter
	Handler
	Connection
	maxPacketSizeRead() uint16
}

func (m *connection) maxPacketSizeRead() uint16 {
	return m.maxpacketSizeRead
}

func (m *connection) Read(b []byte) (int, error) {
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

func (m *connection) Write(b []byte) (int, error) {
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

func (m *connection) StartListening(errHandler func(error)) {
	go m.poll(errHandler, m)
}

func (m *connection) poll(errHandler func(error), d Connection) {
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

func (m *connection) StopListening() {
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

func (m *connection) Flash() {
	m.worm()
	//	time.Sleep(time.Millisecond * 100)
	//	m.AllOn()
	//	time.Sleep(time.Millisecond * 300)
	//	m.AllOff()
}

func (m *connection) worm() {
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

func (m *connection) IsClosed() bool {
	m.mx.RLock()
	closed := m.closed
	m.mx.RUnlock()
	return closed
}

func (m *connection) Close() (err error) {
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

func (m *connection) SetHandler(h Handler) {
	m.mx.Lock()
	m.h = h
	m.mx.Unlock()
}

var defaultPollInterval = 4 * time.Millisecond

func (m *connection) Handle(d Connection, x, y uint8, down bool) {
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

// Connect returns a new Connection to the given usb.Device.
// Normally New should not be called directly, but
// Devices instead (which make use of New).
func Connect(dev *usb.Device, options ...Option) (d *connection, err error) {
	//printDevice(dev)
	var m = &connection{
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
		m.Device = &m128{m}
		//		m.Flash()
		return m, nil
	}

	if b[0] == 0x31 {
		//		fmt.Println("monome64")
		m.Device = &m64{m}
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
