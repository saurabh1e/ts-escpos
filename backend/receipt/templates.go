package receipt

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
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
	ExternalID        interface{}    `json:"externalId"`
	ExternalOrderID   interface{}    `json:"externalOrderId"`
	Pax               interface{}    `json:"pax"`
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
	SetReverse(enabled bool)
	SetDoubleStrike(enabled bool)
	SetSize(width, height uint8)
	Write(data string)
	Feed(n uint8)
	Cut()
	PrintQRCode(data string)
	PrintImage(filePath string)
}

func getInvoiceNoStr(v interface{}) string {
	if v == nil {
		return ""
	}

	switch value := v.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ""
		}
		if value == math.Trunc(value) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		floatValue := float64(value)
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return ""
		}
		if floatValue == math.Trunc(floatValue) {
			return strconv.FormatInt(int64(floatValue), 10)
		}
		return strconv.FormatFloat(floatValue, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int8:
		return strconv.FormatInt(int64(value), 10)
	case int16:
		return strconv.FormatInt(int64(value), 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	}
	return fmt.Sprintf("%v", v)
}

func (d OrderData) GetInvoiceNo() string {
	return getInvoiceNoStr(d.InvoiceNo)
}

func (d OrderData) GetExternalID() string {
	val := strings.TrimSpace(getInvoiceNoStr(d.ExternalID))
	if val != "" {
		return val
	}
	return strings.TrimSpace(getInvoiceNoStr(d.ExternalOrderID))
}

func (d OrderData) GetPax() string {
	return strings.TrimSpace(getInvoiceNoStr(d.Pax))
}

func (d OrderData) GetStoreID() string {
	if d.StoreInfo.StoreCode != "" {
		return d.StoreInfo.StoreCode
	}
	if d.StoreInfo.DisplayName != "" {
		return d.StoreInfo.DisplayName
	}
	return d.StoreInfo.Name
}

func (d OrderData) GetKOTNumber() *int {
	for _, item := range d.Items {
		if kotNumber, hasKOTNumber := findKOTNumber(item); hasKOTNumber {
			return &kotNumber
		}
	}
	return nil
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

func wrapText(value string, width int) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if width < 1 {
		return []string{trimmed}
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return nil
	}

	lines := make([]string, 0, len(words))
	currentLine := ""

	for _, word := range words {
		wordRunes := []rune(word)
		for len(wordRunes) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			lines = append(lines, string(wordRunes[:width]))
			wordRunes = wordRunes[width:]
		}
		word = string(wordRunes)
		if word == "" {
			continue
		}

		if currentLine == "" {
			currentLine = word
			continue
		}

		if len([]rune(currentLine))+1+len([]rune(word)) <= width {
			currentLine += " " + word
			continue
		}

		lines = append(lines, currentLine)
		currentLine = word
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

func truncateName(name string, maxRunes int) string {
	runes := []rune(name)
	if len(runes) <= maxRunes {
		return name
	}
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "…"
}

func formatChildItemLine(child OrderItem, width int) string {
	prefix := "  + "
	ratePart := ""
	if child.UnitPriceValue() > 0 {
		ratePart = fmt.Sprintf(" (%.2f)", child.UnitPriceValue())
	}
	qtyPart := ""
	if child.Quantity > 1 {
		qtyPart = fmt.Sprintf(" (x%s)", formatQuantity(child.Quantity))
	}
	suffix := ratePart + qtyPart
	maxNameLen := width - len([]rune(prefix)) - len([]rune(suffix))
	if maxNameLen < 1 {
		maxNameLen = 1
	}
	name := truncateName(child.DisplayName(), maxNameLen)
	return prefix + name + suffix
}

func writeWrappedLine(p Printer, prefix string, value string, width int) {
	availableWidth := width - len([]rune(prefix))
	lines := wrapText(value, availableWidth)
	if len(lines) == 0 {
		p.Write(prefix + "\n")
		return
	}

	p.Write(prefix + lines[0] + "\n")
	if len(lines) == 1 {
		return
	}

	indent := strings.Repeat(" ", len([]rune(prefix)))
	for _, line := range lines[1:] {
		p.Write(indent + line + "\n")
	}
}

func writeTextBlock(p Printer, value string, width int) {
	for _, part := range splitNonEmptyLines(value) {
		lines := wrapText(part, width)
		if len(lines) == 0 {
			p.Write("\n")
			continue
		}

		for _, line := range lines {
			p.Write(line + "\n")
		}
	}
}

func writeKOTItemLine(p Printer, quantity string, name string, isMultiple bool, width int) {
	qtyLabel := quantity + "x "
	availableWidth := width - len([]rune(qtyLabel))
	lines := wrapText(name, availableWidth)
	if len(lines) == 0 {
		lines = []string{""}
	}

	p.SetBold(true)
	if isMultiple {
		p.SetSize(0, 1)
		p.SetReverse(true)
		p.Write(" " + quantity + " ")
		p.SetReverse(false)
		p.SetSize(0, 0)
		p.Write(" " + lines[0] + "\n")
	} else {
		p.SetSize(0, 1)
		p.Write(quantity)
		p.SetSize(0, 0)
		p.Write("x " + lines[0] + "\n")
	}
	p.SetBold(false)

	indent := strings.Repeat(" ", len([]rune(qtyLabel)))
	for _, line := range lines[1:] {
		p.Write(indent + line + "\n")
	}
}

func findKOTNumber(item OrderItem) (int, bool) {
	if item.KOTNumber > 0 {
		return item.KOTNumber, true
	}

	for _, child := range item.Children {
		if kotNumber, hasKOTNumber := findKOTNumber(child); hasKOTNumber {
			return kotNumber, true
		}
	}

	return 0, false
}

func renderExternalIDHeader(p Printer, externalID string, width int) {
	trimmedExternalID := strings.TrimSpace(externalID)
	if trimmedExternalID == "" {
		return
	}

	externalIDRunes := []rune(trimmedExternalID)
	highlightFrom := len(externalIDRunes) - 4
	if highlightFrom < 0 {
		highlightFrom = 0
	}

	normalPart := string(externalIDRunes[:highlightFrom])
	highlightPart := string(externalIDRunes[highlightFrom:])

	p.SetAlign("center")
	p.SetBold(false)
	p.SetReverse(false)
	p.SetDoubleStrike(false)
	p.SetSize(1, 1)

	highlightWidth := len([]rune(highlightPart)) + 2
	normalLines := wrapText(normalPart, width)
	if len(normalLines) == 0 {
		normalLines = nil
	}

	if len(normalLines) > 0 {
		lastLineIndex := len(normalLines) - 1
		for _, line := range normalLines[:lastLineIndex] {
			p.Write(line + "\n")
		}

		lastLine := normalLines[lastLineIndex]
		if len([]rune(lastLine))+highlightWidth <= width {
			p.Write(lastLine)
		} else {
			p.Write(lastLine + "\n")
		}
	}

	p.SetBold(true)
	p.SetReverse(true)
	p.Write(" " + highlightPart + " ")
	p.SetReverse(false)
	p.SetBold(false)
	p.SetSize(0, 0)
	p.SetDoubleStrike(false)
	p.Write("\n")
}

func RenderKOT(p Printer, data OrderData, size string, isDuplicate bool) {
	width := 32
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetFont("A")
	renderDuplicateHeader(p, isDuplicate)
	p.SetAlign("center")
	renderExternalIDHeader(p, data.GetExternalID(), width)
	p.SetDoubleStrike(true)
	p.SetBold(true)
	p.SetSize(1, 1)
	p.Write("KOT\n")
	p.SetSize(0, 0)
	p.SetBold(false)
	p.SetDoubleStrike(false)

	if data.StoreInfo.FirmName != "" {
		p.SetBold(true)
		writeTextBlock(p, data.StoreInfo.FirmName, width)
		p.SetBold(false)
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.SetAlign("left")
	if data.DisplayOptions.ShowOrderNumber {
		orderLineParts := make([]string, 0, 2)
		invoiceNo := strings.TrimSpace(getInvoiceNoStr(data.InvoiceNo))
		if invoiceNo != "" {
			orderLineParts = append(orderLineParts, "Order #: "+invoiceNo)
		}
		pax := data.GetPax()
		if pax != "" {
			orderLineParts = append(orderLineParts, "Pax: "+pax)
		}
		if len(orderLineParts) > 0 {
			p.Write(strings.Join(orderLineParts, "  ") + "\n")
		}
	}
	if data.TableNo != "" {
		p.SetBold(true)
		p.Write(fmt.Sprintf("Table: %s", data.TableNo))
		p.SetBold(false)
		if data.OrderType != "" {
			p.Write(fmt.Sprintf(" (%s)", data.OrderType))
		}
		p.Write("\n")
	}
	if data.OrderType != "" {
		p.Write(fmt.Sprintf("Type: %s\n", data.OrderType))
	}

	if data.DisplayOptions.ShowCustomerName && data.CustomerName != "" {
		p.Write(fmt.Sprintf("Customer: %s\n", data.CustomerName))
	}
	if data.CashierName != "" {
		writeWrappedLine(p, "Punched By: ", data.CashierName, width)
	}
	p.Write(fmt.Sprintf("Date: %s\n", data.Date))
	p.Write(strings.Repeat("-", width) + "\n")
	p.SetBold(true)
	p.Write("QTY x ITEM\n")
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	for _, item := range data.Items {
		writeKOTItemLine(p, formatQuantity(item.Quantity), item.DisplayName(), item.Quantity > 1, width)

		if item.Instructions() != "" {
			writeWrappedLine(p, "    Note: ", item.Instructions(), width)
		}

		for _, child := range item.Children {
			childLine := "  + " + child.DisplayName()
			if child.Quantity > 1 {
				childLine += fmt.Sprintf(" (x%s)", formatQuantity(child.Quantity))
			}
			p.Write(childLine + "\n")
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.Feed(3)
	p.Cut()
}

func RenderBill(p Printer, data OrderData, size string, isDuplicate bool) {
	width := 32
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetFont("A")
	renderDuplicateHeader(p, isDuplicate)
	p.SetAlign("center")
	renderExternalIDHeader(p, data.GetExternalID(), width)
	p.SetDoubleStrike(true)
	p.SetBold(true)
	p.Write("TAX INVOICE\n")
	p.SetBold(false)
	p.SetDoubleStrike(false)

	if data.StoreInfo.FirmName != "" {
		p.SetBold(true)
		p.SetSize(0, 0)
		writeTextBlock(p, data.StoreInfo.FirmName, width)
		p.SetBold(false)
	} else if data.StoreInfo.Name != "" {
		p.SetBold(true)
		writeTextBlock(p, data.StoreInfo.Name, width)
		p.SetBold(false)
	}

	if data.StoreInfo.Location != "" {
		writeTextBlock(p, data.StoreInfo.Location, width)
	}
	if data.StoreInfo.Address != "" {
		writeTextBlock(p, data.StoreInfo.Address, width)
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
		writeTextBlock(p, strings.Join(locParts, ", "), width)
	}
	if data.StoreInfo.Mobile != "" {
		p.Write("Mobile " + data.StoreInfo.Mobile + "\n")
	}
	if data.HeaderText != "" {
		writeTextBlock(p, data.HeaderText, width)
	}
	if data.StoreInfo.GST != "" {
		p.Write("GSTIN: " + data.StoreInfo.GST + "\n")
	}
	if data.StoreInfo.FSSAI != "" {
		p.Write("FSSAI: " + data.StoreInfo.FSSAI + "\n")
	}
	if data.StoreInfo.CIN != "" {
		p.Write("LLPIN/CIN: " + data.StoreInfo.CIN + "\n")
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
	if data.TableNo != "" {
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
		p.Write(truncateName(item.DisplayName(), width) + "\n")
		p.SetBold(false)

		for _, child := range item.Children {
			p.Write(formatChildItemLine(child, width) + "\n")
		}

		qtyStr := formatQuantity(item.Quantity)
		rateStr := fmt.Sprintf("%.2f", item.UnitPriceValue())
		totalStr := fmt.Sprintf("%.2f", item.FinalAmountValue())
		p.Write(fmt.Sprintf(lineFmt, "", qtyStr, rateStr, totalStr))

		if item.Instructions() != "" {
			p.Write(fmt.Sprintf("  Note: %s\n", item.Instructions()))
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

func renderDuplicateHeader(p Printer, isDuplicate bool) {
	if !isDuplicate {
		return
	}

	p.SetAlign("center")
	p.SetBold(true)
	p.SetSize(1, 1)
	p.Write("DUPLICATE\n")
	p.SetSize(0, 0)
	p.SetBold(false)
	p.Write("\n")
}
