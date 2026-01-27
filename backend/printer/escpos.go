package printer

import (
	"bytes"
	"ts-escpos/backend/receipt"
)

var _ receipt.Printer = (*EscposAdapter)(nil)

// EscposAdapter implements receipt.Printer interface
// It generates ESC/POS commands into a buffer
type EscposAdapter struct {
	buf *bytes.Buffer
}

func NewEscposAdapter() *EscposAdapter {
	return &EscposAdapter{
		buf: new(bytes.Buffer),
	}
}

func (e *EscposAdapter) Init() {
	e.buf.Write([]byte{0x1B, 0x40}) // ESC @
}

func (e *EscposAdapter) SetAlign(align string) {
	switch align {
	case "center":
		e.buf.Write([]byte{0x1B, 0x61, 0x01})
	case "right":
		e.buf.Write([]byte{0x1B, 0x61, 0x02})
	default: // left
		e.buf.Write([]byte{0x1B, 0x61, 0x00})
	}
}

func (e *EscposAdapter) SetFont(font string) {
	// 0 = Font A, 1 = Font B
	if font == "B" {
		e.buf.Write([]byte{0x1B, 0x4D, 0x01})
	} else {
		e.buf.Write([]byte{0x1B, 0x4D, 0x00})
	}
}

func (e *EscposAdapter) SetSize(width, height uint8) {
	// ESC ! n (Select print mode) or GS ! n (Select character size)
	// GS ! n
	// 0-7: Width, 4-7: Height (bits)
	// Actually bits 0-3 width, 4-7 height.
	// width=0 means normal (x1), width=1 means x2
	n := (height << 4) | width
	e.buf.Write([]byte{0x1D, 0x21, n})
}

func (e *EscposAdapter) Write(data string) {
	// Handle encoding if necessary. For now assume UTF-8/ASCII
	// Some printers need specific codepages for non-ASCII.
	e.buf.WriteString(data)
}

func (e *EscposAdapter) Feed(n uint8) {
	// ESC d n
	e.buf.Write([]byte{0x1B, 0x64, n})
}

func (e *EscposAdapter) Cut() {
	// GS V m
	// 66: Feed paper to (cutting position + n x vertical motion unit) and perform a partial cut
	e.buf.Write([]byte{0x1D, 0x56, 0x42, 0x00})
}

func (e *EscposAdapter) GetBytes() []byte {
	return e.buf.Bytes()
}
