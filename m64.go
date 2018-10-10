package monome

import (
	"fmt"
)

var _ Device = &m64{}

type m64 struct{ mn monomeConnection }

func (m *m64) String() string { return "monome64" }
func (m *m64) Rows() uint8    { return 8 }
func (m *m64) Cols() uint8    { return 8 }
func (m *m64) Switch(x, y uint8, on bool) error {
	y = changeY(y)
	var first byte = 0x30
	if on {
		first = 0x21
	}
	_, err := m.mn.Write([]byte{first, (x << 4) | y})
	if err != nil {
		var e Error
		e.Device = m.String()
		e.X = x
		e.Y = y
		e.WrappedError = err
		if on {
			e.Task = "switch on"
		} else {
			e.Task = "switch off"
		}
	}
	return nil
}

func (m *m64) Set(x, y, brightness uint8) error {
	err := m.Switch(x, y, brightness > 0)
	if err == nil {
		return nil
	}
	e := err.(Error)
	e.Task = fmt.Sprintf("set brightness to %d", brightness)
	return e
}

func (m *m64) ReadMessage() error {
	var b = make([]byte, m.mn.maxPacketSizeRead())
	got, err := m.mn.Read(b)

	if err != nil {
		return ReadError{Device: m.String(), WrappedError: err}
	}

	if got > 2 {
		for i := 2; i < got; i += 2 {
			// fmt.Printf("b[i] % X  b[i+1] % X\n", b[i], b[i+1])
			x, y := b[i+1]/16, b[i+1]%16
			y = changeY(y)
			m.mn.Handle(m.mn, x, y, b[i] == 0 /* down */)
		}
	}

	return nil
}

func changeY(in uint8) (out uint8) {
	/*
		7 -> 0
		6 -> 1
		5 -> 2
		4 -> 3
		3 -> 4
		2 -> 5
		1 -> 6
		0 -> 7
	*/

	val := int(in) - 7
	if val < 0 {
		val = val * (-1)
	}
	return uint8(val)
}
