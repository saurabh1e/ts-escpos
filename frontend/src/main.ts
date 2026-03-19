// Import styles
import './style.css';

// Import components
import { Header, UpdateStatus } from './components/Header';
import { PrinterList } from './components/PrinterList';
import { JobsLog } from './components/JobsLog';
import { SystemLog } from './components/SystemLog';

// We need to declare the window.runtime functions i f typing is not yet generated
// or rely on @ts-ignore.
// Best to rely on wailsjs generated files, but they don't exist yet with proper signatures.
// We'll create a local helper to interface with Backend.

// @ts-ignore
const App = window['go']['main']['App'];

class Dashboard {
    private app: HTMLElement;
    private header: Header;
    private printerList: PrinterList;
    private jobsLog: JobsLog;
    private systemLog: SystemLog;
    private machineId: string = "";

    constructor() {
        this.app = document.getElementById('app')!;

        // Init Components
        this.header = new Header();
        this.printerList = new PrinterList();
        this.jobsLog = new JobsLog();
        this.systemLog = new SystemLog();
        this.header.setUpdateCheckHandler(() => this.runManualUpdateCheck());

        this.setupNotifications();

        this.render();
        this.startDataLoop();
    }

    setupNotifications() {
        // @ts-ignore
        if (window.runtime && window.runtime.EventsOn) {
            // @ts-ignore
            window.runtime.EventsOn("error_notification", (data: any) => {
                // Simple alert or toast
                // Using browser notification API or just alert for now as requested "show notification"
                // Ideally, a nice toast component.
                // Assuming data has title and message
                if (Notification.permission === "granted") {
                    new Notification(data.title, { body: data.message });
                } else if (Notification.permission !== "denied") {
                    Notification.requestPermission().then(permission => {
                        if (permission === "granted") {
                            new Notification(data.title, { body: data.message });
                        } else {
                            alert(`${data.title}: ${data.message}`);
                        }
                    });
                } else {
                     alert(`${data.title}: ${data.message}`);
                }
            });

            // @ts-ignore
            window.runtime.EventsOn("update_available", (data: any) => {
                const title = "Update Available";
                const version = data?.version ? `Version ${data.version} was found. The installer will download automatically.` : "A new version was found and will install automatically.";

                if (Notification.permission === "granted") {
                    new Notification(title, { body: version });
                    return;
                }

                if (Notification.permission !== "denied") {
                    Notification.requestPermission().then(permission => {
                        if (permission === "granted") {
                            new Notification(title, { body: version });
                            return;
                        }

                        alert(`${title}: ${version}`);
                    });
                    return;
                }

                alert(`${title}: ${version}`);
            });

            // @ts-ignore
            window.runtime.EventsOn("update_status", (data: any) => {
                this.header.setUpdateStatus(this.normalizeUpdateStatus(data));
            });
        }
    }

    normalizeUpdateStatus(data: any): UpdateStatus {
        return {
            state: typeof data?.state === 'string' ? data.state : 'idle',
            message: typeof data?.message === 'string' ? data.message : 'Automatic updates enabled. Waiting for the next check.',
            version: typeof data?.version === 'string' ? data.version : '',
            lastCheckedAt: typeof data?.lastCheckedAt === 'string' ? data.lastCheckedAt : '',
        };
    }

    async runManualUpdateCheck() {
        if (!App || !App.CheckForUpdatesNow) {
            return;
        }

        try {
            const status = await App.CheckForUpdatesNow();
            this.header.setUpdateStatus(this.normalizeUpdateStatus(status));
        } catch (error) {
            const message = `Failed to check for updates: ${error}`;
            console.error(message);
            alert(message);
        }
    }

    render() {
        this.app.innerHTML = '';
        this.app.appendChild(this.header.getElement());

        // Main Content Area
        const main = document.createElement('main');
        main.className = "flex-1 flex flex-col overflow-hidden";

        // Scrollable content for printers
        const content = document.createElement('div');
        content.className = "flex-1 overflow-y-auto";
        content.appendChild(this.printerList.getElement());

        main.appendChild(content);

        // Bottom Section with Tabs
        const bottomSection = document.createElement('div');
        bottomSection.className = "h-1/3 min-h-[300px] flex flex-col border-t border-gray-700 bg-gray-800";

        // Tabs
        const tabs = document.createElement('div');
        tabs.className = "flex border-b border-gray-700";

        const btnJobs = this.createTabBtn("Recent Jobs", true);
        const btnLogs = this.createTabBtn("System Logs", false);

        btnJobs.onclick = () => {
            this.activateTab(btnJobs, btnLogs);
            this.jobsLog.getElement().classList.remove('hidden');
            this.systemLog.hide();
        };

        btnLogs.onclick = () => {
            this.activateTab(btnLogs, btnJobs);
            this.jobsLog.getElement().classList.add('hidden');
            this.systemLog.show();
        };

        tabs.appendChild(btnJobs);
        tabs.appendChild(btnLogs);
        bottomSection.appendChild(tabs);

        // Containers
        bottomSection.appendChild(this.jobsLog.getElement());
        bottomSection.appendChild(this.systemLog.getElement());

        main.appendChild(bottomSection);

        this.app.appendChild(main);
    }

    createTabBtn(text: string, isActive: boolean) {
        const btn = document.createElement('button');
        btn.textContent = text;
        btn.className = `flex-1 py-2 text-sm font-medium transition-colors ${isActive ? 'bg-gray-800 text-blue-400 border-b-2 border-blue-400' : 'bg-gray-900 text-gray-500 hover:text-gray-300'}`;
        return btn;
    }

    activateTab(active: HTMLElement, inactive: HTMLElement) {
        active.className = "flex-1 py-2 text-sm font-medium transition-colors bg-gray-800 text-blue-400 border-b-2 border-blue-400";
        inactive.className = "flex-1 py-2 text-sm font-medium transition-colors bg-gray-900 text-gray-500 hover:text-gray-300";
    }

    async startDataLoop() {
        if (App) {
            try {
                const version = await App.GetVersion();
                this.header.setVersion(version);

                const updateStatus = await App.GetUpdateStatus();
                this.header.setUpdateStatus(this.normalizeUpdateStatus(updateStatus));

                this.machineId = await App.GetMachineID();

                const printers = await App.GetPrinters();
                this.printerList.updatePrinters(printers);
            } catch (e) {
                console.error("Failed to get initial data:", e);
            }
        }

        const update = async () => {
            try {
                if (App) {
                    const status = await App.GetServerStatus();
                    this.header.updateStatus(this.machineId, status.port, status.running);

                    const jobs = await App.GetPrintJobs();
                    this.jobsLog.updateJobs(jobs);
                } else {
                    console.log("Wails backend not connected.");
                }
            } catch (e) {
                console.error("Failed to fetch data:", e);
            }
        };

        // Initial call
        await update();

        // Poll every 1 minute
        setInterval(update, 60000);
    }
}

// Start app
new Dashboard();
