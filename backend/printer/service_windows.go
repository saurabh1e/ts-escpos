//go:build windows

package printer

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// Windows API constants and types
var (
	modwinspool          = windows.NewLazySystemDLL("winspool.drv")
	procOpenPrinter      = modwinspool.NewProc("OpenPrinterW")
	procClosePrinter     = modwinspool.NewProc("ClosePrinter")
	procSetPrinter       = modwinspool.NewProc("SetPrinterW") // Added
	procStartDocPrinter  = modwinspool.NewProc("StartDocPrinterW")
	procEndDocPrinter    = modwinspool.NewProc("EndDocPrinter")
	procStartPagePrinter = modwinspool.NewProc("StartPagePrinter")
	procEndPagePrinter   = modwinspool.NewProc("EndPagePrinter")
	procWritePrinter     = modwinspool.NewProc("WritePrinter")
	procEnumPrintersW    = modwinspool.NewProc("EnumPrintersW")
)

type DOC_INFO_1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}

type PRINTER_INFO_2 struct {
	pServerName         uintptr
	pPrinterName        uintptr
	pShareName          uintptr
	pPortName           uintptr
	pDriverName         uintptr
	pComment            uintptr
	pLocation           uintptr
	pDevMode            uintptr
	pSepFile            uintptr
	pPrintProcessor     uintptr
	pDatatype           uintptr
	pParameters         uintptr
	pSecurityDescriptor uintptr
	Attributes          uint32
	Priority            uint32
	DefaultPriority     uint32
	StartTime           uint32
	UntilTime           uint32
	Status              uint32
	cJobs               uint32
	AveragePPM          uint32
}

type PRINTER_DEFAULTS struct {
	pDatatype     *uint16
	pDevMode      uintptr
	DesiredAccess uint32
}

const (
	PRINTER_ACCESS_ADMINISTER = 0x00000004
	PRINTER_ACCESS_USE        = 0x00000008
	PRINTER_ALL_ACCESS        = 0x000F000C
)

func GetPrinters() ([]PrinterInfo, error) {
	// Log start of discovery (verbose, maybe only if needed, but "logs for everything" requested)
	// Since GetPrinters doesn't take context, we can't emit events easily here unless we change signature.
	// But the user logs are mostly about actions. Let's keep it simple or print to stdout which might be captured.
	// However, the prompt implies UI logs.
	// We can't change GetPrinters signature easily without breaking App.GetPrinters signature in app.go and Wails binding.
	// For now, I'll rely on fmt.Println which goes to stdout/terminal.
	fmt.Println("[Printer] Discovering printers via EnumPrintersW...")

	var bytesNeeded, countReturned uint32
	const level = 2
	const flags = 0x00000002 | 0x00000004 // PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS

	// First call to get size
	procEnumPrintersW.Call(
		uintptr(flags),
		0,
		uintptr(level),
		0,
		0,
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&countReturned)),
	)

	if bytesNeeded == 0 {
		return []PrinterInfo{}, nil
	}

	buffer := make([]byte, bytesNeeded)
	r1, _, err := procEnumPrintersW.Call(
		uintptr(flags),
		0,
		uintptr(level),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(bytesNeeded),
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&countReturned)),
	)

	if r1 == 0 {
		return nil, err
	}

	printers := make([]PrinterInfo, 0, countReturned)
	infoPtr := unsafe.Pointer(&buffer[0])

	for i := uint32(0); i < countReturned; i++ {
		p := (*PRINTER_INFO_2)(unsafe.Pointer(uintptr(infoPtr) + uintptr(i)*unsafe.Sizeof(PRINTER_INFO_2{})))

		name := ptrToString(p.pPrinterName)
		port := ptrToString(p.pPortName)

		// Status translation could be added here
		printers = append(printers, PrinterInfo{
			Name:      name,
			UniqueID:  name, // Windows printer names are unique enough locally
			WindowsID: port,
			Status:    getStatusString(p.Status),
		})
	}

	return printers, nil
}

func getStatusString(status uint32) string {
	if status == 0 {
		return "Ready"
	}

	var statuses []string

	// Check bits
	if status&0x00000001 != 0 {
		statuses = append(statuses, "Paused")
	}
	if status&0x00000002 != 0 {
		statuses = append(statuses, "Error")
	}
	if status&0x00000004 != 0 {
		statuses = append(statuses, "Pending Deletion")
	}
	if status&0x00000008 != 0 {
		statuses = append(statuses, "Paper Jam")
	}
	if status&0x00000010 != 0 {
		statuses = append(statuses, "Paper Out")
	}
	if status&0x00000020 != 0 {
		statuses = append(statuses, "Manual Feed")
	}
	if status&0x00000040 != 0 {
		statuses = append(statuses, "Paper Problem")
	}
	if status&0x00000080 != 0 {
		statuses = append(statuses, "Offline")
	}
	if status&0x00000100 != 0 {
		statuses = append(statuses, "IO Active")
	}
	if status&0x00000200 != 0 {
		statuses = append(statuses, "Busy")
	}
	if status&0x00000400 != 0 {
		statuses = append(statuses, "Printing")
	}
	if status&0x00000800 != 0 {
		statuses = append(statuses, "Output Bin Full")
	}
	if status&0x00001000 != 0 {
		statuses = append(statuses, "Not Available")
	}
	if status&0x00002000 != 0 {
		statuses = append(statuses, "Waiting")
	}
	if status&0x00004000 != 0 {
		statuses = append(statuses, "Processing")
	}
	if status&0x00008000 != 0 {
		statuses = append(statuses, "Initializing")
	}
	if status&0x00010000 != 0 {
		statuses = append(statuses, "Warming Up")
	}
	if status&0x00020000 != 0 {
		statuses = append(statuses, "Toner Low")
	}
	if status&0x00040000 != 0 {
		statuses = append(statuses, "No Toner")
	}
	if status&0x00080000 != 0 {
		statuses = append(statuses, "Page Punt")
	}
	if status&0x00100000 != 0 {
		statuses = append(statuses, "User Intervention")
	}
	if status&0x00200000 != 0 {
		statuses = append(statuses, "Out of Memory")
	}
	if status&0x00400000 != 0 {
		statuses = append(statuses, "Door Open")
	}
	if status&0x00800000 != 0 {
		statuses = append(statuses, "Server Unknown")
	}
	if status&0x01000000 != 0 {
		statuses = append(statuses, "Power Save")
	}

	if len(statuses) == 0 {
		return fmt.Sprintf("Status Code: %d", status)
	}

	// Join all statuses
	result := statuses[0]
	for i := 1; i < len(statuses); i++ {
		result += ", " + statuses[i]
	}
	return result
}

func ptrToString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr)))
}

func PrintRaw(ctx context.Context, printerName string, data []byte) error {
	logToFrontend(ctx, fmt.Sprintf("[PrintRaw] Starting job for '%s' (%d bytes)", printerName, len(data)))
	name, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		logToFrontend(ctx, fmt.Sprintf("[Printer] UTF16 conversion failed: %v", err))
		return err
	}

	var hPrinter syscall.Handle
	r1, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		logToFrontend(ctx, fmt.Sprintf("[Printer] OpenPrinter failed: %v", err))
		return fmt.Errorf("OpenPrinter failed: %v", err)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))
	logToFrontend(ctx, "[Printer] OpenPrinter success. Handle obtained.")

	docName, _ := syscall.UTF16PtrFromString("RAW Print Job")
	dataType, _ := syscall.UTF16PtrFromString("RAW")

	di := DOC_INFO_1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	r1, _, err = procStartDocPrinter.Call(
		uintptr(hPrinter),
		1,
		uintptr(unsafe.Pointer(&di)),
	)
	if r1 == 0 {
		logToFrontend(ctx, fmt.Sprintf("[Printer] StartDocPrinter failed: %v", err))
		return fmt.Errorf("StartDocPrinter failed: %v", err)
	}
	defer procEndDocPrinter.Call(uintptr(hPrinter))

	r1, _, err = procStartPagePrinter.Call(uintptr(hPrinter))
	if r1 == 0 {
		logToFrontend(ctx, fmt.Sprintf("[Printer] StartPagePrinter failed: %v", err))
		return fmt.Errorf("StartPagePrinter failed: %v", err)
	}
	defer procEndPagePrinter.Call(uintptr(hPrinter))

	var bytesWritten uint32
	r1, _, err = procWritePrinter.Call(
		uintptr(hPrinter),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&bytesWritten)),
	)
	if r1 == 0 {
		logToFrontend(ctx, fmt.Sprintf("[Printer] WritePrinter failed: %v", err))
		return fmt.Errorf("WritePrinter failed: %v", err)
	}

	if bytesWritten != uint32(len(data)) {
		logToFrontend(ctx, fmt.Sprintf("[Printer] Incomplete write: %d/%d bytes", bytesWritten, len(data)))
		return fmt.Errorf("incomplete write: %d/%d", bytesWritten, len(data))
	}

	logToFrontend(ctx, fmt.Sprintf("[Printer] WritePrinter success: %d bytes written to '%s'", bytesWritten, printerName))
	return nil
}

func ClearPrinterQueue(ctx context.Context, printerName string) error {
	logToFrontend(ctx, fmt.Sprintf("[ClearQueue] Requesting to clear queue for '%s'", printerName))
	name, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}

	// Request Administer access to allow Purge
	defaults := PRINTER_DEFAULTS{
		pDatatype:     nil, // RAW
		pDevMode:      0,
		DesiredAccess: PRINTER_ACCESS_ADMINISTER,
	}

	var hPrinter syscall.Handle
	r1, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&hPrinter)),
		uintptr(unsafe.Pointer(&defaults)),
	)
	if r1 == 0 {
		logToFrontend(ctx, fmt.Sprintf("[ClearQueue] OpenPrinter failed (Access Denied?): %v", err))
		return fmt.Errorf("OpenPrinter failed: %v", err)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	// PRINTER_CONTROL_PURGE = 3
	const PRINTER_CONTROL_PURGE = 3
	r1, _, err = procSetPrinter.Call(
		uintptr(hPrinter),
		0,
		0,
		uintptr(PRINTER_CONTROL_PURGE),
	)

	if r1 == 0 {
		logToFrontend(ctx, fmt.Sprintf("[ClearQueue] SetPrinter (Purge) failed: %v", err))
		return fmt.Errorf("failed to purge printer: %v", err)
	}

	logToFrontend(ctx, fmt.Sprintf("[ClearQueue] Queue cleared successfully for '%s'", printerName))
	return nil
}

func logToFrontend(ctx context.Context, msg string) {
	fmt.Println(msg)
	if ctx != nil {
		runtime.EventsEmit(ctx, "backend_log", msg)
	}
}
