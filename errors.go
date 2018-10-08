package monome

import (
	"fmt"

	"github.com/karalabe/gousb/usb"
)

type UnknownMonomeError struct {
	Response          []byte
	USBDevice         *usb.Device
	USBWriterEndPoint usb.EndpointInfo
	USBReaderEndPoint usb.EndpointInfo
}

func (e *UnknownMonomeError) Error() string {
	return fmt.Sprintf("unknown monome kind (got % X (%s))\n", e.Response, string(e.Response))
}

type Error struct {
	X            uint8
	Y            uint8
	Device       string
	WrappedError error
	Task         string
}

func (e Error) Error() string {
	return fmt.Sprintf("device %q had the following error when trying to set %d/%d in order to %s: %v", e.Device, e.X, e.Y, e.Task, e.WrappedError)
}

type Errors struct {
	Task   string
	Errors []error
}

func (m *Errors) Len() int {
	return len(m.Errors)
}

func (m *Errors) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m *Errors) Error() string {
	return fmt.Sprintf("%d errors happened while setting, typecast to monome.Errors to inspect them", m.Len())
}

type USBContextError string

func (e USBContextError) Error() string {
	return string(e)
}

type ConnectError struct {
	USBDevice   *usb.Device
	USBEndPoint struct {
		Purpose   string
		Number    int
		Config    uint8
		Interface uint8
		Setup     uint8
		Info      usb.EndpointInfo
	}
	WrappedError error
}

func (m *ConnectError) Error() string {
	return fmt.Sprintf("the following error happened while trying to connect to USB endpoint %d as %s: %v", m.USBEndPoint.Number, m.USBEndPoint.Purpose, m.WrappedError)
}

type CloseError struct {
	Device       string
	WrappedError error
}

func (e CloseError) Error() string {
	return fmt.Sprintf("when closing device %q the following error occured: %v", e.Device, e.WrappedError)
}

type ReadError struct {
	Device       string
	WrappedError error
}

func (e ReadError) Error() string {
	return fmt.Sprintf("when reading from device %q the following error occured: %v", e.Device, e.WrappedError)
}
