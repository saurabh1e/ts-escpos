# 🖨️ TS-ESCPOS

> **A Vibe Coded App** ✨
> 
> Bridging the gap between your modern web apps and traditional hardware with style.

**TS-ESCPOS** is a robust, cross-platform background service that enables seamless communication between web applications (like Point of Sale systems) and ESC/POS thermal printers. It runs a local HTTP server to accept print jobs and handles the raw communication details with your hardware.

Built with [Wails](https://wails.io), [Go](https://go.dev), and [TypeScript](https://www.typescriptlang.org/).

## 🌟 Features

- **🪟 Cross-Platform:** Optimized for macOS and Windows.
- **🚀 Local Print Server:** Exposes a simple HTTP API on port `9100`.
- **🔌 Hardware Bridge:** Connects to USB and Network ESC/POS printers.
- **🧾 Receipt Templates:** Built-in support for formatted Bills and Kitchen Order Tickets (KOT).
- **🔔 Notifications:** System-level notifications for print status.
- **🛡️ Background Service:** Designed to persist and auto-restart configuration.
- **⚡ Fast & Lightweight:** Native performance powered by Go.

## 🏗️ Architecture & How It Works

Traditional web apps cannot easily access USB hardware directly. **TS-ESCPOS** solves this by running locally on the machine connected to the printers.

1.  **Your Web App** sends a JSON payload to `http://localhost:9100/api/print`.
2.  **TS-ESCPOS** receives the request, validates the machine ID, and processes the order data.
3.  **The Engine** converts the order into raw ESC/POS byte commands.
4.  **The Printer** spits out the receipt.

## 🚀 Getting Started

### Installation

1.  Download the latest release for your OS.
2.  Run the installer/application.
3.  The app will start in the background. You might see a system tray icon or a window indicating the status.
4.  The server starts listening on `port 9100`.

### Usage

Once running, you can interact with the app via its HTTP API.

## 📡 API Reference

Base URL: `http://localhost:9100`

### 1. Identify Machine
Get the unique identifier for the machine running the service.

- **Endpoint:** `GET /api/identifier`
- **Response:**
  ```json
  {
      "identifier": "MACHINE-UNIQUE-ID-123",
      "os": "darwin",
      "hostname": "MacBook-Pro.local"
  }
  ```

### 2. List Printers
Get a list of available printers connected to the system.

- **Endpoint:** `GET /api/printers`
- **Response:**
  ```json
  [
      {
          "name": "EPSON_TM_T82",
          "uniqueId": "USB_123",
          "status": "Ready"
      },
      {
          "name": "POS-80",
          "uniqueId": "USB_456",
          "status": "Ready"
      }
  ]
  ```

### 3. Print
Send a print job.

- **Endpoint:** `POST /api/print`
- **Headers:** `Content-Type: application/json`
- **Body Params:**
    - `machineId`: The ID from `/identifier`.
    - `printerName`: Exact name of the printer to use.
    - `printerSize`: Width of paper (e.g., "80mm").
    - `receiptType`: "bill" or "kot".
    - `orderData`: Object containing receipt details.

#### Example: Print Bill

```json
{
  "machineId": "YOUR-MACHINE-ID",
  "printerName": "EPSON_TM_T82",
  "printerSize": "80mm",
  "receiptType": "bill",
  "orderData": {
    "invoiceNo": "302",
    "date": "23/01/2026, 11:49:46 pm",
    "customerName": "John Doe",
    "tableNo": "5",
    "items": [
      {
        "name": "Adrak Chai (Serves 2)",
        "quantity": 1,
        "price": 129
      },
      {
        "name": "Masala Chai",
        "quantity": 1,
        "price": 129,
        "children": [
            { "name": "Less Sugar", "price": 0, "quantity": 1 }
        ]
      }
    ],
    "subTotal": 258.00,
    "tax": 0,
    "total": 258.00,
    "paymentMode": "Cash"
  }
}
```

#### Example: Print KOT (Kitchen)

```json
{
  "machineId": "YOUR-MACHINE-ID",
  "printerName": "Kitchen_Printer_1",
  "printerSize": "80mm",
  "receiptType": "kot",
  "orderData": {
    "tableNo": "5",
    "date": "23/01/2026, 11:49:46 pm",
    "items": [
      {
        "name": "Adrak Chai",
        "quantity": 1,
        "children": ""
      }
    ]
  }
}
```

### 4. Test Notification
Trigger a system test notification.

- **Endpoint:** `POST /api/test-notification`
- **Body:**
  ```json
  {
      "title": "Hello",
      "message": "Is it me you're looking for?",
      "sound": true
  }
  ```

## 📦 Releasing

To create a new release for Windows users:

1.  **Ensure you are on the `main` branch** and have a clean working directory.
2.  **Run the release script** with the new version tag:

    ```bash
    ./scripts/release.sh v1.0.0
    ```

    This script will:
    - Build the Windows binary and installer locally (using your macOS machine).
    - Commit the built artifacts to the repository.
    - Create a git tag and push everything to GitHub.
    - GitHub Actions will automatically create a new release with the uploaded artifacts.

**Note:** You must have `nsis` installed (`brew install nsis`) to build the installer.

## 🛠️ Development

### Prerequisites
- Go 1.23+
- Node.js & pnpm

### Setup
1. Clone the repo.
2. Install dependencies:
   ```bash
   pnpm install
   ```

### Run Locally
Run the app in development mode with hot-reload:
```bash
wails dev
```

### Build
Build the production binary:
```bash
wails build
```

## 📂 Project Structure

```
ts-escpos/
├── app.go              # App Lifecycle & Wails bindings
├── main.go             # Entry point
├── backend/            # Go Backend Logic
│   ├── config/         # Configuration & OS Specifics
│   ├── jobs/           # Job Store & Logging
│   ├── printer/        # ESC/POS Logic & Printer Services
│   ├── receipt/        # Receipt Templates (Bill/KOT)
│   ├── server/         # HTTP API Server
│   └── updater/        # Self-updater logic
├── frontend/           # Vite + React + Tailwind UI
│   └── src/            # Frontend Source
└── build/              # Build artifacts
```

## 💫 The Vibe

This code was crafted with strict attention to "it just works." It's designed to disappear into the background and do its job reliably, so you can focus on building the front-of-house experience.

---
