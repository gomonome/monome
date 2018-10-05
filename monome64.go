package monome

type monome64 struct{ mn *monome }

func (m *monome64) String() string { return "monome64" }
func (m *monome64) Rows() uint8    { return 8 }
func (m *monome64) Cols() uint8    { return 8 }
func (m *monome64) Off(x, y uint8) {
	y = changeY(y)
	m.mn.usbWriter.Write([]byte{0x30, (x << 4) | y})
}
func (m *monome64) On(x, y uint8) {
	y = changeY(y)
	m.mn.usbWriter.Write([]byte{0x21, (x << 4) | y})
}

func (m *monome64) Set(x, y, brightness uint8) {
	if brightness > 0 {
		m.On(x, y)
		return
	}
	m.Off(x, y)
}

func (m *monome64) Read() error {
	var b = make([]byte, m.mn.maxPacketSizeRead)
	got, err := m.mn.usbReader.Read(b)

	if err != nil {
		return err
	}

	if m.mn.h != nil && got > 2 {
		for i := 2; i < got; i += 2 {
			// fmt.Printf("b[i] % X  b[i+1] % X\n", b[i], b[i+1])
			x, y := b[i+1]/16, b[i+1]%16
			y = changeY(y)
			m.mn.h.Handle(m.mn, x, y, b[i] == 0 /* down */)
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
