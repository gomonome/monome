package monome

type monome128 struct{ mn *monome }

func (m *monome128) String() string { return "monome128" }
func (m *monome128) Rows() uint8    { return 8 }
func (m *monome128) Cols() uint8    { return 16 }
func (m *monome128) Off(x, y uint8) { m.Set(x, y, 0) }
func (m *monome128) On(x, y uint8)  { m.Set(x, y, 15) }

func (m *monome128) Set(x, y, brightness uint8) {
	if brightness > 15 {
		brightness = 15
	}
	m.mn.usbWriter.Write([]byte{24, y, x, brightness})
}

func (m *monome128) Read() error {
	var b = make([]byte, m.mn.maxPacketSizeRead)
	got, err := m.mn.usbReader.Read(b)

	if err != nil {
		return err
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
