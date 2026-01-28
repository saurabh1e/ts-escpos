export class Header {
    private element: HTMLElement;
    private port: number = 9100; // Default

    constructor() {
        this.element = document.createElement('header');
        this.element.className = "bg-brand-dark border-b border-gray-700 p-4 flex justify-between items-center shadow-lg";
        this.render();
    }

    updateStatus(machineId: string, port: number, running: boolean) {
        this.port = port;
        const idEl = this.element.querySelector('#machine-id');
        const statusEl = this.element.querySelector('#server-status');

        if (idEl) idEl.textContent = machineId;
        if (statusEl) {
            statusEl.innerHTML = `
                <span class="flex items-center gap-2">
                    <span class="w-2.5 h-2.5 rounded-full ${running ? 'bg-green-500' : 'bg-red-500'} animate-pulse"></span>
                    <span>Port: ${port}</span>
                </span>
            `;
        }
    }

    setVersion(version: string) {
        const verEl = this.element.querySelector('#app-version');
        if (verEl) {
            // Remove 'v' prefix if present to ensure consistency
            const cleanVersion = version.replace(/^v/, '');
            verEl.textContent = `v${cleanVersion}`;
        }
    }

    render() {
        this.element.innerHTML = `
            <div class="flex items-center gap-3">
                <div class="bg-brand-blue p-2 rounded-lg">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
                    </svg>
                </div>
                <div>
                    <h1 class="text-xl font-bold tracking-wide leading-none">Print Link Service</h1>
                    <span id="app-version" class="text-xs text-gray-500 font-mono">v...</span>
                </div>
            </div>

            <div class="flex items-center gap-4 text-sm text-gray-400">
                 <button id="test-notify-btn" class="bg-gray-800 hover:bg-gray-700 text-gray-300 px-3 py-1.5 rounded-lg border border-gray-700 transition-colors text-xs font-medium flex items-center gap-2">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                    </svg>
                    Test Notify
                </button>
                <div id="server-status" class="bg-gray-800 px-3 py-1 rounded-full border border-gray-700">
                    Loading...
                </div>
                <div class="flex flex-row items-end gap-2">
                    <span class="text-xs uppercase tracking-wider text-gray-500">Machine ID:</span>
                    <span id="machine-id" class="font-mono text-gray-300">...</span>
                </div>
            </div>
        `;

        const btn = this.element.querySelector('#test-notify-btn');
        if (btn) {
            btn.addEventListener('click', async () => {
                try {
                    await fetch(`http://localhost:${this.port}/api/test-notification`, {
                        method: 'POST',
                        body: JSON.stringify({
                            title: "Test Notification",
                            message: "This is a test notification with sound!",
                            sound: true
                        })
                    });
                } catch (e) {
                    console.error("Test notification failed", e);
                    alert("Failed to send test notification");
                }
            });
        }
    }

    getElement(): HTMLElement {
        return this.element;
    }
}
