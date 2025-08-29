import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import { 
  Activity, Cpu, HardDrive, Network, Server, AlertTriangle, 
  CheckCircle, XCircle, TrendingUp, TrendingDown 
} from 'lucide-react';
import { monitorApi } from '@/api/monitor';
import type { MonitorOverview, TimeSeries } from '@/api/monitor.types';
import { formatBytes } from '@/lib/utils';
import { LineChart, Line, ResponsiveContainer, YAxis } from 'recharts';

// Sparkline component for small charts
function Sparkline({ data, color = '#3b82f6', height = 40 }: { 
  data: number[]; 
  color?: string; 
  height?: number;
}) {
  const chartData = data.map((value, index) => ({ index, value }));
  
  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={chartData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
        <YAxis hide />
        <Line 
          type="monotone" 
          dataKey="value" 
          stroke={color} 
          strokeWidth={1.5}
          dot={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

// Metric card component
function MetricCard({ 
  title, 
  value, 
  unit, 
  icon: Icon, 
  trend, 
  sparkline,
  color = 'primary',
  alert = false
}: {
  title: string;
  value: string | number;
  unit?: string;
  icon: any;
  trend?: number;
  sparkline?: number[];
  color?: 'primary' | 'success' | 'warning' | 'danger';
  alert?: boolean;
}) {
  const colorClasses = {
    primary: 'text-primary',
    success: 'text-green-500',
    warning: 'text-yellow-500',
    danger: 'text-red-500',
  };
  
  const sparklineColors = {
    primary: '#3b82f6',
    success: '#10b981',
    warning: '#f59e0b',
    danger: '#ef4444',
  };
  
  return (
    <div className={`bg-card rounded-lg p-4 ${alert ? 'ring-2 ring-red-500' : ''}`}>
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center space-x-2">
          <Icon className={`w-5 h-5 ${colorClasses[color]}`} />
          <h3 className="text-sm font-medium text-muted-foreground">{title}</h3>
        </div>
        {trend !== undefined && (
          <div className="flex items-center text-xs">
            {trend > 0 ? (
              <TrendingUp className="w-3 h-3 text-green-500 mr-1" />
            ) : (
              <TrendingDown className="w-3 h-3 text-red-500 mr-1" />
            )}
            <span className={trend > 0 ? 'text-green-500' : 'text-red-500'}>
              {Math.abs(trend).toFixed(1)}%
            </span>
          </div>
        )}
      </div>
      
      <div className="mb-2">
        <span className="text-2xl font-semibold">{value}</span>
        {unit && <span className="text-sm text-muted-foreground ml-1">{unit}</span>}
      </div>
      
      {sparkline && sparkline.length > 0 && (
        <div className="h-10">
          <Sparkline data={sparkline} color={sparklineColors[color]} />
        </div>
      )}
    </div>
  );
}

// Service status badge
function ServiceBadge({ service }: { service: any }) {
  const isHealthy = service.active && service.running;
  
  return (
    <div className="flex items-center justify-between p-2 bg-card rounded">
      <span className="text-sm font-medium">{service.name}</span>
      <div className="flex items-center space-x-2">
        {service.restart_count > 0 && (
          <span className="text-xs text-yellow-500">
            {service.restart_count} restarts
          </span>
        )}
        {isHealthy ? (
          <CheckCircle className="w-4 h-4 text-green-500" />
        ) : (
          <XCircle className="w-4 h-4 text-red-500" />
        )}
      </div>
    </div>
  );
}

export default function MonitoringDashboard() {
  const [timeRange, setTimeRange] = useState('1h');
  const [cpuHistory, setCpuHistory] = useState<number[]>([]);
  const [memHistory, setMemHistory] = useState<number[]>([]);
  const [diskHistory, setDiskHistory] = useState<number[]>([]);
  const [netHistory, setNetHistory] = useState<number[]>([]);
  
  // Fetch overview data
  const { data: overview, isLoading } = useQuery({
    queryKey: ['monitor-overview'],
    queryFn: () => monitorApi.getOverview(),
    refetchInterval: 5000, // Refresh every 5 seconds
  });
  
  // Fetch CPU history
  useQuery({
    queryKey: ['cpu-history', timeRange],
    queryFn: async () => {
      const endTime = new Date();
      const startTime = new Date();
      
      switch (timeRange) {
        case '1h':
          startTime.setHours(startTime.getHours() - 1);
          break;
        case '6h':
          startTime.setHours(startTime.getHours() - 6);
          break;
        case '24h':
          startTime.setHours(startTime.getHours() - 24);
          break;
        case '7d':
          startTime.setDate(startTime.getDate() - 7);
          break;
      }
      
      const result = await monitorApi.queryTimeSeries({
        metric: 'cpu',
        start_time: startTime.toISOString(),
        end_time: endTime.toISOString(),
      });
      
      if (result.data_points) {
        setCpuHistory(result.data_points.map(p => p.value));
      }
      
      return result;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });
  
  // Fetch memory history
  useQuery({
    queryKey: ['memory-history', timeRange],
    queryFn: async () => {
      const endTime = new Date();
      const startTime = new Date();
      startTime.setHours(startTime.getHours() - 1);
      
      const result = await monitorApi.queryTimeSeries({
        metric: 'memory',
        start_time: startTime.toISOString(),
        end_time: endTime.toISOString(),
      });
      
      if (result.data_points) {
        setMemHistory(result.data_points.map(p => p.value));
      }
      
      return result;
    },
    refetchInterval: 30000,
  });
  
  // Calculate alert status
  const hasAlerts = overview?.alerts_active > 0;
  const cpuAlert = overview?.system.cpu.usage_percent > 90;
  const memAlert = overview?.system.memory.used_percent > 90;
  const diskAlert = overview?.disks.some(d => d.used_percent > 85);
  
  // Calculate uptime string
  const uptimeString = overview 
    ? formatDistanceToNow(new Date(Date.now() - overview.system.uptime_seconds * 1000))
    : 'N/A';
  
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-muted-foreground">Loading monitoring data...</div>
      </div>
    );
  }
  
  if (!overview) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-muted-foreground">No monitoring data available</div>
      </div>
    );
  }
  
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">System Monitoring</h1>
          <p className="text-muted-foreground">
            Real-time system metrics and health status
          </p>
        </div>
        
        <div className="flex items-center space-x-4">
          {/* Time range selector */}
          <select
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
            className="bg-card rounded px-3 py-1 text-sm"
          >
            <option value="1h">Last Hour</option>
            <option value="6h">Last 6 Hours</option>
            <option value="24h">Last 24 Hours</option>
            <option value="7d">Last 7 Days</option>
          </select>
          
          {/* Alert indicator */}
          {hasAlerts && (
            <div className="flex items-center space-x-2 px-3 py-1 bg-red-500/10 text-red-500 rounded">
              <AlertTriangle className="w-4 h-4" />
              <span className="text-sm font-medium">
                {overview.alerts_active} Active {overview.alerts_active === 1 ? 'Alert' : 'Alerts'}
              </span>
            </div>
          )}
        </div>
      </div>
      
      {/* System overview cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="CPU Usage"
          value={overview.system.cpu.usage_percent.toFixed(1)}
          unit="%"
          icon={Cpu}
          sparkline={cpuHistory}
          color={cpuAlert ? 'danger' : 'primary'}
          alert={cpuAlert}
        />
        
        <MetricCard
          title="Memory Usage"
          value={overview.system.memory.used_percent.toFixed(1)}
          unit="%"
          icon={Activity}
          sparkline={memHistory}
          color={memAlert ? 'danger' : 'primary'}
          alert={memAlert}
        />
        
        <MetricCard
          title="Disk Usage"
          value={overview.disks[0]?.used_percent.toFixed(1) || 0}
          unit="%"
          icon={HardDrive}
          sparkline={diskHistory}
          color={diskAlert ? 'danger' : 'primary'}
          alert={diskAlert}
        />
        
        <MetricCard
          title="System Load"
          value={overview.system.load.load1.toFixed(2)}
          unit=""
          icon={Server}
          color="primary"
        />
      </div>
      
      {/* Additional metrics */}
      <div className="grid gap-4 md:grid-cols-3">
        {/* Uptime */}
        <div className="bg-card rounded-lg p-4">
          <h3 className="text-sm font-medium text-muted-foreground mb-2">System Uptime</h3>
          <p className="text-xl font-semibold">{uptimeString}</p>
        </div>
        
        {/* Swap usage */}
        <div className="bg-card rounded-lg p-4">
          <h3 className="text-sm font-medium text-muted-foreground mb-2">Swap Usage</h3>
          <p className="text-xl font-semibold">
            {formatBytes(overview.system.memory.swap_used)} / {formatBytes(overview.system.memory.swap_total)}
          </p>
          <div className="mt-2 w-full bg-muted rounded-full h-2">
            <div 
              className="bg-primary h-2 rounded-full"
              style={{ width: `${overview.system.memory.swap_percent}%` }}
            />
          </div>
        </div>
        
        {/* Temperature */}
        {overview.system.cpu.temperature && (
          <div className="bg-card rounded-lg p-4">
            <h3 className="text-sm font-medium text-muted-foreground mb-2">CPU Temperature</h3>
            <p className="text-xl font-semibold">
              {overview.system.cpu.temperature.toFixed(1)}°C
            </p>
          </div>
        )}
      </div>
      
      {/* Services status */}
      <div className="bg-card rounded-lg p-4">
        <h3 className="text-sm font-medium mb-3">Service Health</h3>
        <div className="grid gap-2 md:grid-cols-2 lg:grid-cols-3">
          {overview.services.map((service) => (
            <ServiceBadge key={service.name} service={service} />
          ))}
        </div>
      </div>
      
      {/* Disk details */}
      <div className="bg-card rounded-lg p-4">
        <h3 className="text-sm font-medium mb-3">Storage Devices</h3>
        <div className="space-y-3">
          {overview.disks.map((disk) => (
            <div key={disk.device} className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-2">
                  <HardDrive className="w-4 h-4 text-muted-foreground" />
                  <span className="text-sm font-medium">{disk.device}</span>
                  {disk.mount_point && (
                    <span className="text-xs text-muted-foreground">({disk.mount_point})</span>
                  )}
                </div>
                <div className="mt-1 flex items-center space-x-4 text-xs text-muted-foreground">
                  <span>{formatBytes(disk.used)} / {formatBytes(disk.total)}</span>
                  {disk.temperature && <span>{disk.temperature}°C</span>}
                  {disk.smart_status && (
                    <span className={disk.smart_status === 'PASSED' ? 'text-green-500' : 'text-red-500'}>
                      SMART: {disk.smart_status}
                    </span>
                  )}
                </div>
              </div>
              <div className="text-right">
                <span className={`text-sm font-medium ${disk.used_percent > 85 ? 'text-red-500' : ''}`}>
                  {disk.used_percent.toFixed(1)}%
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
      
      {/* Network interfaces */}
      <div className="bg-card rounded-lg p-4">
        <h3 className="text-sm font-medium mb-3">Network Interfaces</h3>
        <div className="grid gap-3 md:grid-cols-2">
          {overview.network.map((iface) => (
            <div key={iface.interface} className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Network className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm font-medium">{iface.interface}</span>
                <span className={`text-xs px-1 rounded ${
                  iface.link_state === 'up' ? 'bg-green-500/10 text-green-500' : 'bg-red-500/10 text-red-500'
                }`}>
                  {iface.link_state}
                </span>
              </div>
              <div className="text-xs text-muted-foreground">
                ↓ {formatBytes(iface.rx_bytes)} ↑ {formatBytes(iface.tx_bytes)}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
