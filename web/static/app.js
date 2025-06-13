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
        console.log('Initializing StressPulse UI...');
        this.bindEvents();
        this.loadConfiguration();
        this.startMetricsUpdate();
        this.startLogUpdate();
        this.initAgents();
        console.log('StressPulse UI initialization complete');
    }

    bindEvents() {
        console.log('Binding events...');
        document.getElementById('start-all').addEventListener('click', () => this.startAll());
        document.getElementById('stop-all').addEventListener('click', () => this.stopAll());
        document.getElementById('restart-all').addEventListener('click', () => this.restartAll());
        document.getElementById('save-config').addEventListener('click', () => this.saveConfiguration());

        document.getElementById('clear-logs').addEventListener('click', () => this.clearLogs());
        document.getElementById('toggle-autoscroll').addEventListener('click', () => this.toggleAutoScroll());

        document.getElementById('add-agent').addEventListener('click', () => this.addAgent());
        document.getElementById('start-all-agents').addEventListener('click', () => this.startAllAgents());
        document.getElementById('stop-all-agents').addEventListener('click', () => this.stopAllAgents());
        document.getElementById('refresh-agents').addEventListener('click', () => this.refreshAgents());

        this.bindFormEvents();
        console.log('Events bound successfully');
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

    initAgents() {
        console.log('Initializing agents...');
        this.agents = new Map();
        this.loadAgents();
        this.startAgentUpdate();
        console.log('Agents initialization complete');
    }

    startAgentUpdate() {
        console.log('Starting agent update timer...');
        this.agentUpdateInterval = setInterval(() => {
            this.updateAgents();
        }, 5000);
    }

    async loadAgents() {
        console.log('Loading agents...');
        try {
            const response = await fetch('/api/agents');
            console.log('Agents API response status:', response.status);
            if (response.ok) {
                const agents = await response.json();
                console.log('Loaded agents:', agents);
                this.displayAgents(agents);
            } else {
                console.error('Failed to load agents, status:', response.status);
                this.addLog('⚠️ Failed to load agents', 'warning');
            }
        } catch (error) {
            console.error('Error loading agents:', error);
            this.addLog(`❌ Failed to load agents: ${error.message}`, 'error');
        }
    }

    async addAgent() {
        console.log('Adding agent...');
        const agentId = document.getElementById('agent-id').value.trim();
        const agentUrl = document.getElementById('agent-url').value.trim();

        console.log('Agent ID:', agentId, 'Agent URL:', agentUrl);

        if (!agentId || !agentUrl) {
            this.addLog('⚠️ Please provide both Agent ID and URL', 'warning');
            return;
        }

        try {
            console.log('Sending add agent request...');
            const response = await fetch('/api/agents/add', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    agent_id: agentId,
                    url: agentUrl
                })
            });

            console.log('Add agent response status:', response.status);
            console.log('Add agent response headers:', response.headers);

            if (response.ok) {
                const result = await response.json();
                console.log('Agent added successfully:', result);
                this.addLog(`✅ Agent "${agentId}" added successfully`, 'success');
                document.getElementById('agent-id').value = '';
                document.getElementById('agent-url').value = '';
                this.loadAgents();
            } else {
                const error = await response.text();
                console.error('Failed to add agent:', error);
                this.addLog(`❌ Failed to add agent: ${error}`, 'error');
            }
        } catch (error) {
            console.error('Error adding agent:', error);
            this.addLog(`❌ Error adding agent: ${error.message}`, 'error');
        }
    }

    async removeAgent(agentId) {
        try {
            const response = await fetch('/api/agents/remove', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    agent_id: agentId
                })
            });

            if (response.ok) {
                this.addLog(`🗑️ Agent "${agentId}" removed`, 'info');
                this.loadAgents();
            } else {
                const error = await response.text();
                this.addLog(`❌ Failed to remove agent: ${error}`, 'error');
            }
        } catch (error) {
            this.addLog(`❌ Error removing agent: ${error.message}`, 'error');
        }
    }

    async startAgent(agentId) {
        try {
            const config = this.getConfiguration();
            const agentConfig = this.convertToAgentConfig(config);

            const response = await fetch('/api/agents/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    agent_id: agentId,
                    config: agentConfig
                })
            });

            if (response.ok) {
                this.addLog(`🚀 Started load test on agent "${agentId}"`, 'success');
                this.updateAgentCard(agentId, { running: true });
            } else {
                const error = await response.text();
                this.addLog(`❌ Failed to start agent "${agentId}": ${error}`, 'error');
            }
        } catch (error) {
            this.addLog(`❌ Error starting agent "${agentId}": ${error.message}`, 'error');
        }
    }

    async stopAgent(agentId) {
        try {
            const response = await fetch('/api/agents/stop', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    agent_id: agentId
                })
            });

            if (response.ok) {
                this.addLog(`🛑 Stopped load test on agent "${agentId}"`, 'info');
                this.updateAgentCard(agentId, { running: false });
                
                document.querySelectorAll(`#agent-${agentId} .load-type-btn`).forEach(btn => {
                    btn.classList.remove('running');
                });
            } else {
                const error = await response.text();
                this.addLog(`❌ Failed to stop agent "${agentId}": ${error}`, 'error');
            }
        } catch (error) {
            this.addLog(`❌ Error stopping agent "${agentId}": ${error.message}`, 'error');
        }
    }

    async startAllAgents() {
        const agentIds = Array.from(this.agents.keys());
        if (agentIds.length === 0) {
            this.addLog('⚠️ No agents available to start', 'warning');
            return;
        }

        this.addLog(`🚀 Starting load tests on ${agentIds.length} agents...`, 'info');
        for (const agentId of agentIds) {
            await this.startAgent(agentId);
        }
    }

    async stopAllAgents() {
        const agentIds = Array.from(this.agents.keys());
        if (agentIds.length === 0) {
            this.addLog('⚠️ No agents available to stop', 'warning');
            return;
        }

        this.addLog(`🛑 Stopping load tests on ${agentIds.length} agents...`, 'info');
        for (const agentId of agentIds) {
            await this.stopAgent(agentId);
        }
    }

    async refreshAgents() {
        this.addLog('🔄 Refreshing agent status...', 'info');
        this.loadAgents();
    }

    convertToAgentConfig(config) {
        return {
            cpu: {
                enabled: config.cpu.enabled,
                load: config.cpu.load,
                pattern: config.cpu.pattern,
                drift: config.cpu.drift
            },
            memory: {
                enabled: config.memory.enabled,
                target: config.memory.target,
                pattern: config.memory.pattern
            },
            http: {
                enabled: config.http.enabled,
                url: config.http.url,
                rps: config.http.rps,
                method: config.http.method,
                pattern: config.http.pattern,
                headers: config.http.headers,
                body: config.http.body
            },
            websocket: {
                enabled: config.websocket.enabled,
                url: config.websocket.url,
                cps: config.websocket.cps,
                pattern: config.websocket.pattern,
                messageInterval: config.websocket.messageInterval,
                messageSize: config.websocket.messageSize
            },
            grpc: {
                enabled: config.grpc.enabled,
                address: config.grpc.address,
                rps: config.grpc.rps,
                method: config.grpc.method,
                pattern: config.grpc.pattern,
                secure: config.grpc.secure,
                service: config.grpc.service
            },
            fakeLogsEnabled: config.fakeLogsEnabled,
            fakeLogsType: config.fakeLogsType
        };
    }

    displayAgents(agents) {
        console.log('Displaying agents:', agents);
        const container = document.getElementById('agents-container');
        
        if (!container) {
            console.error('Agents container not found!');
            return;
        }
        
        container.innerHTML = '';

        this.agents.clear();

        if (Object.keys(agents).length === 0) {
            container.innerHTML = '<div class="no-agents">No agents connected. Add some agents to get started!</div>';
            console.log('No agents to display');
            return;
        }

        console.log('Creating agent cards...');
        for (const [agentId, agentInfo] of Object.entries(agents)) {
            console.log('Creating card for agent:', agentId, agentInfo);
            this.agents.set(agentId, agentInfo);
            const agentCard = this.createAgentCard(agentId, agentInfo);
            container.appendChild(agentCard);
        }
        console.log('Agent cards created successfully');
    }

    createAgentCard(agentId, agentInfo) {
        console.log('Creating agent card for:', agentId);
        console.log('Agent info:', agentInfo);
        console.log('is_healthy value:', agentInfo.is_healthy, 'type:', typeof agentInfo.is_healthy);
        
        const card = document.createElement('div');
        card.className = `agent-card ${agentInfo.is_healthy ? 'healthy' : 'unhealthy'}`;
        card.id = `agent-${agentId}`;

        const lastSeen = agentInfo.last_seen ? new Date(agentInfo.last_seen).toLocaleString() : 'Never';
        const status = agentInfo.is_healthy ? 'healthy' : 'unhealthy';
        
        console.log('Calculated status:', status);
        console.log('Card className:', card.className);

        card.innerHTML = `
            <div class="agent-header">
                <div class="agent-id">${agentId}</div>
                <div class="agent-status ${status}">${status.toUpperCase()}</div>
            </div>
            <div class="agent-url">${agentInfo.url}</div>
            <div class="agent-stats">
                <div class="agent-stat">
                    <span class="agent-stat-label">Status:</span>
                    <span class="agent-stat-value">${agentInfo.is_healthy ? 'Online' : 'Offline'}</span>
                </div>
                <div class="agent-stat">
                    <span class="agent-stat-label">Running:</span>
                    <span class="agent-stat-value" id="running-${agentId}">Unknown</span>
                </div>
            </div>
            
            <div class="agent-load-controls">
                <div class="load-type-buttons">
                    <div class="load-type-btn" data-type="cpu" onclick="window.stressPulseUI.toggleLoadConfig('${agentId}', 'cpu')">
                        💻 CPU
                    </div>
                    <div class="load-type-btn" data-type="memory" onclick="window.stressPulseUI.toggleLoadConfig('${agentId}', 'memory')">
                        🧠 Memory
                    </div>
                    <div class="load-type-btn" data-type="http" onclick="window.stressPulseUI.toggleLoadConfig('${agentId}', 'http')">
                        🌐 HTTP
                    </div>
                    <div class="load-type-btn" data-type="websocket" onclick="window.stressPulseUI.toggleLoadConfig('${agentId}', 'websocket')">
                        🔌 WebSocket
                    </div>
                    <div class="load-type-btn" data-type="grpc" onclick="window.stressPulseUI.toggleLoadConfig('${agentId}', 'grpc')">
                        ⚡ gRPC
                    </div>
                </div>
                
                <div id="config-panel-${agentId}" class="agent-config-panel">
                    <!-- Configuration panel content will be inserted here -->
                </div>
                
                <div class="agent-config-actions">
                    <button class="btn btn-primary btn-sm" onclick="window.stressPulseUI.startAgentLoad('${agentId}')">
                        🚀 Start Selected
                    </button>
                    <button class="btn btn-warning btn-sm" onclick="window.stressPulseUI.stopAgent('${agentId}')">
                        🛑 Stop All
                    </button>
                    <button class="btn btn-info btn-sm" onclick="window.stressPulseUI.getAgentStats('${agentId}')">
                        📊 Stats
                    </button>
                    <button class="btn btn-danger btn-sm" onclick="window.stressPulseUI.removeAgent('${agentId}')">
                        🗑️ Remove
                    </button>
                </div>
            </div>
            
            <div class="agent-last-seen">Last seen: ${lastSeen}</div>
        `;

        return card;
    }

    toggleLoadConfig(agentId, loadType) {
        console.log(`Toggling config for agent ${agentId}, type: ${loadType}`);
        
        const panel = document.getElementById(`config-panel-${agentId}`);
        const button = document.querySelector(`#agent-${agentId} .load-type-btn[data-type="${loadType}"]`);
        
        const isActive = button.classList.contains('active');
        
        document.querySelectorAll(`#agent-${agentId} .load-type-btn`).forEach(btn => {
            btn.classList.remove('active');
        });
        
        if (isActive) {
            panel.classList.remove('active');
            panel.innerHTML = '';
        } else {
            button.classList.add('active');
            panel.classList.add('active');
            this.showLoadConfig(agentId, loadType, panel);
        }
    }

    showLoadConfig(agentId, loadType, panel) {
        let configHTML = '';
        
        switch (loadType) {
            case 'cpu':
                configHTML = `
                    <div class="config-section-small">
                        <h4>💻 CPU Load Configuration</h4>
                        <div class="agent-config-grid">
                            <div class="form-group-inline">
                                <label>CPU Load (%)</label>
                                <input type="number" id="agent-${agentId}-cpu-load" min="0" max="100" value="50">
                            </div>
                            <div class="form-group-inline">
                                <label>Pattern</label>
                                <select id="agent-${agentId}-cpu-pattern">
                                    <option value="sine">Sine Wave</option>
                                    <option value="square">Square Wave</option>
                                    <option value="sawtooth">Sawtooth</option>
                                    <option value="random">Random</option>
                                </select>
                            </div>
                            <div class="form-group-inline">
                                <label>Drift (%)</label>
                                <input type="number" id="agent-${agentId}-cpu-drift" min="0" max="50" value="10">
                            </div>
                        </div>
                    </div>
                `;
                break;
                
            case 'memory':
                configHTML = `
                    <div class="config-section-small">
                        <h4>🧠 Memory Load Configuration</h4>
                        <div class="agent-config-grid">
                            <div class="form-group-inline">
                                <label>Target Size (MB)</label>
                                <input type="number" id="agent-${agentId}-memory-size" min="1" max="8192" value="100">
                            </div>
                            <div class="form-group-inline">
                                <label>Pattern</label>
                                <select id="agent-${agentId}-memory-pattern">
                                    <option value="constant">Constant</option>
                                    <option value="leak">Memory Leak</option>
                                    <option value="spike">Spike</option>
                                    <option value="cycle">Cycle</option>
                                    <option value="random">Random</option>
                                </select>
                            </div>
                        </div>
                    </div>
                `;
                break;
                
            case 'http':
                configHTML = `
                    <div class="config-section-small">
                        <h4>🌐 HTTP Load Configuration</h4>
                        <div class="agent-config-grid">
                            <div class="form-group-inline">
                                <label>Target URL</label>
                                <input type="url" id="agent-${agentId}-http-url" value="http://httpbin.org/get" placeholder="http://example.com/api">
                            </div>
                            <div class="form-group-inline">
                                <label>Target RPS</label>
                                <input type="number" id="agent-${agentId}-http-rps" min="1" max="10000" value="10">
                            </div>
                            <div class="form-group-inline">
                                <label>HTTP Method</label>
                                <select id="agent-${agentId}-http-method">
                                    <option value="GET">GET</option>
                                    <option value="POST">POST</option>
                                    <option value="PUT">PUT</option>
                                    <option value="DELETE">DELETE</option>
                                    <option value="PATCH">PATCH</option>
                                </select>
                            </div>
                            <div class="form-group-inline">
                                <label>Load Pattern</label>
                                <select id="agent-${agentId}-http-pattern">
                                    <option value="constant">Constant</option>
                                    <option value="spike">Spike</option>
                                    <option value="cycle">Cycle</option>
                                    <option value="ramp">Ramp</option>
                                    <option value="random">Random</option>
                                </select>
                            </div>
                        </div>
                    </div>
                `;
                break;
                
            case 'websocket':
                configHTML = `
                    <div class="config-section-small">
                        <h4>🔌 WebSocket Load Configuration</h4>
                        <div class="agent-config-grid">
                            <div class="form-group-inline">
                                <label>WebSocket URL</label>
                                <input type="url" id="agent-${agentId}-ws-url" value="ws://echo.websocket.org" placeholder="ws://example.com/ws">
                            </div>
                            <div class="form-group-inline">
                                <label>Connections per Second</label>
                                <input type="number" id="agent-${agentId}-ws-cps" min="1" max="1000" value="5">
                            </div>
                            <div class="form-group-inline">
                                <label>Connection Pattern</label>
                                <select id="agent-${agentId}-ws-pattern">
                                    <option value="constant">Constant</option>
                                    <option value="spike">Spike</option>
                                    <option value="cycle">Cycle</option>
                                    <option value="ramp">Ramp</option>
                                    <option value="random">Random</option>
                                </select>
                            </div>
                            <div class="form-group-inline">
                                <label>Message Interval (sec)</label>
                                <input type="number" id="agent-${agentId}-ws-interval" min="1" max="60" value="2">
                            </div>
                        </div>
                    </div>
                `;
                break;
                
            case 'grpc':
                configHTML = `
                    <div class="config-section-small">
                        <h4>⚡ gRPC Load Configuration</h4>
                        <div class="agent-config-grid">
                            <div class="form-group-inline">
                                <label>gRPC Address</label>
                                <input type="text" id="agent-${agentId}-grpc-addr" value="localhost:9000" placeholder="host:port">
                            </div>
                            <div class="form-group-inline">
                                <label>Target RPS</label>
                                <input type="number" id="agent-${agentId}-grpc-rps" min="1" max="10000" value="10">
                            </div>
                            <div class="form-group-inline">
                                <label>Method Type</label>
                                <select id="agent-${agentId}-grpc-method">
                                    <option value="health_check">Health Check</option>
                                    <option value="unary">Unary</option>
                                    <option value="server_stream">Server Stream</option>
                                    <option value="client_stream">Client Stream</option>
                                    <option value="bidi_stream">Bidirectional Stream</option>
                                </select>
                            </div>
                            <div class="form-group-inline">
                                <label>Load Pattern</label>
                                <select id="agent-${agentId}-grpc-pattern">
                                    <option value="constant">Constant</option>
                                    <option value="spike">Spike</option>
                                    <option value="cycle">Cycle</option>
                                    <option value="ramp">Ramp</option>
                                    <option value="random">Random</option>
                                </select>
                            </div>
                        </div>
                    </div>
                `;
                break;
        }
        
        panel.innerHTML = configHTML;
    }

    startAgentLoad(agentId) {
        console.log(`Starting load for agent: ${agentId}`);
        
        const activeButton = document.querySelector(`#agent-${agentId} .load-type-btn.active`);
        if (!activeButton) {
            this.addLog(`⚠️ Please select a load type for agent "${agentId}"`, 'warning');
            return;
        }
        
        const loadType = activeButton.getAttribute('data-type');
        const config = this.getAgentLoadConfig(agentId, loadType);
        
        this.startAgentWithConfig(agentId, config);
    }

    getAgentLoadConfig(agentId, loadType) {
        const config = {
            cpu: { enabled: false },
            memory: { enabled: false },
            http: { enabled: false },
            websocket: { enabled: false },
            grpc: { enabled: false },
            fakeLogsEnabled: false,
            fakeLogsType: 'generic'
        };
        
        switch (loadType) {
            case 'cpu':
                config.cpu = {
                    enabled: true,
                    load: parseInt(document.getElementById(`agent-${agentId}-cpu-load`).value),
                    pattern: document.getElementById(`agent-${agentId}-cpu-pattern`).value,
                    drift: parseInt(document.getElementById(`agent-${agentId}-cpu-drift`).value)
                };
                break;
                
            case 'memory':
                config.memory = {
                    enabled: true,
                    target: parseInt(document.getElementById(`agent-${agentId}-memory-size`).value),
                    pattern: document.getElementById(`agent-${agentId}-memory-pattern`).value
                };
                break;
                
            case 'http':
                config.http = {
                    enabled: true,
                    url: document.getElementById(`agent-${agentId}-http-url`).value,
                    rps: parseInt(document.getElementById(`agent-${agentId}-http-rps`).value),
                    method: document.getElementById(`agent-${agentId}-http-method`).value,
                    pattern: document.getElementById(`agent-${agentId}-http-pattern`).value,
                    headers: {},
                    body: ''
                };
                break;
                
            case 'websocket':
                config.websocket = {
                    enabled: true,
                    url: document.getElementById(`agent-${agentId}-ws-url`).value,
                    cps: parseInt(document.getElementById(`agent-${agentId}-ws-cps`).value),
                    pattern: document.getElementById(`agent-${agentId}-ws-pattern`).value,
                    messageInterval: parseInt(document.getElementById(`agent-${agentId}-ws-interval`).value),
                    messageSize: 256
                };
                break;
                
            case 'grpc':
                config.grpc = {
                    enabled: true,
                    address: document.getElementById(`agent-${agentId}-grpc-addr`).value,
                    rps: parseInt(document.getElementById(`agent-${agentId}-grpc-rps`).value),
                    method: document.getElementById(`agent-${agentId}-grpc-method`).value,
                    pattern: document.getElementById(`agent-${agentId}-grpc-pattern`).value,
                    secure: false,
                    service: ''
                };
                break;
        }
        
        return config;
    }

    async startAgentWithConfig(agentId, config) {
        try {
            console.log(`Starting agent ${agentId} with config:`, config);
            
            const response = await fetch('/api/agents/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    agent_id: agentId,
                    config: config
                })
            });

            if (response.ok) {
                this.addLog(`🚀 Started load test on agent "${agentId}"`, 'success');
                this.updateAgentCard(agentId, { running: true });
                
                const activeButton = document.querySelector(`#agent-${agentId} .load-type-btn.active`);
                if (activeButton) {
                    activeButton.classList.add('running');
                }
            } else {
                const error = await response.text();
                this.addLog(`❌ Failed to start agent "${agentId}": ${error}`, 'error');
            }
        } catch (error) {
            this.addLog(`❌ Error starting agent "${agentId}": ${error.message}`, 'error');
        }
    }

    updateAgentCard(agentId, updates) {
        const card = document.getElementById(`agent-${agentId}`);
        if (!card) return;

        if (updates.running !== undefined) {
            const runningElement = document.getElementById(`running-${agentId}`);
            if (runningElement) {
                runningElement.textContent = updates.running ? 'Yes' : 'No';
                runningElement.style.color = updates.running ? 'var(--success-color)' : 'var(--muted-text)';
            }
        }
    }

    async updateAgents() {
        try {
            const response = await fetch('/api/agents');
            if (response.ok) {
                const agents = await response.json();
                
                for (const [agentId, agentInfo] of Object.entries(agents)) {
                    const card = document.getElementById(`agent-${agentId}`);
                    if (card) {
                        const isHealthy = agentInfo.is_healthy;
                        card.className = `agent-card ${isHealthy ? 'healthy' : 'unhealthy'}`;
                        
                        const statusElement = card.querySelector('.agent-status');
                        if (statusElement) {
                            statusElement.textContent = isHealthy ? 'HEALTHY' : 'UNHEALTHY';
                            statusElement.className = `agent-status ${isHealthy ? 'healthy' : 'unhealthy'}`;
                        }
                        
                        const lastSeenElement = card.querySelector('.agent-last-seen');
                        if (lastSeenElement && agentInfo.last_seen) {
                            const lastSeen = new Date(agentInfo.last_seen).toLocaleString();
                            lastSeenElement.textContent = `Last seen: ${lastSeen}`;
                        }
                    }
                }
            }
        } catch (error) {
            console.error('Error updating agents:', error);
        }
    }

    async getAgentStats(agentId) {
        console.log(`Getting stats for agent: ${agentId}`);
        try {
            const response = await fetch(`/api/agents/stats?agent_id=${encodeURIComponent(agentId)}`);
            console.log('Agent stats response status:', response.status);
            
            if (response.ok) {
                const stats = await response.json();
                console.log('Agent stats received:', stats);
                this.showAgentStats(agentId, stats);
            } else {
                const error = await response.text();
                console.error('Failed to get agent stats:', error);
                this.addLog(`❌ Failed to get stats for agent "${agentId}": ${error}`, 'error');
            }
        } catch (error) {
            console.error('Error getting agent stats:', error);
            this.addLog(`❌ Error getting agent stats: ${error.message}`, 'error');
        }
    }

    showAgentStats(agentId, stats) {
        console.log('Showing agent stats:', agentId, stats);
        
        if (!stats || !stats.stats) {
            this.addLog(`📊 Agent "${agentId}": No statistics available (agent may not be running)`, 'info');
            return;
        }
        
        let statsText = `📊 Agent "${agentId}" Statistics:\n`;
        statsText += `Agent Status: ${stats.stats.agent_status || 'Unknown'}\n`;
        statsText += `Uptime: ${stats.stats.uptime || 'Unknown'}\n`;
        
        if (stats.stats.system) {
            const sys = stats.stats.system;
            statsText += `\n🖥️ System Information:\n`;
            statsText += `CPU Cores: ${sys.cpu_cores || 'Unknown'}\n`;
            statsText += `Memory Allocated: ${sys.memory_alloc || 0}MB\n`;
            statsText += `Memory System: ${sys.memory_sys || 0}MB\n`;
            statsText += `Memory Heap: ${sys.memory_heap || 0}MB\n`;
            statsText += `Goroutines: ${sys.goroutines || 0}\n`;
            statsText += `GC Runs: ${sys.gc_runs || 0}\n`;
        }
        
        let hasActiveTests = false;
        
        if (stats.stats.cpu) {
            hasActiveTests = true;
            const cpu = stats.stats.cpu;
            statsText += `\n💻 CPU Load Test:\n`;
            statsText += `Current Load: ${Math.round(cpu.CurrentLoad || 0)}%\n`;
            statsText += `Average Load: ${Math.round(cpu.AverageLoad || 0)}%\n`;
            statsText += `Total Samples: ${cpu.TotalSamples || 0}\n`;
            if (cpu.StartTime) {
                const startTime = new Date(cpu.StartTime).toLocaleTimeString();
                statsText += `Started: ${startTime}\n`;
            }
        }
        
        if (stats.stats.memory) {
            hasActiveTests = true;
            const mem = stats.stats.memory;
            statsText += `\n🧠 Memory Load Test:\n`;
            statsText += `Allocated: ${mem.AllocatedMB || 0}MB\n`;
            statsText += `Total Allocated: ${mem.TotalAllocated || 0}MB\n`;
            statsText += `Total Released: ${mem.TotalReleased || 0}MB\n`;
            statsText += `Allocation Count: ${mem.AllocationCount || 0}\n`;
            if (mem.StartTime) {
                const startTime = new Date(mem.StartTime).toLocaleTimeString();
                statsText += `Started: ${startTime}\n`;
            }
        }
        
        if (stats.stats.http) {
            hasActiveTests = true;
            const http = stats.stats.http;
            statsText += `\n🌐 HTTP Load Test:\n`;
            statsText += `Current RPS: ${http.CurrentRPS || 0}\n`;
            statsText += `Total Requests: ${http.TotalRequests || 0}\n`;
            statsText += `Success Requests: ${http.SuccessRequests || 0}\n`;
            statsText += `Failed Requests: ${http.FailedRequests || 0}\n`;
            statsText += `Success Rate: ${Math.round(http.SuccessRate || 0)}%\n`;
            statsText += `Average Response Time: ${http.AverageResponseTime || 'Unknown'}\n`;
            if (http.StartTime) {
                const startTime = new Date(http.StartTime).toLocaleTimeString();
                statsText += `Started: ${startTime}\n`;
            }
        }
        
        if (stats.stats.websocket) {
            hasActiveTests = true;
            const ws = stats.stats.websocket;
            statsText += `\n🔌 WebSocket Load Test:\n`;
            statsText += `Current CPS: ${ws.CurrentCPS || 0}\n`;
            statsText += `Active Connections: ${ws.ActiveConnections || 0}\n`;
            statsText += `Total Connections: ${ws.TotalConnections || 0}\n`;
            statsText += `Success Rate: ${Math.round(ws.SuccessRate || 0)}%\n`;
            if (ws.StartTime) {
                const startTime = new Date(ws.StartTime).toLocaleTimeString();
                statsText += `Started: ${startTime}\n`;
            }
        }
        
        if (stats.stats.grpc) {
            hasActiveTests = true;
            const grpc = stats.stats.grpc;
            statsText += `\n⚡ gRPC Load Test:\n`;
            statsText += `Current RPS: ${grpc.CurrentRPS || 0}\n`;
            statsText += `Total Requests: ${grpc.TotalRequests || 0}\n`;
            statsText += `Success Rate: ${Math.round(grpc.SuccessRate || 0)}%\n`;
            if (grpc.StartTime) {
                const startTime = new Date(grpc.StartTime).toLocaleTimeString();
                statsText += `Started: ${startTime}\n`;
            }
        }

        if (!hasActiveTests) {
            statsText += `\n⚠️ No active load tests running on this agent.\n`;
        }

        this.addLog(statsText, 'info');
    }

    cleanup() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }
        if (this.logUpdateInterval) {
            clearInterval(this.logUpdateInterval);
        }
        if (this.agentUpdateInterval) {
            clearInterval(this.agentUpdateInterval);
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