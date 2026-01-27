// Import styles
import './style.css';

// Import components
import { Header } from './components/Header';
import { PrinterList } from './components/PrinterList';
import { JobsLog } from './components/JobsLog';

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

    constructor() {
        this.app = document.getElementById('app')!;

        // Init Components
        this.header = new Header();
        this.printerList = new PrinterList();
        this.jobsLog = new JobsLog();

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

        // Log Section (Fixed height or resizable could be better, but split view is good)
        // Let's make it occupy the bottom half or 1/3
        const logContainer = document.createElement('div');
        logContainer.className = "h-1/3 min-h-[300px] flex flex-col";
        logContainer.appendChild(this.jobsLog.getElement());

        main.appendChild(logContainer);

        this.app.appendChild(main);
    }

    async startDataLoop() {
        const update = async () => {
            try {
                if (App) {
                    const machineId = await App.GetMachineID();
                    const status = await App.GetServerStatus();
                    this.header.updateStatus(machineId, status.port, status.running);

                    const printers = await App.GetPrinters();
                    this.printerList.updatePrinters(printers);

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

        // Poll every 3 seconds
        setInterval(update, 3000);
    }
}

// Start app
new Dashboard();
