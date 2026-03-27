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

func (p *testPrinter) SetReverse(enabled bool) {
	if enabled {
		p.operations = append(p.operations, "reverse:on")
		return
	}
	p.operations = append(p.operations, "reverse:off")
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

func TestRenderBillPrintsChildItemsBelowNameBeforeQtyRow(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.Items[0].Children[0].Quantity = 2
	data.Items[0].Children[0].UnitPrice = 12.96
	data.Items[0].Children[0].FinalAmount = 99.99

	RenderBill(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")

	expectedChildLine := "  + Extra Hot Fudge (12.96) (x2)\n"
	if !strings.Contains(output, expectedChildLine) {
		t.Fatalf("expected child item with rate and qty on one line, got %s", output)
	}

	childPos := strings.Index(output, expectedChildLine)
	namePos := strings.Index(output, "Anjeer Ice Cream\n")
	mainQtyRow := fmt.Sprintf("%-18s %5s %10s %12s\n", "", "1", "72.04", "72.04")
	mainQtyPos := strings.Index(output, mainQtyRow)

	if namePos < 0 || childPos < namePos {
		t.Fatalf("expected child line after main item name, got %s", output)
	}
	if mainQtyPos < 0 || mainQtyPos < childPos {
		t.Fatalf("expected main qty/rate/amount row after child items, got %s", output)
	}

	if strings.Contains(output, "99.99") {
		t.Fatalf("expected child final amount to be omitted, got %s", output)
	}
}

func TestRenderBillChildItemSingleQtyOmitsQuantitySuffix(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.Items[0].Children[0].Quantity = 1
	data.Items[0].Children[0].UnitPrice = 12.96

	RenderBill(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")

	expectedChildLine := "  + Extra Hot Fudge (12.96)\n"
	if !strings.Contains(output, expectedChildLine) {
		t.Fatalf("expected child item without qty suffix when qty=1, got %s", output)
	}
	if strings.Contains(output, "(x1)") {
		t.Fatalf("expected no (x1) suffix for single quantity child, got %s", output)
	}
}

func TestRenderBillPrintsExternalIDAboveTaxInvoice(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.ExternalID = "NAT-ORDER-12345678"

	RenderBill(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	if !strings.Contains(output, "NAT-ORDER-1234") {
		t.Fatalf("expected external ID prefix in output, got %s", output)
	}
	if !strings.Contains(output, " 5678 ") {
		t.Fatalf("expected padded bold last four characters, got %s", output)
	}

	operations := strings.Join(printer.operations, "|")
	if !strings.Contains(operations, "reverse:off|size:1:1|write:NAT-ORDER-1234|bold:on|reverse:on|double-strike:on|write: 5678 |reverse:off|bold:off|write:\n\n|size:0:0") {
		t.Fatalf("expected external ID formatting with reverse, got %s", operations)
	}
}

func TestRenderBillUsesExternalOrderIDFallback(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.ExternalID = nil
	data.ExternalOrderID = "7896135193"

	RenderBill(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	if !strings.Contains(output, "789613") {
		t.Fatalf("expected external order ID prefix, got %s", output)
	}
	if !strings.Contains(output, " 5193 ") {
		t.Fatalf("expected padded bold last four characters, got %s", output)
	}
	if !strings.Contains(output, "TAX INVOICE") {
		t.Fatalf("expected tax invoice title after external order ID, got %s", output)
	}
}

func TestRenderKOTUsesExternalOrderIDFallback(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.ExternalID = nil
	data.ExternalOrderID = "7896135193"

	RenderKOT(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	if !strings.Contains(output, "789613") {
		t.Fatalf("expected external order ID prefix, got %s", output)
	}
	if !strings.Contains(output, " 5193 ") {
		t.Fatalf("expected padded bold last four characters, got %s", output)
	}
	if !strings.Contains(output, "KOT") {
		t.Fatalf("expected KOT title after external order ID, got %s", output)
	}
}

func TestGetExternalIDPrefersExternalIDOverExternalOrderID(t *testing.T) {
	data := OrderData{
		ExternalID:      "PREFERRED-1234",
		ExternalOrderID: "FALLBACK-5678",
	}

	result := data.GetExternalID()
	if result != "PREFERRED-1234" {
		t.Fatalf("expected ExternalID to take precedence, got %q", result)
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
	if !strings.Contains(output, "  + Extra Hot Fudge\n") {
		t.Fatalf("expected child item line without qty suffix when qty=1 in KOT output, got %s", output)
	}
	if !strings.Contains(operations, "size:0:1|write:2x Anjeer Ice Cream Deluxe\n|size:0:0") {
		t.Fatalf("expected entire item line double-height in KOT operations, got %s", operations)
	}
}

func TestRenderKOTChildShowsQuantityWhenGreaterThanOne(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.Items[0].Children[0].Quantity = 3

	RenderKOT(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	if !strings.Contains(output, "  + Extra Hot Fudge (x3)\n") {
		t.Fatalf("expected KOT child item with (x3) suffix, got %s", output)
	}
}

func TestRenderKOTPrintsPunchedByName(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.CashierName = "Deepak Sharma"

	RenderKOT(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	if !strings.Contains(output, "Punched By: Deepak Sharma\n") {
		t.Fatalf("expected punched-by line in KOT output, got %s", output)
	}
	if strings.Index(output, "Punched By: Deepak Sharma\n") > strings.Index(output, "Date: ") {
		t.Fatalf("expected punched-by line before date, got %s", output)
	}
}

func TestRenderKOTPrintsExternalIDAboveTitleAndPaxAfterInvoice(t *testing.T) {
	printer := &testPrinter{}
	data := GetSampleOrderData()
	data.ExternalID = "KOT-ORDER-12345678"
	data.InvoiceNo = "INV-1001"
	data.Pax = 4

	RenderKOT(printer, data, "80mm", false)

	output := strings.Join(printer.writes, "")
	operations := strings.Join(printer.operations, "|")

	if !strings.Contains(output, "KOT-ORDER-1234") {
		t.Fatalf("expected KOT external ID prefix, got %s", output)
	}
	if !strings.Contains(output, " 5678 ") {
		t.Fatalf("expected padded bold last four characters, got %s", output)
	}
	if !strings.Contains(output, "KOT\n") {
		t.Fatalf("expected KOT title after external ID, got %s", output)
	}
	if !strings.Contains(output, "Order #: INV-1001  Pax: 4\n") {
		t.Fatalf("expected pax after invoice on KOT order line, got %s", output)
	}
	if !strings.Contains(operations, "reverse:on|double-strike:on|write: 5678 |reverse:off|bold:off|write:\n\n|size:0:0") {
		t.Fatalf("expected KOT external ID reverse formatting, got %s", operations)
	}
	if strings.Index(output, "Order #: INV-1001  Pax: 4\n") > strings.Index(output, "Date: ") {
		t.Fatalf("expected order and pax line before date, got %s", output)
	}
}
