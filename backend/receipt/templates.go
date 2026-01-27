package receipt

import (
	"encoding/json"
	"fmt"
	"strings"
)

type OrderItemChildren []OrderItem

func (c *OrderItemChildren) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == `""` || str == "null" {
		*c = nil
		return nil
	}
	var items []OrderItem
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	*c = OrderItemChildren(items)
	return nil
}

type OrderItem struct {
	Name     string            `json:"name"`
	Quantity int               `json:"quantity"`
	Price    float64           `json:"price"`
	Children OrderItemChildren `json:"children"`
}

type OrderData struct {
	InvoiceNo    string      `json:"invoiceNo"`
	Date         string      `json:"date"`
	CustomerName string      `json:"customerName"`
	TableNo      string      `json:"tableNo"`
	Items        []OrderItem `json:"items"`
	SubTotal     float64     `json:"subTotal"`
	Tax          float64     `json:"tax"`
	Total        float64     `json:"total"`
	PaymentMode  string      `json:"paymentMode"`
}

// Formatters for ESC/POS
// We will return a slice of byte commands or just plain text with formatting instructions?
// The go-escpos library usually takes commands.
// For modularity, let's have this package generate a list of "Print Actions" or just string content
// that the escpos wrapper can execute.
// However, to keep it simple with the library `github.com/DevLumuz/go-escpos`,
// we might want to pass the `*escpos.Escpos` object to these functions, NOT good for separation.
// BETTER: Return a struct or interface that describes the receipt, or return strings with embedded commands if possible.
//
// OR: Just return the raw raw bytes if possible, but that couples it to specific esc/pos implementation.
//
// Let's assume we pass a wrapper interface or just return text lines with some metadata.
//
// Actually, `go-escpos` works by calling methods on the writer.
// So let's define a `Printer` interface here that `escpos` package implements.

type Printer interface {
	Init()
	SetAlign(align string)
	SetFont(font string)
	SetSize(width, height uint8)
	Write(data string)
	Feed(n uint8)
	Cut()
}

func RenderKOT(p Printer, data OrderData, size string) {
	width := 32 // Default 58mm
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetAlign("center")
	p.SetSize(1, 1) // Double height/width for title
	p.Write("KOT\n")
	p.SetSize(0, 0) // Normal
	p.Write(fmt.Sprintf("Table: %s\n", data.TableNo))
	p.Write(fmt.Sprintf("Date: %s\n", data.Date))
	p.Write(strings.Repeat("-", width) + "\n")

	p.SetAlign("left")
	// Header
	// Qty Name
	p.Write(fmt.Sprintf("%-4s %s\n", "Qty", "Item"))
	p.Write(strings.Repeat("-", width) + "\n")

	for _, item := range data.Items {
		p.Write(fmt.Sprintf("%-4d %s\n", item.Quantity, item.Name))
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.Feed(3)
	p.Cut()
}

func RenderBill(p Printer, data OrderData, size string) {
	width := 32 // Default 58mm
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetAlign("center")
	p.SetSize(1, 1)
	p.Write("INVOICE\n")
	p.SetSize(0, 0)
	p.Write(fmt.Sprintf("#%s\n", data.InvoiceNo))
	p.Write(data.Date + "\n")
	p.Write(strings.Repeat("-", width) + "\n")

	p.SetAlign("left")
	// Item Price Total
	// We need careful formatting here.

	for _, item := range data.Items {
		lineTotal := float64(item.Quantity) * item.Price
		p.Write(fmt.Sprintf("%s\n", item.Name))
		p.Write(fmt.Sprintf("%d x %.2f = %.2f\n", item.Quantity, item.Price, lineTotal))

		for _, child := range item.Children {
			childTotal := float64(child.Quantity) * child.Price
			p.Write(fmt.Sprintf("  %s\n", child.Name))
			if child.Price > 0 {
				p.Write(fmt.Sprintf("  %d x %.2f = %.2f\n", child.Quantity, child.Price, childTotal))
			}
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")

	p.SetAlign("right")
	p.Write(fmt.Sprintf("Subtotal: %.2f\n", data.SubTotal))
	p.Write(fmt.Sprintf("Tax: %.2f\n", data.Tax))
	p.SetSize(0, 1)
	p.Write(fmt.Sprintf("TOTAL: %.2f\n", data.Total))
	p.SetSize(0, 0)

	p.SetAlign("center")
	p.Write(fmt.Sprintf("Payment: %s\n", data.PaymentMode))
	p.Write("Thank You!\n")

	p.Feed(3)
	p.Cut()
}

func GetSampleOrderData() OrderData {
	return OrderData{
		InvoiceNo:    "TEST-001",
		Date:         "2023-10-27 10:30 AM",
		CustomerName: "Test Customer",
		TableNo:      "T-1",
		Items: []OrderItem{
			{Name: "Chicken Biryani", Quantity: 1, Price: 250},
			{Name: "Coke", Quantity: 2, Price: 50},
		},
		SubTotal:    350,
		Tax:         17.5,
		Total:       367.5,
		PaymentMode: "Cash",
	}
}
