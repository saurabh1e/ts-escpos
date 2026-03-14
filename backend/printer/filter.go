package printer

import "strings"

var blockedPrinterTerms = []string{
	"onenote",
	"print to pdf",
	"pdf",
	"xps",
	"fax",
	"document writer",
	"image writer",
	"pdfcreator",
	"cutepdf",
	"dopdf",
	"foxit pdf",
	"adobe pdf",
	"snagit",
	"paperport",
	"microsoft office document image writer",
}

var blockedPortTerms = []string{
	"file:",
	"prompt:",
	"nul:",
	"xpsport:",
	"pdf",
	"onenote",
}

func IsSupportedPrinter(name string, driverName string, portName string) bool {
	normalizedName := normalizePrinterValue(name)
	if normalizedName == "" {
		return false
	}

	normalizedDriverName := normalizePrinterValue(driverName)
	normalizedPortName := normalizePrinterValue(portName)

	if hasBlockedPrinterTerm(normalizedName) {
		return false
	}

	if hasBlockedPrinterTerm(normalizedDriverName) {
		return false
	}

	if hasBlockedPortTerm(normalizedPortName) {
		return false
	}

	return true
}

func normalizePrinterValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hasBlockedPrinterTerm(value string) bool {
	return containsAnyTerm(value, blockedPrinterTerms)
}

func hasBlockedPortTerm(value string) bool {
	return containsAnyTerm(value, blockedPortTerms)
}

func containsAnyTerm(value string, terms []string) bool {
	if value == "" {
		return false
	}

	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}

	return false
}
