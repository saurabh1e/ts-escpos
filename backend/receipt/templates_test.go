package receipt

import (
	"fmt"
	"strings"
	"testing"
)

type testPrinter struct {
	operations []string
	writes     []string
}

func (p *testPrinter) Init() {
	p.operations = append(p.operations, "init")
}

func (p *testPrinter) SetAlign(align string) {
	p.operations = append(p.operations, "align:"+align)
}

func (p *testPrinter) SetFont(font string) {
	p.operations = append(p.operations, "font:"+font)
}

func (p *testPrinter) SetBold(bold bool) {
	if bold {
		p.operations = append(p.operations, "bold:on")
		return
	}
	p.operations = append(p.operations, "bold:off")
}

func (p *testPrinter) SetDoubleStrike(enabled bool) {
	if enabled {
		p.operations = append(p.operations, "double-strike:on")
		return
	}
	p.operations = append(p.operations, "double-strike:off")
}

func (p *testPrinter) SetSize(width, height uint8) {
	p.operations = append(p.operations, "size:"+string(rune('0'+width))+":"+string(rune('0'+height)))
}

func (p *testPrinter) Write(data string) {
	p.writes = append(p.writes, data)
	p.operations = append(p.operations, "write:"+data)
}

func (p *testPrinter) Feed(n uint8) {
	p.operations = append(p.operations, "feed")
}

func (p *testPrinter) Cut() {
	p.operations = append(p.operations, "cut")
}

func (p *testPrinter) PrintQRCode(data string) {
	p.operations = append(p.operations, "qrcode")
}

func (p *testPrinter) PrintImage(filePath string) {
	p.operations = append(p.operations, "image")
}

func TestRenderBillPrintsDuplicateHeaderAtTop(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()

	RenderBill(printer, data, "80mm", true)

	if len(printer.writes) < 3 {
		t.Fatalf("expected initial writes, got %+v", printer.writes)
	}
	if printer.writes[0] != "DUPLICATE\n" {
		t.Fatalf("expected first write to be DUPLICATE, got %q", printer.writes[0])
	}
	if printer.writes[2] != "TAX INVOICE\n" {
		t.Fatalf("expected bill title after duplicate header, got %q", printer.writes[2])
	}

	operations := strings.Join(printer.operations, "|")
	if !strings.Contains(operations, "bold:on|size:1:1|write:DUPLICATE\n|size:0:0|bold:off") {
		t.Fatalf("expected bold large duplicate header, got %s", operations)
	}
}

func TestRenderKOTPrintsDuplicateHeaderAtTop(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()

	RenderKOT(printer, data, "80mm", true)

	if len(printer.writes) < 3 {
		t.Fatalf("expected initial writes, got %+v", printer.writes)
	}
	if printer.writes[0] != "DUPLICATE\n" {
		t.Fatalf("expected first write to be DUPLICATE, got %q", printer.writes[0])
	}
	if printer.writes[2] != "KOT\n" {
		t.Fatalf("expected KOT title after duplicate header, got %q", printer.writes[2])
	}
}

func TestRenderBillOmitsDuplicateHeaderForNormalPrint(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()

	RenderBill(printer, data, "80mm", false)

	if len(printer.writes) == 0 {
		t.Fatal("expected bill output")
	}
	if printer.writes[0] != "TAX INVOICE\n" {
		t.Fatalf("expected normal bill title first, got %q", printer.writes[0])
	}
}

func TestRenderBillPrintsOnlyChildUnitPrice(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.Items[0].Children[0].Quantity = 2
	data.Items[0].Children[0].UnitPrice = 12.96
	data.Items[0].Children[0].FinalAmount = 99.99

	RenderBill(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	expectedRow := fmt.Sprintf("%-18s %5s %10s %12s\n", "", "2", "12.96", "")
	if !strings.Contains(output, "  + Extra Hot Fudge\n") {
		t.Fatalf("expected child item label in bill output, got %s", output)
	}
	if !strings.Contains(output, expectedRow) {
		t.Fatalf("expected child row to print only unit price, got %q", output)
	}
	if strings.Contains(output, "99.99") {
		t.Fatalf("expected child final amount to be omitted, got %s", output)
	}
}

func TestRenderKOTHighlightsQuantityOnSameLine(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.Items[0].Quantity = 2
	data.Items[0].ProductName = "Anjeer Ice Cream Deluxe"

	RenderKOT(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	operations := strings.Join(printer.operations, "|")
	if !strings.Contains(output, "QTY x ITEM\n") {
		t.Fatalf("expected updated KOT heading, got %s", output)
	}
	if !strings.Contains(output, "2x Anjeer Ice Cream Deluxe\n") {
		t.Fatalf("expected quantity and item name on the same line, got %s", output)
	}
	if !strings.Contains(output, "    Note: Single scoop cup\n") {
		t.Fatalf("expected wrapped note prefix in KOT output, got %s", output)
	}
	if !strings.Contains(output, "  + 1x Extra Hot Fudge\n") {
		t.Fatalf("expected child item line in KOT output, got %s", output)
	}
	if !strings.Contains(operations, "size:0:1|write:2x |size:0:0|write:Anjeer Ice Cream Deluxe\n") {
		t.Fatalf("expected emphasized quantity in KOT operations, got %s", operations)
	}
}
