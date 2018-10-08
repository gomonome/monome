package monome

import "fmt"

type m128 struct{ mn *monome }

func (m *m128) String() string { return "monome128" }
func (m *m128) Rows() uint8    { return 8 }
func (m *m128) Cols() uint8    { return 16 }
func (m *m128) Switch(x, y uint8, on bool) error {
	var brightness uint8
	if on {
		brightness = 15
	}
	err := m.Set(x, y, brightness)
	if err == nil {
		return nil
	}

	e := err.(Error)
	if on {
		e.Task = "switch on"
	} else {
		e.Task = "switch off"
	}
	return e
}

func (m *m128) Set(x, y, brightness uint8) error {
	if brightness > 15 {
		brightness = 15
	}
	_, err := m.mn.usbWriter.Write([]byte{24, y, x, brightness})
	if err != nil {
		var e Error
		e.Device = m.String()
		e.X = x
		e.Y = y
		e.WrappedError = err
		e.Task = fmt.Sprintf("set brightness to %d", brightness)
		return e
	}
	return nil
}

func (m *m128) Read() error {
	var b = make([]byte, m.mn.maxPacketSizeRead)
	got, err := m.mn.usbReader.Read(b)

	if err != nil {
		return ReadError{
			Device:       m.String(),
			WrappedError: err,
		}
	}

	if m.mn.h != nil && got > 2 {
		for i := 2; i < got-2; i += 3 {
			if b[i] == 0 {
				continue
			}
			x, y := b[i+2], b[i+1]
			m.mn.h.Handle(m.mn, x, y, b[i] == 0x21 /* down */)
		}
	}
	return nil
}
