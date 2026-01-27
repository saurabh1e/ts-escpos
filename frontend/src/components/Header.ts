export class Header {
    private element: HTMLElement;

    constructor() {
        this.element = document.createElement('header');
        this.element.className = "bg-brand-dark border-b border-gray-700 p-4 flex justify-between items-center shadow-lg";
        this.render();
    }

    updateStatus(machineId: string, port: number, running: boolean) {
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

    render() {
        this.element.innerHTML = `
            <div class="flex items-center gap-3">
                <div class="bg-brand-blue p-2 rounded-lg">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
                    </svg>
                </div>
                <h1 class="text-xl font-bold tracking-wide">Print Link Service</h1>
            </div>

            <div class="flex items-center gap-6 text-sm text-gray-400">
                <div id="server-status" class="bg-gray-800 px-3 py-1 rounded-full border border-gray-700">
                    Loading...
                </div>
                <div class="flex flex-row items-end gap-2">
                    <span class="text-xs uppercase tracking-wider text-gray-500">Machine ID:</span>
                    <span id="machine-id" class="font-mono text-gray-300">...</span>
                </div>
            </div>
        `;
    }

    getElement(): HTMLElement {
        return this.element;
    }
}
