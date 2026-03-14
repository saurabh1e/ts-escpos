export interface UpdateStatus {
    state: string;
    message: string;
    version: string;
    lastCheckedAt: string;
}

export class Header {
    private element: HTMLElement;
    private port: number = 9100;
    private onCheckUpdates: (() => Promise<void> | void) | null = null;

    constructor() {
        this.element = document.createElement('header');
        this.element.className = 'bg-brand-dark border-b border-white/[0.06] px-5 py-3 flex items-center justify-between flex-shrink-0';
        this.render();
    }

    updateStatus(machineId: string, port: number, running: boolean) {
        this.port = port;

        const idEl = this.element.querySelector('#machine-id') as HTMLElement | null;
        const idWrap = this.element.querySelector('#machine-id-wrap') as HTMLElement | null;
        const statusEl = this.element.querySelector('#server-status') as HTMLElement | null;

        if (idEl) {
            idEl.textContent = machineId ? machineId.slice(0, 8) + '…' : '—';
        }
        if (idWrap && machineId) {
            idWrap.title = machineId;
        }
        if (statusEl) {
            const dotClass = running
                ? 'bg-green-400 shadow-[0_0_5px_#4ade80]'
                : 'bg-red-400';
            statusEl.innerHTML = `
                <span class="w-1.5 h-1.5 rounded-full flex-shrink-0 ${dotClass}"></span>
                <span class="font-mono">:${port}</span>
            `;
        }
    }

    setVersion(version: string) {
        const verEl = this.element.querySelector('#app-version');
        if (verEl) {
            verEl.textContent = `v${version.replace(/^v/, '')}`;
        }
    }

    setUpdateCheckHandler(handler: () => Promise<void> | void) {
        this.onCheckUpdates = handler;
    }

    setUpdateStatus(status: UpdateStatus) {
        const btnEl = this.element.querySelector('#check-updates-btn') as HTMLButtonElement | null;
        const iconEl = this.element.querySelector('#update-icon') as HTMLElement | null;
        const labelEl = this.element.querySelector('#update-label') as HTMLElement | null;
        const timeEl = this.element.querySelector('#update-time') as HTMLElement | null;

        const isBusy = ['checking', 'downloading', 'installing', 'closing_instances'].includes(status.state);

        if (iconEl) iconEl.innerHTML = this.renderUpdateIcon(status.state, isBusy);
        if (labelEl) labelEl.textContent = this.getUpdateLabel(status.state);

        if (btnEl) {
            btnEl.disabled = isBusy;
            btnEl.className = `flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-all ${this.getUpdateBtnClass(status.state, isBusy)}`;
            btnEl.title = status.message || '';
        }

        if (timeEl) {
            timeEl.textContent = status.lastCheckedAt
                ? new Date(status.lastCheckedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
                : '';
        }
    }

    private getUpdateLabel(state: string): string {
        const map: Record<string, string> = {
            checking: 'Checking…',
            downloading: 'Downloading…',
            installing: 'Installing…',
            closing_instances: 'Closing…',
            up_to_date: 'Up to date',
            update_available: 'Update available',
            error: 'Check failed',
        };
        return map[state] ?? 'Check updates';
    }

    private getUpdateBtnClass(state: string, isBusy: boolean): string {
        if (isBusy) return 'bg-blue-500/10 text-blue-400 border-blue-500/20 cursor-wait opacity-80';
        if (state === 'up_to_date') return 'bg-green-500/10 text-green-400 border-green-500/20 hover:bg-green-500/15 cursor-pointer';
        if (state === 'update_available') return 'bg-amber-500/10 text-amber-400 border-amber-500/20 hover:bg-amber-500/15 cursor-pointer';
        if (state === 'error') return 'bg-red-500/10 text-red-400 border-red-500/20 hover:bg-red-500/15 cursor-pointer';
        return 'bg-white/5 text-gray-400 border-white/10 hover:bg-white/8 hover:text-gray-200 cursor-pointer';
    }

    private renderUpdateIcon(state: string, isBusy: boolean): string {
        if (isBusy) {
            return `<svg class="h-3.5 w-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-20" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-80" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>`;
        }
        if (state === 'up_to_date') {
            return `<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
            </svg>`;
        }
        if (state === 'update_available') {
            return `<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
            </svg>`;
        }
        if (state === 'error') {
            return `<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>`;
        }
        return `<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>`;
    }

    render() {
        this.element.innerHTML = `
            <div class="flex items-center gap-3">
                <div class="bg-brand-blue/10 border border-brand-blue/20 p-2 rounded-xl flex-shrink-0">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-brand-blue" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75"
                            d="M6 9V2h12v7M6 18H4a2 2 0 01-2-2v-5a2 2 0 012-2h16a2 2 0 012 2v5a2 2 0 01-2 2h-2M6 14h12v8H6v-8z"/>
                    </svg>
                </div>
                <div>
                    <h1 class="text-sm font-bold tracking-tight leading-none text-white">Print Link</h1>
                    <div class="flex items-center gap-1.5 mt-0.5">
                        <span class="text-[9px] font-semibold uppercase tracking-widest text-gray-600">ESC/POS</span>
                        <span class="text-gray-700 text-[9px]">·</span>
                        <span id="app-version" class="text-[10px] font-mono text-brand-blue/60">v…</span>
                    </div>
                </div>
            </div>

            <div class="flex items-center gap-2">

                <button id="test-notify-btn"
                    title="Send test notification"
                    class="p-1.5 rounded-lg text-gray-600 hover:text-gray-300 hover:bg-white/5 transition-colors">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
                    </svg>
                </button>

                <button id="check-updates-btn"
                    class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border transition-all bg-white/5 text-gray-400 border-white/10 hover:bg-white/8 hover:text-gray-200 cursor-pointer">
                    <span id="update-icon">
                        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                        </svg>
                    </span>
                    <span id="update-label">Check updates</span>
                </button>

                <span id="update-time" class="text-[10px] text-gray-700 tabular-nums"></span>

                <div class="w-px h-4 bg-white/[0.07] mx-1"></div>

                <div id="server-status"
                    class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-white/[0.04] border border-white/[0.06] text-xs text-gray-500">
                    <span class="w-1.5 h-1.5 rounded-full bg-gray-700 flex-shrink-0"></span>
                    <span class="font-mono">Loading</span>
                </div>

                <div id="machine-id-wrap"
                    class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-white/[0.04] border border-white/[0.06]"
                    title="Machine ID">
                    <span class="text-[9px] font-semibold uppercase tracking-widest text-gray-700">ID</span>
                    <span id="machine-id" class="text-xs font-mono text-gray-500">…</span>
                </div>

            </div>
        `;

        this.element.querySelector('#check-updates-btn')?.addEventListener('click', async () => {
            if (this.onCheckUpdates) await this.onCheckUpdates();
        });

        this.element.querySelector('#test-notify-btn')?.addEventListener('click', async () => {
            try {
                await fetch(`http://localhost:${this.port}/api/test-notification`, {
                    method: 'POST',
                    body: JSON.stringify({ title: 'Test Notification', message: 'This is a test notification with sound!', sound: true }),
                });
            } catch (e) {
                console.error('Test notification failed', e);
            }
        });
    }

    getElement(): HTMLElement {
        return this.element;
    }
}
