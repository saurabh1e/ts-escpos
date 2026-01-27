import { TestPrint } from '../../wailsjs/go/main/App';

// Temporary interfaces until wails generates them
export interface PrinterInfo {
    name: string;
    uniqueId: string;
    windowsId: string;
    status: string;
}

export class PrinterList {
    private element: HTMLElement;
    private printers: PrinterInfo[] = [];

    constructor() {
        this.element = document.createElement('div');
        this.element.className = "p-6";
        this.render();
    }

    updatePrinters(printers: PrinterInfo[]) {
        this.printers = printers;
        this.render();
    }

    render() {
        if (this.printers.length === 0) {
            this.element.innerHTML = `
                <div class="text-center py-10 text-gray-500 bg-gray-800/50 rounded-xl border border-dashed border-gray-700">
                    <p>No printers found or loading...</p>
                </div>
            `;
            return;
        }

        const grid = document.createElement('div');
        grid.className = "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4";

        this.printers.forEach(printer => {
            const card = document.createElement('div');
            card.className = "bg-gray-800 rounded-xl p-4 border border-gray-700 shadow-sm hover:shadow-md transition-shadow relative overflow-hidden group";

            const isReady = printer.status.includes("Ready");
            const statusColor = isReady ? "text-green-400" : "text-yellow-400";
            const borderColor = isReady ? "border-green-500/20" : "border-yellow-500/20";

            card.classList.add(borderColor);

            card.innerHTML = `
                <div class="flex justify-between items-start mb-2">
                    <div class="bg-gray-700 p-2 rounded-lg">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                             <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
                        </svg>
                    </div>
                    <span class="text-xs font-semibold px-2 py-1 rounded-full bg-gray-900 ${statusColor}">
                        ${printer.status}
                    </span>
                </div>
                <h3 class="font-bold text-lg mb-1 truncate" title="${printer.name}">${printer.name}</h3>
                <div class="space-y-1 text-xs text-gray-400 mb-3">
                    <p class="flex justify-between">
                        <span>Win ID:</span>
                        <span class="font-mono text-gray-300">${printer.windowsId}</span>
                    </p>
                    <p class="flex justify-between">
                        <span>UID:</span>
                        <span class="font-mono text-gray-300 truncate w-24 text-right" title="${printer.uniqueId}">${printer.uniqueId}</span>
                    </p>
                </div>
                <button class="test-print-btn w-full py-2 px-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2" data-printer="${printer.name}">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
                    </svg>
                    Test Print
                </button>
            `;

            const btn = card.querySelector('.test-print-btn') as HTMLButtonElement;
            if (btn) {
                btn.onclick = async (e) => {
                    e.preventDefault();
                    if (btn.disabled) return;

                    const originalContent = btn.innerHTML;
                    const originalClass = btn.className;

                    btn.disabled = true;
                    btn.innerHTML = `<svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg> Printing...`;

                    try {
                        await TestPrint(printer.name);

                        btn.className = "w-full py-2 px-3 bg-green-600 hover:bg-green-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2";
                        btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                        </svg> Done`;

                        setTimeout(() => {
                            btn.className = originalClass;
                            btn.innerHTML = originalContent;
                            btn.disabled = false;
                        }, 2000);

                    } catch (error) {
                        console.error('Test print failed:', error);
                        btn.className = "w-full py-2 px-3 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2";
                        btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
                        </svg> Failed`;

                        setTimeout(() => {
                            btn.className = originalClass;
                            btn.innerHTML = originalContent;
                            btn.disabled = false;
                        }, 3000);
                    }
                };
            }

            grid.appendChild(card);
        });

        this.element.innerHTML = `<h2 class="text-xl font-bold mb-4 flex items-center gap-2">
            Connected Printers <span class="text-sm font-normal text-gray-500 bg-gray-800 px-2 py-0.5 rounded-full">${this.printers.length}</span>
        </h2>`;
        this.element.appendChild(grid);
    }

    getElement(): HTMLElement {
        return this.element;
    }
}
