// MQTT Dashboard JavaScript

class MQTTDashboard {
    constructor() {
        this.charts = {};
        this.data = {
            messageInflow: [],
            messageOutflow: [],
            messageDropped: [],
            connections: [],
            topics: [],
            subscriptions: []
        };
        this.timeRange = '1h';
        this.updateInterval = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.initializeCharts();
        this.startDataUpdates();
        this.updateKPICards();
    }

    setupEventListeners() {
        // 时间范围选择器
        document.getElementById('timeRange').addEventListener('change', (e) => {
            this.timeRange = e.target.value;
            this.updateCharts();
        });

        // 刷新按钮
        document.querySelector('.btn-refresh').addEventListener('click', () => {
            this.refreshData();
        });

        // 侧边栏导航 - 现在使用真实页面跳转，不需要特殊处理

        // 了解更多按钮
        document.querySelector('.btn-learn-more').addEventListener('click', () => {
            alert('MQTT Serverless 功能即将推出！');
        });
    }

    initializeCharts() {
        // 初始化所有图表
        this.createChart('messageInflowChart', '消息流入', '#3b82f6');
        this.createChart('messageOutflowChart', '消息发出', '#8b5cf6');
        this.createChart('messageDroppedChart', '消息丢弃', '#6b7280');
        this.createChart('connectionsChart', '连接数', '#10b981');
        this.createChart('topicsChart', '主题数', '#f59e0b');
        this.createChart('subscriptionsChart', '订阅数', '#06b6d4');

        // 初始化小图表
        this.createMiniChart('inflowChart', '#3b82f6');
        this.createMiniChart('outflowChart', '#8b5cf6');
    }

    createChart(canvasId, label, color) {
        const ctx = document.getElementById(canvasId).getContext('2d');
        
        this.charts[canvasId] = new Chart(ctx, {
            type: 'line',
            data: {
                labels: this.generateTimeLabels(),
                datasets: [{
                    label: label,
                    data: this.generateRandomData(20, 25000, 30000),
                    borderColor: color,
                    backgroundColor: color + '20',
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0,
                    pointHoverRadius: 4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            color: '#374151'
                        },
                        ticks: {
                            color: '#9ca3af',
                            font: {
                                size: 10
                            }
                        }
                    },
                    y: {
                        display: true,
                        grid: {
                            color: '#374151'
                        },
                        ticks: {
                            color: '#9ca3af',
                            font: {
                                size: 10
                            }
                        }
                    }
                },
                interaction: {
                    intersect: false,
                    mode: 'index'
                }
            }
        });
    }

    createMiniChart(canvasId, color) {
        const ctx = document.getElementById(canvasId).getContext('2d');
        
        new Chart(ctx, {
            type: 'line',
            data: {
                labels: this.generateTimeLabels(10),
                datasets: [{
                    data: this.generateRandomData(10, 2000, 4000),
                    borderColor: color,
                    backgroundColor: color + '20',
                    borderWidth: 1,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        display: false
                    },
                    y: {
                        display: false
                    }
                },
                elements: {
                    point: {
                        radius: 0
                    }
                }
            }
        });
    }

    generateTimeLabels(count = 20) {
        const labels = [];
        const now = new Date();
        
        for (let i = count - 1; i >= 0; i--) {
            const time = new Date(now.getTime() - i * 3 * 60 * 1000); // 每3分钟一个点
            labels.push(time.toLocaleTimeString('zh-CN', { 
                hour: '2-digit', 
                minute: '2-digit' 
            }));
        }
        
        return labels;
    }

    generateRandomData(count, min, max) {
        const data = [];
        for (let i = 0; i < count; i++) {
            data.push(Math.floor(Math.random() * (max - min + 1)) + min);
        }
        return data;
    }

    updateKPICards() {
        // 更新KPI卡片数据
        this.updateKPIValue('messageInflowRate', this.formatNumber(this.generateRandomValue(3000, 3500)) + ' 条/秒');
        this.updateKPIValue('messageOutflowRate', this.formatNumber(this.generateRandomValue(5500, 6000)) + ' 条/秒');
        this.updateKPIValue('totalConnections', this.formatNumber(this.generateRandomValue(100000, 110000)));
        this.updateKPIValue('onlineConnections', this.formatNumber(this.generateRandomValue(55000, 65000)));
        this.updateKPIValue('topicCount', this.formatNumber(this.generateRandomValue(24000, 26000)));
        this.updateKPIValue('subscriptionCount', this.formatNumber(this.generateRandomValue(115000, 120000)));
    }

    updateKPIValue(elementId, value) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = value;
        }
    }

    generateRandomValue(min, max) {
        return Math.floor(Math.random() * (max - min + 1)) + min;
    }

    formatNumber(num) {
        return num.toLocaleString('zh-CN');
    }

    updateCharts() {
        // 更新所有图表数据
        Object.keys(this.charts).forEach(chartId => {
            const chart = this.charts[chartId];
            const newData = this.generateRandomData(20, 20000, 30000);
            
            chart.data.datasets[0].data = newData;
            chart.update('none');
        });
    }

    refreshData() {
        // 显示加载状态
        const refreshBtn = document.querySelector('.btn-refresh i');
        refreshBtn.classList.add('fa-spin');
        
        // 模拟数据刷新
        setTimeout(() => {
            this.updateKPICards();
            this.updateCharts();
            refreshBtn.classList.remove('fa-spin');
        }, 1000);
    }

    startDataUpdates() {
        // 每5秒更新一次数据
        this.updateInterval = setInterval(() => {
            this.updateKPICards();
            this.updateCharts();
        }, 5000);
    }

    handleNavigation(href) {
        // 更新导航状态
        document.querySelectorAll('.sidebar-nav li').forEach(li => {
            li.classList.remove('active');
        });
        
        const activeLink = document.querySelector(`a[href="${href}"]`);
        if (activeLink) {
            activeLink.closest('li').classList.add('active');
        }

        // 更新面包屑
        const breadcrumb = document.querySelector('.breadcrumb span');
        switch (href) {
            case '#overview':
                breadcrumb.textContent = '概览';
                break;
            case '#connections':
                breadcrumb.textContent = '连接';
                break;
            case '#topics':
                breadcrumb.textContent = '主题';
                break;
            case '#messages':
                breadcrumb.textContent = '消息';
                break;
            case '#logs':
                breadcrumb.textContent = '日志';
                break;
        }

        // 这里可以添加页面切换逻辑
        console.log('导航到:', href);
    }

    // 模拟实时数据更新
    simulateRealTimeData() {
        setInterval(() => {
            // 随机更新一些KPI值
            const kpiElements = [
                'messageInflowRate',
                'messageOutflowRate',
                'onlineConnections'
            ];
            
            const randomElement = kpiElements[Math.floor(Math.random() * kpiElements.length)];
            const currentValue = document.getElementById(randomElement).textContent;
            
            if (randomElement === 'onlineConnections') {
                const newValue = this.generateRandomValue(55000, 65000);
                this.updateKPIValue(randomElement, this.formatNumber(newValue));
            } else {
                const newValue = this.generateRandomValue(3000, 4000);
                this.updateKPIValue(randomElement, this.formatNumber(newValue) + ' 条/秒');
            }
        }, 3000);
    }

    // 添加连接状态指示器
    addConnectionStatus() {
        const statusDot = document.querySelector('.status-dot');
        if (statusDot) {
            // 模拟连接状态变化
            setInterval(() => {
                statusDot.style.backgroundColor = Math.random() > 0.1 ? '#4ade80' : '#ef4444';
            }, 10000);
        }
    }

    // 添加工具提示
    addTooltips() {
        const kpiCards = document.querySelectorAll('.kpi-card');
        kpiCards.forEach(card => {
            card.addEventListener('mouseenter', (e) => {
                const title = e.currentTarget.querySelector('h3').textContent;
                this.showTooltip(e, title);
            });
            
            card.addEventListener('mouseleave', () => {
                this.hideTooltip();
            });
        });
    }

    showTooltip(event, text) {
        const tooltip = document.createElement('div');
        tooltip.className = 'tooltip-content';
        tooltip.textContent = text;
        tooltip.style.cssText = `
            position: absolute;
            background: #374151;
            color: white;
            padding: 8px 12px;
            border-radius: 6px;
            font-size: 12px;
            z-index: 1000;
            pointer-events: none;
        `;
        
        document.body.appendChild(tooltip);
        
        const rect = event.currentTarget.getBoundingClientRect();
        tooltip.style.left = rect.left + 'px';
        tooltip.style.top = (rect.top - tooltip.offsetHeight - 8) + 'px';
        
        this.currentTooltip = tooltip;
    }

    hideTooltip() {
        if (this.currentTooltip) {
            this.currentTooltip.remove();
            this.currentTooltip = null;
        }
    }

    // 销毁方法
    destroy() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }
        
        Object.values(this.charts).forEach(chart => {
            chart.destroy();
        });
    }
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    const dashboard = new MQTTDashboard();
    
    // 启动实时数据模拟
    dashboard.simulateRealTimeData();
    dashboard.addConnectionStatus();
    dashboard.addTooltips();
    
    // 将dashboard实例挂载到window对象，方便调试
    window.mqttDashboard = dashboard;
});

// 添加一些额外的交互功能
document.addEventListener('DOMContentLoaded', () => {
    // 添加键盘快捷键
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey && e.key === 'r') {
            e.preventDefault();
            document.querySelector('.btn-refresh').click();
        }
    });

    // 移动端适配 - 顶部导航栏
    if (window.innerWidth <= 768) {
        // 可以在这里添加移动端特定的功能
        console.log('Mobile view detected');
    }
});
