// mavlink-anywhere Dashboard — Alpine.js Application
// No build pipeline, no CDN, works fully offline.

function dashboard() {
    return {
        // State
        status: {},
        input: null,
        endpointList: [],
        rawConfig: '',
        systemInfo: {},
        diagnostics: {
            mavlink: null,
            alerts: [],
            docs: {},
            firewall: {},
        },
        profileBackups: [],
        profileFileName: '',
        profileImportMode: 'replace',
        profilePreview: null,
        profileDraft: null,
        logLines: [],
        logsPaused: false,
        templates: [],
        toasts: [],

        // UI state
        showRawConfig: false,
        showAddWizard: false,
        showEditModal: false,
        showProfilePreview: false,

        // Loading flags
        loading: {
            service: false,
            config: false,
            add: false,
            edit: false,
            profilePreview: false,
            profileApply: false,
            profileRestore: false,
        },

        // Wizard state
        wizardStep: 1,
        wizardType: '',
        wizardName: '',
        wizardAddr: '',
        wizardPort: 0,
        wizardMode: 'normal',

        // Edit state
        editingEp: null,
        editAddr: '',
        editPort: 0,
        editMode: 'normal',

        // SSE connection
        _logSource: null,

        // Computed
        get serviceStateClass() {
            const state = this.status.service?.state;
            if (state === 'running') return 'status-running';
            if (state === 'stopped') return 'status-stopped';
            return 'status-unknown';
        },

        get serviceStateLabel() {
            const state = this.status.service?.state;
            if (state === 'running') return 'Running';
            if (state === 'stopped') return 'Stopped';
            if (state === 'not_installed') return 'Not Installed';
            return 'Unknown';
        },

        get selectedTemplate() {
            return this.templates.find(t => t.id === this.wizardType) || null;
        },

        get listenerEndpoints() {
            return this.endpointList.filter((ep) => ep.category !== 'input' && ep.mode === 'server');
        },

        get outputEndpoints() {
            return this.endpointList.filter((ep) => ep.category !== 'input' && ep.mode !== 'server');
        },

        get mavlinkStateClass() {
            if (!this.diagnostics?.mavlink) return 'status-unknown';
            if (this.diagnostics.mavlink.active) return 'status-running';
            if (this.diagnostics.mavlink.available) return 'status-stopped';
            return 'status-unknown';
        },

        get mavlinkStateLabel() {
            if (!this.diagnostics?.mavlink) return 'Unknown';
            if (this.diagnostics.mavlink.active) return 'Active';
            if (this.diagnostics.mavlink.available) return 'No Data';
            return 'Unavailable';
        },

        get wizardPreview() {
            const name = this.wizardName || this.wizardType || 'endpoint';
            const addr = this.wizardAddr || this.selectedTemplate?.defaultAddress || '0.0.0.0';
            const port = this.wizardPort || this.selectedTemplate?.defaultPort || 14550;
            const mode = this.wizardMode || this.selectedTemplate?.mode || 'normal';
            return `[UdpEndpoint ${name}]\nMode=${mode}\nAddress=${addr}\nPort=${port}`;
        },

        get latestProfileBackup() {
            return this.profileBackups.length > 0 ? this.profileBackups[0] : null;
        },

        endpointModeLabel(ep) {
            if (ep.type === 'UartEndpoint') return 'UART';
            if (ep.mode === 'server') return 'SERVER';
            if (ep.mode === 'normal') return 'NORMAL';
            if (ep.type === 'UdpEndpoint') return 'UDP';
            return ep.type || 'ENDPOINT';
        },

        endpointModeClass(ep) {
            if (ep.mode === 'server') return 'mode-server';
            return 'mode-normal';
        },

        // Initialization
        async init() {
            await Promise.all([
                this.loadStatus(),
                this.loadEndpoints(),
                this.loadInput(),
                this.loadConfig(),
                this.loadSystemInfo(),
                this.loadDiagnostics(),
                this.loadProfileBackups(),
                this.loadTemplates(),
                this.loadRecentLogs(),
            ]);
            this.connectLogStream();

            // Poll status every 5 seconds
            setInterval(() => this.loadStatus(), 5000);
            setInterval(() => this.loadDiagnostics(), 8000);
        },

        // API calls
        async api(method, path, body) {
            const opts = {
                method,
                headers: { 'Content-Type': 'application/json' },
            };
            if (body) opts.body = JSON.stringify(body);
            const res = await fetch('/api/v1' + path, opts);
            const data = await res.json();
            if (!res.ok) {
                throw new Error(data.error || 'API error');
            }
            return data;
        },

        async loadStatus() {
            try {
                this.status = await this.api('GET', '/status');
            } catch (e) {
                // Silently fail — might be during startup
            }
        },

        async loadEndpoints() {
            try {
                const data = await this.api('GET', '/endpoints');
                this.endpointList = data.endpoints || [];
            } catch (e) {
                this.endpointList = [];
            }
        },

        async loadInput() {
            try {
                this.input = await this.api('GET', '/input');
            } catch (e) { /* ignore */ }
        },

        async loadConfig() {
            try {
                const data = await this.api('GET', '/config');
                this.rawConfig = data.raw || '';
            } catch (e) { /* ignore */ }
        },

        async loadSystemInfo() {
            try {
                this.systemInfo = await this.api('GET', '/system/info');
            } catch (e) { /* ignore */ }
        },

        async loadTemplates() {
            try {
                const data = await this.api('GET', '/templates');
                this.templates = data.templates || [];
            } catch (e) { /* ignore */ }
        },

        async loadProfileBackups() {
            try {
                const data = await this.api('GET', '/profiles/backups');
                this.profileBackups = data.backups || [];
            } catch (e) {
                this.profileBackups = [];
            }
        },

        async loadDiagnostics() {
            try {
                this.diagnostics = await this.api('GET', '/diagnostics');
            } catch (e) { /* ignore */ }
        },

        async loadRecentLogs() {
            try {
                const data = await this.api('GET', '/logs/recent?n=50');
                this.logLines = data.lines || [];
                this.$nextTick(() => this.scrollLogs());
            } catch (e) { /* ignore */ }
        },

        // Service actions
        async serviceAction(action) {
            this.loading.service = true;
            try {
                await this.api('POST', `/service/${action}`);
                this.toast('success', `Service ${action} successful`);
                setTimeout(() => this.loadStatus(), 1500);
            } catch (e) {
                this.toast('error', `Failed to ${action}: ${e.message}`);
            } finally {
                this.loading.service = false;
            }
        },

        // Raw config save
        async saveRawConfig() {
            this.loading.config = true;
            try {
                await this.api('PUT', '/config', { raw: this.rawConfig });
                this.toast('success', 'Config saved');
                await this.loadEndpoints();
                await this.loadInput();
                await this.loadDiagnostics();
            } catch (e) {
                this.toast('error', 'Failed to save: ' + e.message);
            } finally {
                this.loading.config = false;
            }
        },

        // Endpoint CRUD
        async toggleEp(ep) {
            try {
                await this.api('PATCH', `/endpoints/${ep.name}`, { enabled: !ep.enabled });
                this.toast('info', `${ep.name} ${ep.enabled ? 'disabled' : 'enabled'}`);
                await this.loadEndpoints();
                await this.loadConfig();
            } catch (e) {
                this.toast('error', e.message);
            }
        },

        async deleteEp(ep) {
            if (!confirm(`Delete endpoint "${ep.name}"? This will modify the config file.`)) return;
            try {
                await this.api('DELETE', `/endpoints/${ep.name}`);
                this.toast('success', `${ep.name} deleted`);
                await this.loadEndpoints();
                await this.loadConfig();
                await this.loadInput();
                await this.loadDiagnostics();
            } catch (e) {
                this.toast('error', e.message);
            }
        },

        editEndpoint(ep) {
            this.editingEp = ep;
            this.editAddr = ep.address;
            this.editPort = ep.port;
            this.editMode = ep.mode || 'normal';
            this.showEditModal = true;
        },

        async saveEdit() {
            this.loading.edit = true;
            try {
                await this.api('PUT', `/endpoints/${this.editingEp.name}`, {
                    name: this.editingEp.name,
                    address: this.editAddr,
                    port: this.editPort,
                    mode: this.editMode,
                    enabled: true,
                });
                this.toast('success', `${this.editingEp.name} updated`);
                this.showEditModal = false;
                await this.loadEndpoints();
                await this.loadConfig();
                await this.loadDiagnostics();
            } catch (e) {
                this.toast('error', e.message);
            } finally {
                this.loading.edit = false;
            }
        },

        // Add wizard
        async submitWizard() {
            this.loading.add = true;
            const name = this.wizardName || this.wizardType;
            const addr = this.wizardAddr || this.selectedTemplate?.defaultAddress || '0.0.0.0';
            const port = this.wizardPort || this.selectedTemplate?.defaultPort || 14550;
            const mode = this.wizardMode || this.selectedTemplate?.mode || 'normal';

            try {
                await this.api('POST', '/endpoints', {
                    name: name,
                    type: 'UdpEndpoint',
                    mode: mode,
                    address: addr,
                    port: port,
                });
                this.toast('success', `Endpoint "${name}" added`);
                this.resetWizard();
                this.showAddWizard = false;
                await this.loadEndpoints();
                await this.loadConfig();
                await this.loadDiagnostics();

                // Restart service to apply
                try {
                    await this.api('POST', '/service/restart');
                    this.toast('info', 'Service restarted to apply changes');
                    setTimeout(() => this.loadStatus(), 2000);
                } catch (e) { /* best effort */ }
            } catch (e) {
                this.toast('error', e.message);
            } finally {
                this.loading.add = false;
            }
        },

        resetWizard() {
            this.wizardStep = 1;
            this.wizardType = '';
            this.wizardName = '';
            this.wizardAddr = '';
            this.wizardPort = 0;
            this.wizardMode = 'normal';
        },

        // Profiles
        async exportProfile() {
            try {
                const data = await this.api('GET', '/profiles/export');
                const suggested = data.metadata?.profileName || 'routing-profile';
                const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
                const url = URL.createObjectURL(blob);
                const link = document.createElement('a');
                link.href = url;
                link.download = `${suggested}.json`;
                document.body.appendChild(link);
                link.click();
                link.remove();
                URL.revokeObjectURL(url);
                this.toast('success', 'Current routing profile exported');
            } catch (e) {
                this.toast('error', 'Failed to export profile: ' + e.message);
            }
        },

        openProfilePicker() {
            this.$refs.profileFile.click();
        },

        async onProfileFileSelected(event) {
            const file = event.target.files?.[0];
            if (!file) return;
            try {
                const text = await file.text();
                this.profileDraft = JSON.parse(text);
                this.profileFileName = file.name;
                await this.refreshProfilePreview();
                if (this.profilePreview) {
                    this.showProfilePreview = true;
                }
            } catch (e) {
                this.toast('error', 'Invalid profile file: ' + e.message);
            } finally {
                event.target.value = '';
            }
        },

        async refreshProfilePreview() {
            if (!this.profileDraft) return;
            this.loading.profilePreview = true;
            try {
                this.profilePreview = await this.api('POST', '/profiles/preview', {
                    mode: this.profileImportMode,
                    profile: this.profileDraft,
                });
            } catch (e) {
                this.profilePreview = null;
                this.toast('error', 'Profile preview failed: ' + e.message);
            } finally {
                this.loading.profilePreview = false;
            }
        },

        async applyProfileImport() {
            if (!this.profileDraft) return;
            this.loading.profileApply = true;
            try {
                const result = await this.api('POST', '/profiles/apply', {
                    mode: this.profileImportMode,
                    profile: this.profileDraft,
                });
                this.toast('success', 'Routing profile applied');
                this.showProfilePreview = false;
                this.profileDraft = null;
                this.profilePreview = null;
                await Promise.all([
                    this.loadStatus(),
                    this.loadEndpoints(),
                    this.loadInput(),
                    this.loadConfig(),
                    this.loadDiagnostics(),
                    this.loadProfileBackups(),
                ]);
                if (result?.backup?.createdAt) {
                    this.toast('info', `Backup created at ${this.formatTimestamp(result.backup.createdAt)}`);
                }
            } catch (e) {
                this.toast('error', 'Failed to apply profile: ' + e.message);
            } finally {
                this.loading.profileApply = false;
            }
        },

        async restoreLatestBackup() {
            if (!this.latestProfileBackup) {
                this.toast('error', 'No backup is available to restore');
                return;
            }
            if (!confirm('Restore the most recent routing backup and restart mavlink-router?')) return;
            this.loading.profileRestore = true;
            try {
                await this.api('POST', '/profiles/restore');
                this.toast('success', 'Latest routing backup restored');
                await Promise.all([
                    this.loadStatus(),
                    this.loadEndpoints(),
                    this.loadInput(),
                    this.loadConfig(),
                    this.loadDiagnostics(),
                    this.loadProfileBackups(),
                ]);
            } catch (e) {
                this.toast('error', 'Failed to restore backup: ' + e.message);
            } finally {
                this.loading.profileRestore = false;
            }
        },

        closeProfilePreview() {
            this.showProfilePreview = false;
            this.profilePreview = null;
            this.profileDraft = null;
            this.profileFileName = '';
            this.profileImportMode = 'replace';
        },

        formatTimestamp(value) {
            if (!value) return 'unknown';
            const date = new Date(value);
            if (Number.isNaN(date.getTime())) return value;
            return date.toLocaleString();
        },

        // Log streaming
        connectLogStream() {
            if (this._logSource) {
                this._logSource.close();
            }
            this._logSource = new EventSource('/api/v1/logs/stream');
            this._logSource.onmessage = (event) => {
                if (this.logsPaused) return;
                this.logLines.push(event.data);
                // Keep last 500 lines
                if (this.logLines.length > 500) {
                    this.logLines = this.logLines.slice(-500);
                }
                this.$nextTick(() => this.scrollLogs());
            };
            this._logSource.onerror = () => {
                // Reconnect after 3 seconds
                this._logSource.close();
                setTimeout(() => this.connectLogStream(), 3000);
            };
        },

        toggleLogs() {
            this.logsPaused = !this.logsPaused;
        },

        clearLogs() {
            this.logLines = [];
        },

        scrollLogs() {
            const el = this.$refs.logContainer;
            if (el) {
                el.scrollTop = el.scrollHeight;
            }
        },

        // Toast notifications
        toast(type, message) {
            const t = { type, message, visible: true };
            this.toasts.push(t);
            setTimeout(() => {
                t.visible = false;
                setTimeout(() => {
                    this.toasts = this.toasts.filter(x => x !== t);
                }, 300);
            }, 4000);
        },
    };
}

let dashboardRegistered = false;

function registerDashboardComponent() {
    if (dashboardRegistered) return;
    if (typeof window === 'undefined') return;
    if (!window.Alpine || typeof window.Alpine.data !== 'function') return;

    window.Alpine.data('dashboard', dashboard);
    window.dashboard = dashboard;
    dashboardRegistered = true;
}

if (typeof window !== 'undefined') {
    window.dashboard = dashboard;
}

if (typeof document !== 'undefined') {
    document.addEventListener('alpine:init', registerDashboardComponent, { once: true });
}

registerDashboardComponent();
