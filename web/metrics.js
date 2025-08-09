// MQTT Dashboard 指标页面 JavaScript

class MetricsDashboard {
    constructor() {
        this.metricsData = {};
        this.updateInterval = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.initializeMetrics();
        this.startDataUpdates();
    }

    setupEventListeners() {
        // 刷新按钮
        document.querySelector('#metricsRefresh').addEventListener('click', () => {
            this.refreshData();
        });

        // 导出按钮
        document.querySelector('#metricsExport').addEventListener('click', () => {
            this.exportMetrics();
        });

        // 搜索框
        document.querySelector('#metricsSearch').addEventListener('input', (e) => {
            this.filterMetrics(e.target.value);
        });

        // 指标区域折叠/展开
        document.querySelectorAll('.section-header').forEach(header => {
            header.addEventListener('click', (e) => {
                this.toggleSection(e.currentTarget);
            });
        });

        // 了解更多按钮
        document.querySelector('.btn-learn-more').addEventListener('click', () => {
            alert('MQTT Serverless 功能即将推出！');
        });
    }

    initializeMetrics() {
        // 初始化所有指标数据
        this.metricsData = {
            // 连接指标
            'client-connack': 187161966,
            'client-connack-hook': 187161966,
            'client-connect': 187162815,
            'client-connect-hook': 187162815,
            'client-connected': 163199776,
            'client-connected-hook': 163199776,
            'client-disconnected': 161881474,
            'client-disconnected-hook': 161881474,
            'client-subscribe': 190905723,
            'client-subscribe-hook': 190905723,
            'client-unsubscribe': 9692561,
            'client-unsubscribe-hook': 9692561,

            // 会话指标
            'session-created': 144593425,
            'session-created-hook': 144593425,
            'session-discarded': 85491103,
            'session-discarded-hook': 85491103,
            'session-resumed': 18606430,
            'session-resumed-hook': 18606430,
            'session-takenover': 18456235,
            'session-takenover-hook': 18456235,
            'session-terminated': 58346909,
            'session-terminated-hook': 58346909,

            // 认证与权限指标
            'authorization-allow': 4215753365,
            'authorization-cache-hit': 3275075620,
            'authorization-cache-miss': 940739882,
            'authorization-deny': 62134,
            'authorization-matched-allow': 939459301,
            'authorization-matched-deny': 61604,
            'authorization-nomatch': 1218974,
            'authorization-superuser': 0,
            'client-authorize': 940739881,
            'client-authorize-hook': 940739881,
            'client-authenticate': 186825183,
            'client-authenticate-hook': 186825183,
            'client-auth-anonymous': 0,

            // 消息传输指标
            'bytes-received': 1596593474,
            'bytes-received-alt': 274,

            // 消息数量指标
            'messages-acked': 250620670,
            'messages-acked-alt': 250620670,

            // 消息分发指标
            'delivery-dropped': 81805815,
            'delivery-dropped-alt': 81805815
        };

        // 更新所有指标显示
        this.updateAllMetrics();
    }

    updateAllMetrics() {
        Object.keys(this.metricsData).forEach(key => {
            this.updateMetricValue(key, this.metricsData[key]);
        });
    }

    updateMetricValue(elementId, value) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = this.formatNumber(value);
        }
    }

    formatNumber(num) {
        return num.toLocaleString('zh-CN');
    }

    generateRandomIncrement(baseValue) {
        // 生成一个小的随机增量，模拟真实数据变化
        const increment = Math.floor(Math.random() * 1000);
        return baseValue + increment;
    }

    updateMetricsData() {
        // 模拟数据更新
        Object.keys(this.metricsData).forEach(key => {
            this.metricsData[key] = this.generateRandomIncrement(this.metricsData[key]);
        });
        
        this.updateAllMetrics();
    }

    refreshData() {
        // 显示加载状态
        const refreshBtn = document.querySelector('.btn-refresh i');
        refreshBtn.classList.add('fa-spin');
        
        // 模拟数据刷新
        setTimeout(() => {
            this.updateMetricsData();
            refreshBtn.classList.remove('fa-spin');
        }, 1000);
    }

    startDataUpdates() {
        // 每10秒更新一次数据
        this.updateInterval = setInterval(() => {
            this.updateMetricsData();
        }, 10000);
    }

    toggleSection(header) {
        const section = header.closest('.metrics-section');
        const table = section.querySelector('.metrics-table');
        
        if (header.classList.contains('collapsed')) {
            // 展开
            header.classList.remove('collapsed');
            table.style.display = 'block';
            header.querySelector('.toggle-icon').classList.remove('fa-chevron-down');
            header.querySelector('.toggle-icon').classList.add('fa-chevron-up');
        } else {
            // 折叠
            header.classList.add('collapsed');
            table.style.display = 'none';
            header.querySelector('.toggle-icon').classList.remove('fa-chevron-up');
            header.querySelector('.toggle-icon').classList.add('fa-chevron-down');
        }
    }

    // 移除动态创建搜索框的功能，因为现在搜索框已经在HTML中
    addSearchFunctionality() {
        // 搜索框已经在HTML中，不需要动态创建
        console.log('搜索功能已通过HTML实现');
    }

    filterMetrics(searchTerm) {
        const metricRows = document.querySelectorAll('.metric-row');
        
        metricRows.forEach(row => {
            const metricName = row.querySelector('.metric-name').textContent.toLowerCase();
            const metricValue = row.querySelector('.metric-value').textContent.toLowerCase();
            const searchLower = searchTerm.toLowerCase();
            
            if (metricName.includes(searchLower) || metricValue.includes(searchLower)) {
                row.style.display = 'flex';
            } else {
                row.style.display = 'none';
            }
        });
    }

    // 添加指标排序功能
    addSortFunctionality() {
        const sectionHeaders = document.querySelectorAll('.section-header');
        
        sectionHeaders.forEach(header => {
            const sortBtn = document.createElement('button');
            sortBtn.innerHTML = '<i class="fas fa-sort"></i>';
            sortBtn.className = 'sort-btn';
            sortBtn.style.cssText = `
                background: none;
                border: none;
                color: #9ca3af;
                cursor: pointer;
                padding: 4px;
                margin-left: 8px;
                font-size: 12px;
            `;
            
            header.appendChild(sortBtn);
            
            sortBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.sortMetrics(header);
            });
        });
    }

    sortMetrics(header) {
        const section = header.closest('.metrics-section');
        const table = section.querySelector('.metrics-table');
        const rows = Array.from(table.querySelectorAll('.metric-row'));
        
        // 切换排序方向
        const isAscending = !table.dataset.sorted || table.dataset.sorted === 'desc';
        
        rows.sort((a, b) => {
            const aValue = parseInt(a.querySelector('.metric-value').textContent.replace(/,/g, ''));
            const bValue = parseInt(b.querySelector('.metric-value').textContent.replace(/,/g, ''));
            
            return isAscending ? aValue - bValue : bValue - aValue;
        });
        
        // 重新排列行
        rows.forEach(row => table.appendChild(row));
        
        // 更新排序状态
        table.dataset.sorted = isAscending ? 'asc' : 'desc';
        
        // 更新排序按钮图标
        const sortBtn = header.querySelector('.sort-btn i');
        sortBtn.className = isAscending ? 'fas fa-sort-up' : 'fas fa-sort-down';
    }

    // 添加指标导出功能
    addExportFunctionality() {
        const exportBtn = document.querySelector('.btn-export');
        if (exportBtn) {
            exportBtn.addEventListener('click', () => {
                this.exportMetrics();
            });
        }
    }

    exportMetrics() {
        const csvContent = this.generateCSV();
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        
        if (link.download !== undefined) {
            const url = URL.createObjectURL(blob);
            link.setAttribute('href', url);
            link.setAttribute('download', `mqtt-metrics-${new Date().toISOString().slice(0, 10)}.csv`);
            link.style.visibility = 'hidden';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        }
    }

    generateCSV() {
        const rows = [];
        rows.push(['指标名称', '数值', '更新时间']);
        
        const metricRows = document.querySelectorAll('.metric-row');
        metricRows.forEach(row => {
            const name = row.querySelector('.metric-name').textContent;
            const value = row.querySelector('.metric-value').textContent;
            const timestamp = new Date().toLocaleString('zh-CN');
            rows.push([name, value, timestamp]);
        });
        
        return rows.map(row => row.join(',')).join('\n');
    }

    // 销毁方法
    destroy() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    const metricsDashboard = new MetricsDashboard();
    
    // 添加额外功能
    metricsDashboard.addSearchFunctionality();
    metricsDashboard.addSortFunctionality();
    metricsDashboard.addExportFunctionality();
    
    // 将dashboard实例挂载到window对象，方便调试
    window.metricsDashboard = metricsDashboard;
});

// 添加键盘快捷键
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 'r') {
        e.preventDefault();
        document.querySelector('.btn-refresh').click();
    }
    
    if (e.ctrlKey && e.key === 'f') {
        e.preventDefault();
        const searchInput = document.querySelector('.metrics-search');
        if (searchInput) {
            searchInput.focus();
        }
    }
    
    if (e.ctrlKey && e.key === 'e') {
        e.preventDefault();
        const exportBtn = document.querySelector('.btn-export');
        if (exportBtn) {
            exportBtn.click();
        }
    }
});

// 移动端适配 - 顶部导航栏
document.addEventListener('DOMContentLoaded', () => {
    if (window.innerWidth <= 768) {
        // 可以在这里添加移动端特定的功能
        console.log('Mobile view detected');
    }
});
