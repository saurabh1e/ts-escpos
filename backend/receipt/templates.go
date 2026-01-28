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
	Name           string            `json:"name"`
	Quantity       int               `json:"quantity"`
	Price          float64           `json:"price"`
	Sku            string            `json:"sku"`
	ItemNote       string            `json:"itemNote"`
	Variant        string            `json:"variant"`
	Children       OrderItemChildren `json:"children"`
	TaxAmount      float64           `json:"taxAmount"`
	DiscountAmount float64           `json:"discountAmount"`
}

type TaxItem struct {
	Name   string  `json:"name"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
}

type ChargeItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
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
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	BrandName      string `json:"brandName"`
	StoreGroupName string `json:"storeGroupName"`
	HeaderText     string `json:"headerText"`
	FooterText     string `json:"footerText"`
	ShowLogo       bool   `json:"showLogo"`
	LogoURL        string `json:"logoURL"`
	GST            string `json:"gst"`
	Address        string `json:"address"`
	City           string `json:"city"`
	ContactNumber  string `json:"contactNumber"`
	Email          string `json:"email"`
	Policy         string `json:"policy"`
	FSSAIState     string `json:"fssaiState"`
	FSSAICentral   string `json:"fssaiCentral"`
	CIN            string `json:"cin"`
	LLPIN          string `json:"llpin"`
	Website        string `json:"website"`
}

type DisplayOptions struct {
	ShowTaxBreakdown      bool   `json:"showTaxBreakdown"`
	ShowDiscountBreakdown bool   `json:"showDiscountBreakdown"`
	ShowPaymentDetails    bool   `json:"showPaymentDetails"`
	ShowCustomerInfo      bool   `json:"showCustomerInfo"`
	ShowBarcode           bool   `json:"showBarcode"`
	ShowQRCode            bool   `json:"showQRCode"`
	QrCodeData            string `json:"qrCodeData"`

	// KOT fields
	ShowTableInfo       bool `json:"showTableInfo"`
	ShowCustomerName    bool `json:"showCustomerName"`
	ShowOrderNumber     bool `json:"showOrderNumber"`
	ShowPreparationTime bool `json:"showPreparationTime"`
	GroupByCategory     bool `json:"groupByCategory"`
}

type OrderData struct {
	InvoiceNo       interface{}    `json:"invoiceNo"` // string or int
	Date            string         `json:"date"`
	CustomerName    string         `json:"customerName"`
	CustomerContact string         `json:"customerContact"`
	TableNo         string         `json:"tableNo"`
	OrderType       string         `json:"orderType"` // For KOT
	OrderSource     string         `json:"orderSource"`
	CashierName     string         `json:"cashierName"`
	Items           []OrderItem    `json:"items"`
	SubTotal        float64        `json:"subTotal"`
	Tax             float64        `json:"tax"`
	Total           float64        `json:"total"`
	PaymentMode     string         `json:"paymentMode"`
	StoreInfo       StoreInfo      `json:"storeInfo"`
	DisplayOptions  DisplayOptions `json:"displayOptions"`

	TaxBreakdown      []TaxItem      `json:"taxBreakdown"`
	DiscountBreakdown []DiscountItem `json:"discountBreakdown"`
	Charges           []ChargeItem   `json:"charges"`
	Payments          []PaymentItem  `json:"payments"`
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

func RenderKOT(p Printer, data OrderData, size string) {
	width := 32 // Default 58mm
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetDoubleStrike(true)

	// HEADER
	p.SetAlign("center")
	p.SetBold(true)
	p.SetSize(1, 1) // Double height/width
	p.Write("KOT\n")
	p.SetSize(0, 0) // Normal
	p.SetBold(false)

	if data.StoreInfo.BrandName != "" {
		p.SetBold(true)
		p.Write(data.StoreInfo.BrandName + "\n")
		p.SetBold(false)
	}

	p.Write(strings.Repeat("-", width) + "\n")

	// KOT INFO
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

	// ITEMS HEADER
	// Qty Item
	p.SetBold(true)
	p.Write(fmt.Sprintf("%-4s %s\n", "Qty", "Item"))
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	// ITEMS
	for _, item := range data.Items {
		p.SetBold(true)
		p.Write(fmt.Sprintf("%-4d %s\n", item.Quantity, item.Name))
		p.SetBold(false)

		if item.Variant != "" {
			p.Write(fmt.Sprintf("     Var: %s\n", item.Variant))
		}
		if item.ItemNote != "" {
			p.Write(fmt.Sprintf("     Note: %s\n", item.ItemNote))
		}

		for _, child := range item.Children {
			p.Write(fmt.Sprintf("     + %-2d %s\n", child.Quantity, child.Name))
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")
	p.Feed(3)
	p.Cut()
}

func truncateString(str string, num int) string {
	if len(str) > num {
		return str[0:num]
	}
	return str
}

func RenderBill(p Printer, data OrderData, size string) {
	width := 32 // Default 58mm
	if size == "80mm" {
		width = 48
	}

	p.Init()
	p.SetDoubleStrike(true)
	p.SetAlign("center")

	// 1. HEADER
	// Logo
	if data.StoreInfo.ShowLogo && data.StoreInfo.LogoURL != "" {
		p.PrintImage(data.StoreInfo.LogoURL)
	}

	// Brand / Store Name
	if data.StoreInfo.BrandName != "" {
		p.SetBold(true)
		p.SetSize(1, 1) // Double Height/Width
		p.Write(data.StoreInfo.BrandName + "\n")
		p.SetSize(0, 0)
		p.SetBold(false)
	}
	if data.StoreInfo.DisplayName != "" {
		p.Write(data.StoreInfo.DisplayName + "\n")
	} else if data.StoreInfo.Name != "" {
		p.Write(data.StoreInfo.Name + "\n")
	}

	// Address & City
	if data.StoreInfo.Address != "" {
		p.Write(data.StoreInfo.Address + "\n")
	}
	if data.StoreInfo.City != "" {
		p.Write(data.StoreInfo.City + "\n")
	}
	// Contact Info
	if data.StoreInfo.ContactNumber != "" {
		p.Write("Phone: " + data.StoreInfo.ContactNumber + "\n")
	}
	if data.StoreInfo.Email != "" {
		p.Write("Email: " + data.StoreInfo.Email + "\n")
	}

	// Compliance IDs
	if data.StoreInfo.GST != "" {
		p.Write("GSTIN: " + data.StoreInfo.GST + "\n")
	}
	if data.StoreInfo.FSSAIState != "" {
		p.Write("FSSAI (State): " + data.StoreInfo.FSSAIState + "\n")
	}
	if data.StoreInfo.FSSAICentral != "" {
		p.Write("FSSAI (Central): " + data.StoreInfo.FSSAICentral + "\n")
	}
	if data.StoreInfo.CIN != "" {
		p.Write("CIN: " + data.StoreInfo.CIN + "\n")
	}
	if data.StoreInfo.LLPIN != "" {
		p.Write("LLPIN: " + data.StoreInfo.LLPIN + "\n")
	}

	p.Write("\n")
	p.SetBold(true)
	p.Write("TAX INVOICE\n")
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	// 2. TRANSACTION DETAILS
	p.SetAlign("left")
	p.Write(fmt.Sprintf("Invoice No: %s\n", getInvoiceNoStr(data.InvoiceNo)))
	p.Write(fmt.Sprintf("Date: %s\n", data.Date))

	if data.OrderSource != "" {
		p.Write(fmt.Sprintf("Source: %s\n", data.OrderSource))
	}
	if data.OrderType != "" {
		p.Write(fmt.Sprintf("Type: %s\n", data.OrderType))
	}
	if data.TableNo != "" {
		p.Write(fmt.Sprintf("Table: %s\n", data.TableNo))
	}

	// Customer Info
	if data.DisplayOptions.ShowCustomerInfo {
		if data.CustomerName != "" {
			p.Write(fmt.Sprintf("Customer: %s\n", data.CustomerName))
		}
		if data.CustomerContact != "" {
			p.Write(fmt.Sprintf("Phone: %s\n", data.CustomerContact))
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")

	// 3. ITEM DETAILS
	// Header
	var itemLen int
	var fmtStr string

	if width == 48 {
		// 80mm
		// Format: Item(22) Qty(4) Rate(9) Amount(10) | Spaces=3 => 48
		itemLen = 22
		fmtStr = "%-22s %4s %9s %10s\n"
	} else {
		// 58mm (32 chars)
		// Format: Item(10) Qt(4) Rt(8) Amt(8) | Spaces=2 => 32
		itemLen = 10
		fmtStr = "%-10s %4s %8s %8s\n"
	}

	p.SetBold(true)
	p.Write(fmt.Sprintf(fmtStr, "Item", "Qty", "Rate", "Total"))
	p.SetBold(false)
	p.Write(strings.Repeat("-", width) + "\n")

	// Define item formatter
	printItemLine := func(name string, qty int, price, total float64) {
		nameTrunc := truncateString(name, itemLen)
		qtyStr := fmt.Sprintf("%d", qty)
		priceStr := fmt.Sprintf("%.2f", price)
		totalStr := fmt.Sprintf("%.2f", total)
		p.Write(fmt.Sprintf(fmtStr, nameTrunc, qtyStr, priceStr, totalStr))
	}

	// Loop Items
	for _, item := range data.Items {
		lineTotal := float64(item.Quantity) * item.Price
		printItemLine(item.Name, item.Quantity, item.Price, lineTotal)

		// Variants
		if item.Variant != "" {
			p.Write(fmt.Sprintf("  Var: %s\n", item.Variant))
		}
		// Note
		if item.ItemNote != "" {
			p.Write(fmt.Sprintf("  Note: %s\n", item.ItemNote))
		}
		// Children
		for _, child := range item.Children {
			childTotal := float64(child.Quantity) * child.Price
			if child.Price > 0 {
				displayName := "  + " + child.Name
				printItemLine(displayName, child.Quantity, child.Price, childTotal)
			} else {
				p.Write(fmt.Sprintf("  + %s\n", child.Name))
			}
		}
	}

	p.Write(strings.Repeat("-", width) + "\n")

	// 4. TOTALS
	p.SetAlign("right")
	p.Write(fmt.Sprintf("Subtotal: %.2f\n", data.SubTotal))

	if data.DisplayOptions.ShowDiscountBreakdown && len(data.DiscountBreakdown) > 0 {
		for _, d := range data.DiscountBreakdown {
			p.Write(fmt.Sprintf("%s: -%.2f\n", d.Name, d.Amount))
		}
	}

	for _, c := range data.Charges {
		p.Write(fmt.Sprintf("%s: %.2f\n", c.Name, c.Amount))
	}

	// Tax Sum
	if data.Tax > 0 {
		p.Write(fmt.Sprintf("Total Tax: %.2f\n", data.Tax))
	}

	p.Write(strings.Repeat("-", width) + "\n")

	// Grand Total
	p.SetBold(true)
	p.SetSize(0, 1) // Double Height
	p.Write(fmt.Sprintf("GRAND TOTAL: %.2f\n", data.Total))
	p.SetSize(0, 0)
	p.SetBold(false)

	p.Write(strings.Repeat("-", width) + "\n")

	// 5. TAX BREAKDOWN
	if data.DisplayOptions.ShowTaxBreakdown && len(data.TaxBreakdown) > 0 {
		p.SetAlign("left")
		p.Write("Tax Details:\n")
		// Header for tax? No space. Just list.
		for _, t := range data.TaxBreakdown {
			// e.g., SGST @ 9.00% : 18.50
			p.Write(fmt.Sprintf(" %s @ %.2f%% : %.2f\n", t.Name, t.Rate, t.Amount))
		}
		p.Write(strings.Repeat("-", width) + "\n")
	}

	// 6. FOOTER INFO
	p.SetAlign("center")

	// Payment Info
	if data.DisplayOptions.ShowPaymentDetails {
		if len(data.Payments) > 0 {
			p.Write("Payment Mode:\n")
			for _, pay := range data.Payments {
				p.Write(fmt.Sprintf("%s: %.2f\n", pay.Mode, pay.Amount))
			}
		} else if data.DisplayOptions.ShowPaymentDetails && data.PaymentMode != "" {
			p.Write(fmt.Sprintf("Payment Mode: %s\n", data.PaymentMode))
		}
	}

	if data.CashierName != "" {
		p.Write(fmt.Sprintf("Cashier: %s\n", data.CashierName))
	}

	// Policy / Closing
	p.Write("\n")
	if data.StoreInfo.Policy != "" {
		p.Write(data.StoreInfo.Policy + "\n")
	} else if data.StoreInfo.FooterText != "" {
		p.Write(data.StoreInfo.FooterText + "\n")
	} else {
		p.Write("Thank you! Visit Again.\n")
	}

	if data.StoreInfo.Website != "" {
		p.Write("Visit: " + data.StoreInfo.Website + "\n")
	}

	// QR Code
	if data.DisplayOptions.ShowQRCode && data.DisplayOptions.QrCodeData != "" {
		p.Write("\n")
		p.PrintQRCode(data.DisplayOptions.QrCodeData)
	}

	p.Feed(4)
	p.Cut()
}

func GetSampleOrderData() OrderData {
	return OrderData{
		InvoiceNo:       "INV-2026-001",
		Date:            "28/01/2026, 01:30 PM",
		CustomerName:    "Saurabh Sharma",
		CustomerContact: "9876543210",
		TableNo:         "T-12",
		OrderType:       "Dine-In",
		OrderSource:     "POS",
		CashierName:     "Rahul",
		Items: []OrderItem{
			{
				Name:           "Paneer Tikka Masala",
				Quantity:       1,
				Price:          280.00,
				Sku:            "SKU_101",
				ItemNote:       "Spicy",
				Variant:        "Full",
				TaxAmount:      14.00,
				DiscountAmount: 0.00,
				Children: []OrderItem{
					{Name: "Extra Gravy", Quantity: 1, Price: 20.00},
				},
			},
			{
				Name:     "Butter Naan",
				Quantity: 2,
				Price:    40.00,
				Variant:  "",
				Children: nil,
			},
			{
				Name:     "Veg Thali",
				Quantity: 1,
				Price:    350.00,
				Variant:  "Deluxe",
				Children: []OrderItem{
					{Name: "Roti", Quantity: 2, Price: 0},
					{Name: "Rice", Quantity: 1, Price: 0},
					{Name: "Sweet", Quantity: 1, Price: 0},
					{Name: "Extra Papad", Quantity: 1, Price: 10},
				},
			},
		},
		SubTotal:    740.00,
		Tax:         37.00,
		Total:       797.00,
		PaymentMode: "UPI",
		StoreInfo: StoreInfo{
			Name:           "Mumbai Branch",
			DisplayName:    "The Food Place - Mumbai",
			BrandName:      "The Food Place",
			StoreGroupName: "West Region",
			HeaderText:     "Welcome to The Food Place",
			FooterText:     "Visit again!",
			ShowLogo:       true,
			LogoURL:        "https://via.placeholder.com/150",
			GST:            "27ABCDE1234F1Z5",
			Address:        "Shop 12, Main Street, Andheri West",
			City:           "Mumbai, Maharashtra 400053",
			ContactNumber:  "022-12345678",
			Email:          "contact@thefoodplace.com",
			Policy:         "No refund, No exchange",
			FSSAIState:     "12345678901234",
			FSSAICentral:   "98765432109876",
			CIN:            "U12345MH2023PTC123456",
			LLPIN:          "A12345MH2023PLC123456",
			Website:        "https://thefoodplace.com",
		},
		DisplayOptions: DisplayOptions{
			ShowTaxBreakdown:      true,
			ShowDiscountBreakdown: true,
			ShowPaymentDetails:    true,
			ShowCustomerInfo:      true,
			ShowBarcode:           true,
			ShowQRCode:            true,
			QrCodeData:            "https://thefoodplace.com/feedback/INV-2026-001",
			ShowTableInfo:         true,
			ShowCustomerName:      true,
			ShowOrderNumber:       true,
			ShowPreparationTime:   true,
			GroupByCategory:       true,
		},
		TaxBreakdown: []TaxItem{
			{Name: "CGST", Rate: 9.0, Amount: 18.50},
			{Name: "SGST", Rate: 9.0, Amount: 18.50},
		},
		DiscountBreakdown: []DiscountItem{
			// Example discount
		},
		Charges: []ChargeItem{
			{Name: "Service Charge", Amount: 20.00},
		},
		Payments: []PaymentItem{
			{Mode: "UPI", Amount: 797.00},
		},
	}
}
