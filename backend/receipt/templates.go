package receipt

import (
	"encoding/json"
	"fmt"
	"math"
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
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	ProductName         string            `json:"productName"`
	SKU                 string            `json:"sku"`
	Barcode             string            `json:"barcode"`
	Quantity            float64           `json:"quantity"`
	UnitPrice           float64           `json:"unitPrice"`
	LineTotal           float64           `json:"lineTotal"`
	DiscountAmount      float64           `json:"discountAmount"`
	TaxAmount           float64           `json:"taxAmount"`
	FinalAmount         float64           `json:"finalAmount"`
	ItemType            string            `json:"itemType"`
	Status              string            `json:"status"`
	KOTNumber           int               `json:"kotNumber"`
	SequenceNumber      int               `json:"sequenceNumber"`
	SpecialInstructions string            `json:"specialInstructions"`
	Children            OrderItemChildren `json:"children"`

	Price    float64 `json:"price"`
	Sku      string  `json:"-"`
	ItemNote string  `json:"itemNote"`
	Variant  string  `json:"variant"`
}

type TaxItem struct {
	Name   string  `json:"name"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
}

type ChargeItem struct {
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	TaxAmount float64 `json:"taxAmount"`
}

type DiscountItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type PaymentItem struct {
	Mode   string  `json:"mode"`
	Amount float64 `json:"amount"`
}

type StoreInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	StoreCode   string `json:"storeCode"`
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	PinCode     string `json:"pinCode"`
	Location    string `json:"location"`
	FirmName    string `json:"firmName"`
	Mobile      string `json:"mobile"`
	Email       string `json:"email"`
	GST         string `json:"gst"`
	FSSAI       string `json:"fssai"`
	CIN         string `json:"cin"`
}

type DisplayOptions struct {
	ShowTaxBreakdown      bool   `json:"showTaxBreakdown"`
	ShowDiscountBreakdown bool   `json:"showDiscountBreakdown"`
	ShowPaymentDetails    bool   `json:"showPaymentDetails"`
	ShowCustomerInfo      bool   `json:"showCustomerInfo"`
	ShowBarcode           bool   `json:"showBarcode"`
	ShowQRCode            bool   `json:"showQRCode"`
	QrCodeData            string `json:"qrCodeData"`
	ShowTableInfo         bool   `json:"showTableInfo"`
	ShowCustomerName      bool   `json:"showCustomerName"`
	ShowOrderNumber       bool   `json:"showOrderNumber"`
	ShowPreparationTime   bool   `json:"showPreparationTime"`
	GroupByCategory       bool   `json:"groupByCategory"`
}

type OrderData struct {
	InvoiceNo         interface{}    `json:"invoiceNo"`
	TableNo           string         `json:"tableNo"`
	CustomerName      string         `json:"customerName"`
	CustomerContact   string         `json:"customerContact"`
	CustomerAddress   string         `json:"customerAddress"`
	CustomerGST       string         `json:"customerGST"`
	Date              string         `json:"date"`
	Items             []OrderItem    `json:"items"`
	SubTotal          float64        `json:"subTotal"`
	Tax               float64        `json:"tax"`
	Total             float64        `json:"total"`
	PaymentMode       string         `json:"paymentMode"`
	OrderType         string         `json:"orderType"`
	OrderSource       string         `json:"orderSource"`
	CashierName       string         `json:"cashierName"`
	StoreInfo         StoreInfo      `json:"storeInfo"`
	HeaderText        string         `json:"headerText"`
	FooterText        string         `json:"footerText"`
	TaxBreakdown      []TaxItem      `json:"taxBreakdown"`
	DiscountBreakdown []DiscountItem `json:"discountBreakdown"`
	Charges           []ChargeItem   `json:"charges"`
	Payments          []PaymentItem  `json:"payments"`
	DisplayOptions    DisplayOptions `json:"displayOptions"`
}

type Printer interface {
	Init()
	SetAlign(align string)
	SetFont(font string)
	SetBold(bold bool)
	SetDoubleStrike(enabled bool)
	SetSize(width, height uint8)
	Write(data string)
	Feed(n uint8)
	Cut()
	PrintQRCode(data string)
	PrintImage(filePath string)
}

func getInvoiceNoStr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (d OrderData) GetInvoiceNo() string {
	return getInvoiceNoStr(d.InvoiceNo)
}

func (d OrderData) InvoiceDisplayNo() string {
	invoiceNo := d.GetInvoiceNo()
	if d.StoreInfo.StoreCode != "" && invoiceNo != "" {
		return d.StoreInfo.StoreCode + "/" + invoiceNo
	}
	return invoiceNo
}

func formatQuantity(quantity float64) string {
	if math.Mod(quantity, 1) == 0 {
		return fmt.Sprintf("%.0f", quantity)
	}
	return fmt.Sprintf("%.2f", quantity)
}

func (i OrderItem) DisplayName() string {
	if i.ProductName != "" {
		return i.ProductName
	}
	return i.Name
}

func (i OrderItem) SKUValue() string {
	if i.SKU != "" {
		return i.SKU
	}
	return i.Sku
}

func (i OrderItem) UnitPriceValue() float64 {
	if i.UnitPrice > 0 {
		return i.UnitPrice
	}
	return i.Price
}

func (i OrderItem) LineTotalValue() float64 {
	if i.LineTotal > 0 {
		return i.LineTotal
	}
	quantity := i.Quantity
	if quantity == 0 {
		quantity = 1
	}
	return quantity * i.UnitPriceValue()
}

func (i OrderItem) FinalAmountValue() float64 {
	if i.FinalAmount > 0 {
		return i.FinalAmount
	}
	return i.LineTotalValue()
}

func (i OrderItem) Instructions() string {
	if i.SpecialInstructions != "" {
		return i.SpecialInstructions
	}
	return i.ItemNote
}

func splitNonEmptyLines(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		line := strings.TrimSpace(part)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func RenderKOT(p Printer, data OrderData, size string) {
	width := 32
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetDoubleStrike(true)
	p.SetAlign("center")
	p.SetBold(true)
	p.SetSize(1, 1)
	p.Write("KOT\n")
	p.SetSize(0, 0)
	p.SetBold(false)

	if data.StoreInfo.FirmName != "" {
		p.SetBold(true)
		p.Write(data.StoreInfo.FirmName + "\n")
		p.SetBold(false)
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.SetAlign("left")
	if data.DisplayOptions.ShowOrderNumber {
		p.Write(fmt.Sprintf("Order #: %s\n", getInvoiceNoStr(data.InvoiceNo)))
	}
	if data.DisplayOptions.ShowTableInfo && data.TableNo != "" {
		p.SetBold(true)
		p.Write(fmt.Sprintf("Table: %s", data.TableNo))
		p.SetBold(false)
		if data.OrderType != "" {
			p.Write(fmt.Sprintf(" (%s)", data.OrderType))
		}
		p.Write("\n")
	} else if data.OrderType != "" {
		p.Write(fmt.Sprintf("Type: %s\n", data.OrderType))
	}

	if data.DisplayOptions.ShowCustomerName && data.CustomerName != "" {
		p.Write(fmt.Sprintf("Customer: %s\n", data.CustomerName))
	}
	p.Write(fmt.Sprintf("Date: %s\n", data.Date))
	p.Write(strings.Repeat("-", width) + "\n")
	p.SetBold(true)
	p.Write(fmt.Sprintf("%-4s %s\n", "Qty", "Item"))
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	for _, item := range data.Items {
		p.SetBold(true)
		p.Write(fmt.Sprintf("%-4s %s\n", formatQuantity(item.Quantity), item.DisplayName()))
		p.SetBold(false)

		if item.Instructions() != "" {
			p.Write(fmt.Sprintf("     Note: %s\n", item.Instructions()))
		}

		for _, child := range item.Children {
			p.Write(fmt.Sprintf("     + %-4s %s\n", formatQuantity(child.Quantity), child.DisplayName()))
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.Feed(3)
	p.Cut()
}

func RenderBill(p Printer, data OrderData, size string) {
	width := 32
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetDoubleStrike(true)
	p.SetAlign("center")
	p.SetBold(true)
	p.Write("TAX INVOICE\n")
	p.SetBold(false)

	if data.StoreInfo.FirmName != "" {
		p.SetBold(true)
		p.SetSize(0, 0)
		p.Write(data.StoreInfo.FirmName + "\n")
		p.SetBold(false)
	} else if data.StoreInfo.Name != "" {
		p.SetBold(true)
		p.Write(data.StoreInfo.Name + "\n")
		p.SetBold(false)
	}

	if data.StoreInfo.Location != "" {
		p.Write(data.StoreInfo.Location + "\n")
	}
	if data.StoreInfo.Address != "" {
		p.Write(data.StoreInfo.Address + "\n")
	}

	locParts := []string{}
	if data.StoreInfo.City != "" {
		locParts = append(locParts, data.StoreInfo.City)
	}
	if data.StoreInfo.State != "" {
		locParts = append(locParts, data.StoreInfo.State)
	}
	if data.StoreInfo.PinCode != "" {
		locParts = append(locParts, data.StoreInfo.PinCode)
	}
	if len(locParts) > 0 {
		p.Write(strings.Join(locParts, ", ") + "\n")
	}
	if data.StoreInfo.Mobile != "" {
		p.Write("Mobile " + data.StoreInfo.Mobile + "\n")
	}
	if data.StoreInfo.GST != "" {
		p.Write("GSTIN: " + data.StoreInfo.GST + "\n")
	}
	if data.StoreInfo.FSSAI != "" {
		p.Write("FSSAI: " + data.StoreInfo.FSSAI + "\n")
	}
	if data.StoreInfo.CIN != "" {
		p.Write("CIN: " + data.StoreInfo.CIN + "\n")
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.SetAlign("left")
	p.Write(fmt.Sprintf("Date: %s\n", data.Date))
	p.Write(fmt.Sprintf("Invoice: %s\n", data.InvoiceDisplayNo()))
	if data.OrderSource != "" {
		p.Write(fmt.Sprintf("Source: %s\n", data.OrderSource))
	}
	if data.OrderType != "" {
		p.Write(fmt.Sprintf("Order Type: %s\n", data.OrderType))
	}
	if data.DisplayOptions.ShowTableInfo && data.TableNo != "" {
		p.Write(fmt.Sprintf("Table: %s\n", data.TableNo))
	}

	customerLine := []string{}
	if data.DisplayOptions.ShowCustomerName && data.CustomerName != "" {
		customerLine = append(customerLine, "Customer: "+data.CustomerName)
	}
	if data.DisplayOptions.ShowCustomerInfo && data.CustomerContact != "" {
		customerLine = append(customerLine, "Phone: "+data.CustomerContact)
	}
	if len(customerLine) > 0 {
		p.Write(strings.Join(customerLine, ", ") + "\n")
	}
	if data.DisplayOptions.ShowCustomerInfo && data.CustomerGST != "" {
		p.Write("GSTIN: " + data.CustomerGST + "\n")
	}
	if data.DisplayOptions.ShowCustomerInfo && data.CustomerAddress != "" {
		p.Write("Address: " + data.CustomerAddress + "\n")
	}

	p.Write(strings.Repeat("-", width) + "\n")

	colItem := 11
	colQty := 4
	colRate := 6
	colAmt := 8
	if width == 48 {
		colItem = 18
		colQty = 5
		colRate = 10
		colAmt = 12
	}

	p.SetBold(true)
	headerFmt := fmt.Sprintf("%%-%ds %%%ds %%%ds %%%ds\n", colItem, colQty, colRate, colAmt)
	p.Write(fmt.Sprintf(headerFmt, "ITEM", "QTY", "RATE", "AMOUNT"))
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	lineFmt := fmt.Sprintf("%%-%ds %%%ds %%%ds %%%ds\n", colItem, colQty, colRate, colAmt)

	for _, item := range data.Items {
		p.SetBold(true)
		p.Write(item.DisplayName() + "\n")
		p.SetBold(false)
		qtyStr := formatQuantity(item.Quantity)
		rateStr := fmt.Sprintf("%.2f", item.UnitPriceValue())
		totalStr := fmt.Sprintf("%.2f", item.FinalAmountValue())
		p.Write(fmt.Sprintf(lineFmt, "", qtyStr, rateStr, totalStr))

		if item.Instructions() != "" {
			p.Write(fmt.Sprintf("  Note: %s\n", item.Instructions()))
		}

		for _, child := range item.Children {
			p.Write("  + " + child.DisplayName() + "\n")
			if child.UnitPriceValue() > 0 || child.FinalAmountValue() > 0 {
				p.Write(fmt.Sprintf(lineFmt, "", formatQuantity(child.Quantity), fmt.Sprintf("%.2f", child.UnitPriceValue()), fmt.Sprintf("%.2f", child.FinalAmountValue())))
			}
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.SetAlign("right")
	p.Write(fmt.Sprintf("Sub Total: %.2f\n", data.SubTotal))

	if data.DisplayOptions.ShowTaxBreakdown && len(data.TaxBreakdown) > 0 {
		for _, t := range data.TaxBreakdown {
			if t.Rate > 0 {
				p.Write(fmt.Sprintf("%s@%.2f%%: %.2f\n", t.Name, t.Rate, t.Amount))
				continue
			}
			p.Write(fmt.Sprintf("%s: %.2f\n", t.Name, t.Amount))
		}
	}

	if data.DisplayOptions.ShowDiscountBreakdown && len(data.DiscountBreakdown) > 0 {
		for _, d := range data.DiscountBreakdown {
			p.Write(fmt.Sprintf("%s: -%.2f\n", d.Name, d.Amount))
		}
	}

	for _, c := range data.Charges {
		p.Write(fmt.Sprintf("%s: %.2f\n", c.Name, c.Amount))
	}

	p.SetBold(true)
	p.Write(fmt.Sprintf("Total: %.2f\n", data.Total))
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	p.SetAlign("left")
	if data.DisplayOptions.ShowPaymentDetails && len(data.Payments) > 0 {
		for _, pay := range data.Payments {
			p.Write(fmt.Sprintf("%s: %.2f\n", pay.Mode, pay.Amount))
		}
	} else if data.DisplayOptions.ShowPaymentDetails && data.PaymentMode != "" {
		p.Write(fmt.Sprintf("%s: %.2f\n", data.PaymentMode, data.Total))
	}

	if data.CashierName != "" {
		p.Write(fmt.Sprintf("Cashier Name: %s\n", data.CashierName))
	}

	p.SetAlign("center")
	p.Write("\n")
	if data.FooterText != "" {
		for _, line := range splitNonEmptyLines(data.FooterText) {
			p.Write(line + "\n")
		}
	} else {
		p.Write("Thank you! Visit Again.\n")
	}

	if data.DisplayOptions.ShowQRCode && data.DisplayOptions.QrCodeData != "" {
		p.Write("\n")
		p.PrintQRCode(data.DisplayOptions.QrCodeData)
	}

	p.Feed(4)
	p.Cut()
}

func GetSampleOrderData() OrderData {
	return OrderData{
		InvoiceNo:       "N00472023085828",
		Date:            "03/08/2023, 7:24:33",
		CustomerName:    "Naturals",
		CustomerContact: "0000000000",
		CustomerAddress: "Connaught Place, New Delhi 110001",
		CustomerGST:     "07ABCDE1234F1Z5",
		TableNo:         "A-12",
		OrderType:       "Take Away",
		OrderSource:     "POS",
		CashierName:     "Pankaj Kshirsagar",
		Items: []OrderItem{
			{
				ID:                  "1",
				Name:                "Anjeer Ice Cream",
				ProductName:         "Anjeer Ice Cream",
				Quantity:            1,
				UnitPrice:           72.04,
				LineTotal:           72.04,
				FinalAmount:         72.04,
				SKU:                 "IC-ANJ-001",
				SpecialInstructions: "Single scoop cup",
				TaxAmount:           12.96,
				Children: OrderItemChildren{
					{
						ID:          "1-1",
						Name:        "Extra Hot Fudge",
						ProductName: "Extra Hot Fudge",
						Quantity:    1,
						UnitPrice:   12.96,
						LineTotal:   12.96,
						FinalAmount: 12.96,
					},
				},
			},
			{
				ID:          "2",
				Name:        "Tender Coconut",
				ProductName: "Tender Coconut",
				Quantity:    1,
				UnitPrice:   85.00,
				LineTotal:   85.00,
				FinalAmount: 85.00,
				SKU:         "IC-TC-002",
			},
		},
		SubTotal:    157.04,
		Tax:         28.27,
		Total:       185.31,
		PaymentMode: "Cash",
		StoreInfo: StoreInfo{
			Name:        "Naturals Ice Cream",
			DisplayName: "Naturals",
			StoreCode:   "NAT-001",
			FirmName:    "Kamaths Natural Retail Private Limited",
			GST:         "07AAFLK3562K1ZB",
			Address:     "Connaught Place, Block, Outer Circle, Connaught Place",
			City:        "New Delhi",
			State:       "Delhi",
			PinCode:     "110001",
			Location:    "Connaught Place",
			Mobile:      "7738091984",
			Email:       "info@naturalicecreams.in",
			FSSAI:       "13319009001023",
			CIN:         "U52390MH2013P1C248637",
		},
		HeaderText: "Scooping your e-bill...",
		FooterText: "Whatsapp \" \" on 7738095605 to order online.\nEmail: info@naturalicecreams.in\nE&OE Thanks and visit again.",
		DisplayOptions: DisplayOptions{
			ShowTaxBreakdown:      true,
			ShowDiscountBreakdown: true,
			ShowPaymentDetails:    true,
			ShowCustomerInfo:      true,
			ShowBarcode:           true,
			ShowQRCode:            false,
			QrCodeData:            "",
			ShowTableInfo:         true,
			ShowCustomerName:      true,
			ShowOrderNumber:       true,
			ShowPreparationTime:   false,
			GroupByCategory:       false,
		},
		TaxBreakdown: []TaxItem{
			{Name: "SGST/UGST", Rate: 9.0, Amount: 14.14},
			{Name: "CGST", Rate: 9.0, Amount: 14.13},
		},
		DiscountBreakdown: []DiscountItem{},
		Charges:           []ChargeItem{},
		Payments: []PaymentItem{
			{Mode: "Cash", Amount: 185.31},
		},
	}
}
