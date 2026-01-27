export interface PrintJob {
    id: string;
    invoiceNo: string;
    printerName: string;
    status: string;
    error?: string;
    timestamp: string;
    receiptType: string;
}

export class JobsLog {
    private element: HTMLElement;
    private jobs: PrintJob[] = [];

    constructor() {
        this.element = document.createElement('div');
        this.element.className = "flex-1 flex flex-col min-h-0 bg-gray-850 border-t border-gray-700"; // min-h-0 for scroll
        this.render();
    }

    updateJobs(jobs: PrintJob[]) {
        this.jobs = jobs;
        this.renderList();
    }

    renderList() {
        const listContainer = this.element.querySelector('#jobs-list');
        if (!listContainer) return;

        if (this.jobs.length === 0) {
            listContainer.innerHTML = `
                <div class="flex flex-col items-center justify-center h-full text-gray-500 py-8">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                    </svg>
                    <p>No print jobs yet</p>
                </div>
            `;
            return;
        }

        listContainer.innerHTML = '';
        this.jobs.forEach(job => {
            const row = document.createElement('div');
            row.className = "flex items-center gap-4 p-3 hover:bg-gray-800 border-b border-gray-800 transition-colors text-sm";

            const isSuccess = job.status === 'success';
            const iconColor = isSuccess ? 'text-green-400' : 'text-red-400';
            const iconPath = isSuccess
                ? 'M5 13l4 4L19 7'
                : 'M6 18L18 6M6 6l12 12';

            const date = new Date(job.timestamp).toLocaleTimeString();

            row.innerHTML = `
                <div class="w-8 h-8 rounded-full bg-gray-800 flex items-center justify-center shrink-0 ${iconColor}">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="${iconPath}" />
                    </svg>
                </div>
                <div class="flex-1 min-w-0">
                    <div class="flex justify-between items-baseline mb-0.5">
                        <h4 class="font-medium text-white truncate">Inv #${job.invoiceNo}</h4>
                        <span class="text-xs text-gray-500">${date}</span>
                    </div>
                    <div class="flex items-center gap-2 text-xs text-gray-400">
                        <span class="uppercase tracking-wider font-bold text-[10px] px-1.5 py-0.5 rounded bg-gray-700">${job.receiptType}</span>
                        <span class="truncate">via ${job.printerName}</span>
                    </div>
                    ${!isSuccess ? `<div class="text-red-400 text-xs mt-1 truncate">${job.error}</div>` : ''}
                </div>
            `;
            listContainer.appendChild(row);
        });
    }

    render() {
        this.element.innerHTML = `
            <div class="p-4 border-b border-gray-700 bg-gray-800">
                <h2 class="text-lg font-bold">Recent Jobs</h2>
            </div>
            <div id="jobs-list" class="overflow-y-auto flex-1 custom-scrollbar">
                <!-- List items -->
            </div>
        `;
        this.renderList();
    }

    getElement(): HTMLElement {
        return this.element;
    }
}
