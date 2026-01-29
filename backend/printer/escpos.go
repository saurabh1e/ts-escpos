package printer

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"

	_ "golang.org/x/image/webp"

	"ts-escpos/backend/receipt"

	"github.com/nfnt/resize"
	"github.com/skip2/go-qrcode"
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

func (e *EscposAdapter) SetBold(bold bool) {
	if bold {
		e.buf.Write([]byte{0x1B, 0x45, 0x01})
	} else {
		e.buf.Write([]byte{0x1B, 0x45, 0x00})
	}
}

func (e *EscposAdapter) SetDoubleStrike(enabled bool) {
	// ESC G n
	if enabled {
		e.buf.Write([]byte{0x1B, 0x47, 0x01})
	} else {
		e.buf.Write([]byte{0x1B, 0x47, 0x00})
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

func (e *EscposAdapter) PrintQRCode(data string) {
	if data == "" {
		return
	}

	// Native QR code commands often fail on generic printers.
	// Falling back to raster mode for cross-printer compatibility.

	// Create QR code
	// Size 256 is reasonable for most papers (approx 32mm width)
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		fmt.Printf("Error creating QR code: %v\n", err)
		return
	}

	img := qr.Image(256)
	e.printGraphics(img)
}

// PrintImage downloads (or uses cache), resizes, dithers, and prints an image from a URL
// It assumes a max width of 384 dots (typical for 58mm, looks ok centered on 80mm).
func (e *EscposAdapter) PrintImage(urlStr string) {
	if urlStr == "" {
		return
	}

	img, err := getImageFromURL(urlStr)
	if err != nil {
		fmt.Printf("Error processing image from URL %s: %v\n", urlStr, err)
		return
	}

	// Resize if necessary (max width 384 for broad compatibility)
	maxWidth := uint(384)
	if img.Bounds().Dx() > int(maxWidth) {
		img = resize.Resize(maxWidth, 0, img, resize.Lanczos3)
	}

	e.printGraphics(img)
}

func (e *EscposAdapter) printGraphics(img image.Image) {
	// Convert to monochrome raster
	rasterData, widthBytes, height := convertToRaster(img)

	// GS v 0 m xL xH yL yH d1...dk
	// m = 0 (Normal)
	// xL, xH = bytes horizontal
	// yL, yH = dots vertical
	e.buf.Write([]byte{0x1D, 0x76, 0x30, 0x00})
	e.buf.Write([]byte{byte(widthBytes % 256), byte(widthBytes / 256)})
	e.buf.Write([]byte{byte(height % 256), byte(height / 256)})
	e.buf.Write(rasterData)
}

func getImageFromURL(urlStr string) (image.Image, error) {
	// 1. Check Cache
	cacheDir := filepath.Join(os.TempDir(), "ts-escpos", "images")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %v", err)
	}

	hash := md5.Sum([]byte(urlStr))
	filename := hex.EncodeToString(hash[:])
	cachePath := filepath.Join(cacheDir, filename)

	// Try reading from file first
	if f, err := os.Open(cachePath); err == nil {
		defer f.Close()
		img, _, err := image.Decode(f)
		if err == nil {
			return img, nil
		}
		// If decode fails, maybe file is corrupt. Continue to download.
	}

	// 2. Download
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// Read into bytes to save to file and decode
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		fmt.Printf("Warning: Failed to write to cache: %v\n", err)
	}

	// Decode
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func convertToRaster(img image.Image) ([]byte, int, int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	widthBytes := (width + 7) / 8

	data := make([]byte, widthBytes*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(x+bounds.Min.X, y+bounds.Min.Y)
			g := color.GrayModel.Convert(c).(color.Gray)
			// Thresholding (128) - 1 is black (print), 0 is white (no print)
			// In ESC/POS raster, 1 means print dot.
			// Light pixels (high value) are white. Dark pixels (low value) are black.
			if g.Y < 128 {
				byteIndex := (y * widthBytes) + (x / 8)
				bitIndex := 7 - (x % 8)
				data[byteIndex] |= (1 << bitIndex)
			}
		}
	}
	return data, widthBytes, height
}

func (e *EscposAdapter) GetBytes() []byte {
	return e.buf.Bytes()
}
