package printer

import (
	"bytes"
	"testing"
)

func TestEscposAdapterOpenCashDrawerWritesPulse(t *testing.T) {
	adapter := NewEscposAdapter()

	adapter.OpenCashDrawer()

	expected := []byte{0x1B, 0x70, 0x00, 0x19, 0xFA}
	if !bytes.Equal(adapter.GetBytes(), expected) {
		t.Fatalf("expected drawer pulse %v, got %v", expected, adapter.GetBytes())
	}
}

