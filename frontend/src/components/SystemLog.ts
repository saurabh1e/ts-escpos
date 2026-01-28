export class SystemLog {
    private element: HTMLElement;
    private logs: string[] = [];
    private maxLogs = 100;

    constructor() {
        this.element = document.createElement('div');
        this.element.className = "flex-1 flex flex-col min-h-0 bg-gray-900 border-t border-gray-700 hidden"; // Hidden by default if we use tabs
        this.setupListener();
        this.render();
    }

    setupListener() {
        // @ts-ignore
        if (window.runtime && window.runtime.EventsOn) {
            // @ts-ignore
            window.runtime.EventsOn("backend_log", (msg: string) => {
                this.addLog(msg);
            });
        }
    }

    addLog(msg: string) {
        const timestamp = new Date().toLocaleTimeString();
        const fullMsg = `[${timestamp}] ${msg}`;
        this.logs.push(fullMsg);
        if (this.logs.length > this.maxLogs) {
            this.logs.shift();
        }
        this.renderList();
    }

    renderList() {
        const listContainer = this.element.querySelector('#sys-log-list');
        if (!listContainer) return;


        const logHTML = this.logs.map(log => {
             // Basic highlight for errors
             const colorClass = log.toLowerCase().includes('error') || log.toLowerCase().includes('fail')
                ? 'text-red-400'
                : log.toLowerCase().includes('success')
                    ? 'text-green-400'
                    : 'text-gray-400';

             return `<div class="font-mono text-xs py-0.5 border-b border-gray-800/50 ${colorClass} break-all whitespace-pre-wrap">${log}</div>`;
        }).join('');

        listContainer.innerHTML = logHTML;

        if (true) { // Always scroll to bottom for now
            listContainer.scrollTop = listContainer.scrollHeight;
        }
    }

    render() {
        this.element.innerHTML = `
            <div class="p-2 border-b border-gray-700 bg-gray-800 flex justify-between items-center">
                <h2 class="text-xs font-bold uppercase tracking-wider text-gray-500">System Logs</h2>
                <button id="clear-logs" class="text-xs text-gray-500 hover:text-white transition-colors">Clear</button>
            </div>
            <div id="sys-log-list" class="overflow-y-auto flex-1 p-2 bg-black/30 font-mono">
                <!-- List items -->
            </div>
        `;

        this.element.querySelector('#clear-logs')?.addEventListener('click', () => {
            this.logs = [];
            this.renderList();
        });

        this.renderList();
    }

    getElement(): HTMLElement {
        return this.element;
    }

    show() {
        this.element.classList.remove('hidden');
    }

    hide() {
        this.element.classList.add('hidden');
    }
}
