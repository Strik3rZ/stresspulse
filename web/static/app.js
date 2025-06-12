class StressPulseUI {
    constructor() {
        this.isRunning = false;
        this.startTime = null;
        this.autoScroll = true;
        this.updateInterval = null;
        this.logUpdateInterval = null;
        this.maxLogEntries = 1000;
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadConfiguration();
        this.startMetricsUpdate();
        this.startLogUpdate();
    }

    bindEvents() {
        document.getElementById('start-all').addEventListener('click', () => this.startAll());
        document.getElementById('stop-all').addEventListener('click', () => this.stopAll());
        document.getElementById('restart-all').addEventListener('click', () => this.restartAll());
        document.getElementById('save-config').addEventListener('click', () => this.saveConfiguration());

        document.getElementById('clear-logs').addEventListener('click', () => this.clearLogs());
        document.getElementById('toggle-autoscroll').addEventListener('click', () => this.toggleAutoScroll());

        this.bindFormEvents();
    }

    bindFormEvents() {
        const inputs = document.querySelectorAll('input, select, textarea');
        inputs.forEach(input => {
            input.addEventListener('change', () => this.onConfigurationChange());
            input.addEventListener('input', () => this.onConfigurationChange());
        });
    }

    onConfigurationChange() {
        clearTimeout(this.saveTimeout);
        this.saveTimeout = setTimeout(() => {
            this.saveConfiguration(false);
        }, 1000);
    }

    async startAll() {
        try {
            const config = this.getConfiguration();
            const response = await fetch('/api/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config)
            });

            if (response.ok) {
                this.isRunning = true;
                this.startTime = new Date();
                this.updateStatus();
                this.addLog('✅ All stress tests started successfully', 'success');
            } else {
                const error = await response.text();
                this.addLog(`❌ Failed to start: ${error}`, 'error');
            }
        } catch (error) {
            this.addLog(`❌ Error starting stress tests: ${error.message}`, 'error');
        }
    }

    async stopAll() {
        try {
            const response = await fetch('/api/stop', {
                method: 'POST'
            });

            if (response.ok) {
                this.isRunning = false;
                this.startTime = null;
                this.updateStatus();
                this.addLog('🛑 All stress tests stopped', 'info');
            } else {
                const error = await response.text();
                this.addLog(`❌ Failed to stop: ${error}`, 'error');
            }
        } catch (error) {
            this.addLog(`❌ Error stopping stress tests: ${error.message}`, 'error');
        }
    }

    async restartAll() {
        this.addLog('🔄 Restarting all stress tests...', 'info');
        await this.stopAll();
        setTimeout(() => this.startAll(), 1000);
    }

    getConfiguration() {
        return {
            cpu: {
                enabled: document.getElementById('cpu-enabled').checked,
                load: parseInt(document.getElementById('cpu-load').value),
                pattern: document.getElementById('cpu-pattern').value,
                drift: parseInt(document.getElementById('cpu-drift').value)
            },
            memory: {
                enabled: document.getElementById('memory-enabled').checked,
                target: parseInt(document.getElementById('memory-size').value),
                pattern: document.getElementById('memory-pattern').value
            },
            http: {
                enabled: document.getElementById('http-enabled').checked,
                url: document.getElementById('http-url').value,
                rps: parseInt(document.getElementById('http-rps').value),
                method: document.getElementById('http-method').value,
                pattern: document.getElementById('http-pattern').value,
                headers: this.parseHeaders(document.getElementById('http-headers').value),
                body: document.getElementById('http-body').value
            },
            websocket: {
                enabled: document.getElementById('ws-enabled').checked,
                url: document.getElementById('ws-url').value,
                cps: parseInt(document.getElementById('ws-cps').value),
                pattern: document.getElementById('ws-pattern').value,
                messageInterval: parseInt(document.getElementById('ws-message-interval').value),
                messageSize: parseInt(document.getElementById('ws-message-size').value)
            },
            grpc: {
                enabled: document.getElementById('grpc-enabled').checked,
                address: document.getElementById('grpc-addr').value,
                rps: parseInt(document.getElementById('grpc-rps').value),
                method: document.getElementById('grpc-method').value,
                pattern: document.getElementById('grpc-pattern').value,
                secure: document.getElementById('grpc-secure').checked,
                service: document.getElementById('grpc-service').value
            },
            fakeLogsEnabled: document.getElementById('fake-logs-enabled').checked,
            fakeLogsType: document.getElementById('fake-logs-type').value,
            duration: document.getElementById('duration').value,
            workers: parseInt(document.getElementById('workers').value) || 0
        };
    }

    setConfiguration(config) {
        document.getElementById('cpu-enabled').checked = config.cpu?.enabled || false;
        document.getElementById('cpu-load').value = config.cpu?.load || 50;
        document.getElementById('cpu-pattern').value = config.cpu?.pattern || 'sine';
        document.getElementById('cpu-drift').value = config.cpu?.drift || 10;

        document.getElementById('memory-enabled').checked = config.memory?.enabled || false;
        document.getElementById('memory-size').value = config.memory?.target || 100;
        document.getElementById('memory-pattern').value = config.memory?.pattern || 'constant';

        document.getElementById('http-enabled').checked = config.http?.enabled || false;
        document.getElementById('http-url').value = config.http?.url || 'http://httpbin.org/get';
        document.getElementById('http-rps').value = config.http?.rps || 10;
        document.getElementById('http-method').value = config.http?.method || 'GET';
        document.getElementById('http-pattern').value = config.http?.pattern || 'constant';
        document.getElementById('http-headers').value = this.headersToString(config.http?.headers);
        document.getElementById('http-body').value = config.http?.body || '';

        document.getElementById('ws-enabled').checked = config.websocket?.enabled || false;
        document.getElementById('ws-url').value = config.websocket?.url || 'ws://echo.websocket.org';
        document.getElementById('ws-cps').value = config.websocket?.cps || 5;
        document.getElementById('ws-pattern').value = config.websocket?.pattern || 'constant';
        document.getElementById('ws-message-interval').value = config.websocket?.messageInterval || 2;
        document.getElementById('ws-message-size').value = config.websocket?.messageSize || 256;

        document.getElementById('grpc-enabled').checked = config.grpc?.enabled || false;
        document.getElementById('grpc-addr').value = config.grpc?.address || 'localhost:9000';
        document.getElementById('grpc-rps').value = config.grpc?.rps || 10;
        document.getElementById('grpc-method').value = config.grpc?.method || 'health_check';
        document.getElementById('grpc-pattern').value = config.grpc?.pattern || 'constant';
        document.getElementById('grpc-secure').checked = config.grpc?.secure || false;
        document.getElementById('grpc-service').value = config.grpc?.service || '';

        document.getElementById('fake-logs-enabled').checked = config.fakeLogsEnabled || false;
        document.getElementById('fake-logs-type').value = config.fakeLogsType || 'generic';
        document.getElementById('duration').value = config.duration || '0';
        document.getElementById('workers').value = config.workers || 0;
    }

    parseHeaders(headersString) {
        if (!headersString) return {};
        
        const headers = {};
        headersString.split(',').forEach(pair => {
            const [key, value] = pair.split(':');
            if (key && value) {
                headers[key.trim()] = value.trim();
            }
        });
        return headers;
    }

    headersToString(headers) {
        if (!headers) return '';
        return Object.entries(headers).map(([key, value]) => `${key}:${value}`).join(',');
    }

    async saveConfiguration(showNotification = true) {
        try {
            const config = this.getConfiguration();
            localStorage.setItem('stresspulse-config', JSON.stringify(config));
            
            if (showNotification) {
                this.addLog('💾 Configuration saved', 'success');
            }
        } catch (error) {
            this.addLog(`❌ Failed to save configuration: ${error.message}`, 'error');
        }
    }

    loadConfiguration() {
        try {
            const saved = localStorage.getItem('stresspulse-config');
            if (saved) {
                const config = JSON.parse(saved);
                this.setConfiguration(config);
                this.addLog('📁 Configuration loaded from browser storage', 'info');
            }
        } catch (error) {
            this.addLog(`⚠️ Failed to load configuration: ${error.message}`, 'warning');
        }
    }

    startMetricsUpdate() {
        this.updateInterval = setInterval(() => {
            this.updateMetrics();
            this.updateStatus();
        }, 2000);
    }

    async updateMetrics() {
        try {
            const response = await fetch('/api/stats');
            if (response.ok) {
                const stats = await response.json();
                this.updateMetricsDisplay(stats);
            }
        } catch (error) {

        }
    }

    updateMetricsDisplay(stats) {
        if (stats.cpu) {
            document.getElementById('cpu-current').textContent = `${Math.round(stats.cpu.current)}%`;
            document.getElementById('cpu-target').textContent = `${stats.cpu.target}%`;
            document.getElementById('cpu-progress').style.width = `${stats.cpu.current}%`;
        }

        if (stats.memory) {
            document.getElementById('memory-current').textContent = `${Math.round(stats.memory.current)} MB`;
            document.getElementById('memory-target').textContent = `${stats.memory.target} MB`;
            const memoryPercent = stats.memory.target > 0 ? (stats.memory.current / stats.memory.target) * 100 : 0;
            document.getElementById('memory-progress').style.width = `${Math.min(memoryPercent, 100)}%`;
        }

        if (stats.http) {
            document.getElementById('http-current').textContent = stats.http.currentRPS || '0';
            document.getElementById('http-target').textContent = stats.http.targetRPS || '0';
            document.getElementById('http-success').textContent = `${Math.round(stats.http.successRate || 0)}%`;
        }

        if (stats.websocket) {
            document.getElementById('ws-current').textContent = stats.websocket.currentCPS || '0';
            document.getElementById('ws-active').textContent = stats.websocket.activeConnections || '0';
            document.getElementById('ws-success').textContent = `${Math.round(stats.websocket.successRate || 0)}%`;
        }

        if (stats.grpc) {
            document.getElementById('grpc-current').textContent = stats.grpc.currentRPS || '0';
            document.getElementById('grpc-target').textContent = stats.grpc.targetRPS || '0';
            document.getElementById('grpc-success').textContent = `${Math.round(stats.grpc.successRate || 0)}%`;
        }

        this.updateActiveStates(stats);
    }

    updateActiveStates(stats) {
        const cpuCard = document.querySelector('.metric-card:nth-child(1)');
        const memoryCard = document.querySelector('.metric-card:nth-child(2)');
        const httpCard = document.querySelector('.metric-card:nth-child(3)');
        const wsCard = document.querySelector('.metric-card:nth-child(4)');
        const grpcCard = document.querySelector('.metric-card:nth-child(5)');

        cpuCard?.classList.toggle('active', stats.cpu?.enabled);
        memoryCard?.classList.toggle('active', stats.memory?.enabled);
        httpCard?.classList.toggle('active', stats.http?.enabled);
        wsCard?.classList.toggle('active', stats.websocket?.enabled);
        grpcCard?.classList.toggle('active', stats.grpc?.enabled);
    }

    updateStatus() {
        const statusElement = document.getElementById('global-status');
        const uptimeElement = document.getElementById('uptime');

        if (this.isRunning) {
            statusElement.textContent = 'RUNNING';
            statusElement.className = 'status-value running';
            
            if (this.startTime) {
                const uptime = this.formatUptime(new Date() - this.startTime);
                uptimeElement.textContent = uptime;
            }
        } else {
            statusElement.textContent = 'STOPPED';
            statusElement.className = 'status-value stopped';
            uptimeElement.textContent = '00:00:00';
        }
    }

    formatUptime(milliseconds) {
        const seconds = Math.floor(milliseconds / 1000);
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }

    startLogUpdate() {
        this.logUpdateInterval = setInterval(() => {
            this.updateLogs();
        }, 1000);
    }

    async updateLogs() {
        try {
            const response = await fetch('/api/logs');
            if (response.ok) {
                const logs = await response.json();
                this.displayLogs(logs);
            }
        } catch (error) {

        }
    }

    displayLogs(logs) {
        const logsOutput = document.getElementById('logs');
        const logsContainer = document.querySelector('.logs-container');
        
        logs.forEach(log => {
            this.addLogEntry(log.timestamp, log.level, log.message);
        });

        this.limitLogEntries();

        if (this.autoScroll) {
            logsContainer.scrollTop = logsContainer.scrollHeight;
        }
    }

    addLog(message, level = 'info') {
        const timestamp = new Date().toISOString();
        this.addLogEntry(timestamp, level, message);
        this.limitLogEntries();

        if (this.autoScroll) {
            const logsContainer = document.querySelector('.logs-container');
            logsContainer.scrollTop = logsContainer.scrollHeight;
        }
    }

    addLogEntry(timestamp, level, message) {
        const logsOutput = document.getElementById('logs');
        const time = new Date(timestamp).toLocaleTimeString();
        const logLine = `[${time}] ${level.toUpperCase()}: ${message}\n`;
        
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${level}`;
        logEntry.textContent = logLine;
        
        logsOutput.appendChild(logEntry);
    }

    limitLogEntries() {
        const logsOutput = document.getElementById('logs');
        const entries = logsOutput.children;
        
        while (entries.length > this.maxLogEntries) {
            logsOutput.removeChild(entries[0]);
        }
    }

    clearLogs() {
        document.getElementById('logs').innerHTML = '';
        this.addLog('📝 Logs cleared', 'info');
    }

    toggleAutoScroll() {
        this.autoScroll = !this.autoScroll;
        const button = document.getElementById('toggle-autoscroll');
        button.textContent = `Auto-scroll: ${this.autoScroll ? 'ON' : 'OFF'}`;
        button.classList.toggle('btn-success', this.autoScroll);
        button.classList.toggle('btn-secondary', !this.autoScroll);
    }

    cleanup() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }
        if (this.logUpdateInterval) {
            clearInterval(this.logUpdateInterval);
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.stressPulseUI = new StressPulseUI();
    
    window.addEventListener('beforeunload', () => {
        window.stressPulseUI.cleanup();
    });
    
    setTimeout(() => {
        window.stressPulseUI.addLog('🚀 StressPulse Web Interface initialized', 'success');
        window.stressPulseUI.addLog('📡 Ready to start stress testing', 'info');
    }, 500);
});

window.StressPulseUtils = {
    formatBytes: (bytes) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    },
    
    formatNumber: (num) => {
        return new Intl.NumberFormat().format(num);
    },
    
    formatDuration: (seconds) => {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        if (hours > 0) {
            return `${hours}h ${minutes}m ${secs}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${secs}s`;
        } else {
            return `${secs}s`;
        }
    }
}; 