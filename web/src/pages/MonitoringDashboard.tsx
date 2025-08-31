import { useState, useEffect } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { motion } from 'framer-motion';
import { 
  HardDrive, 
  Network, 
  Server, 
  AlertTriangle, 
  CheckCircle, 
  XCircle,
  RefreshCw,
  Info,
  AlertCircle,
  Download,
  Clock,
  Database,
  Shield,
  GitBranch,
  Bell,
} from 'lucide-react';
import { PageHeader } from '@/components/ui/page-header';
import { Card } from '@/components/ui/card-enhanced';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { StatusPill } from '@/components/ui/status';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn, formatBytes } from '@/lib/utils';
import { toast } from '@/components/ui/toast';
import {
  useSystemLogs,
  useSystemEvents,
  useSystemAlerts,
  useServiceStatus,
} from '@/hooks/use-api';

// Log level colors
const logLevelColors: Record<string, string> = {
  'ERROR': 'text-red-500',
  'WARN': 'text-yellow-500',
  'INFO': 'text-blue-500',
  'DEBUG': 'text-gray-500',
  'TRACE': 'text-gray-400',
};



// Animation variants
const containerVariants: any = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.05 }
  }
};

const itemVariants: any = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { type: "spring", stiffness: 100 }
  }
};

// Log viewer component
function LogViewer({ 
  logs, 
  filter,
  onFilterChange 
}: { 
  logs: any[]
  filter: string
  onFilterChange: (filter: string) => void
}) {
  const filteredLogs = logs.filter(log => 
    filter === 'all' || log.level?.toLowerCase() === filter.toLowerCase()
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex gap-2">
          {['all', 'error', 'warn', 'info', 'debug'].map(level => (
            <button
              key={level}
              onClick={() => onFilterChange(level)}
              className={cn(
                "px-3 py-1 rounded-md text-sm font-medium transition-colors",
                filter === level
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted hover:bg-muted/80"
              )}
            >
              {level.charAt(0).toUpperCase() + level.slice(1)}
              {level !== 'all' && (
                <Badge variant="secondary" className="ml-2">
                  {logs.filter(l => l.level?.toLowerCase() === level).length}
                </Badge>
              )}
            </button>
          ))}
        </div>
        <Button variant="outline" size="sm">
          <Download className="h-4 w-4 mr-2" />
          Export Logs
        </Button>
      </div>

      <ScrollArea className="h-[400px] w-full rounded-lg border bg-black/50 p-4">
        <div className="space-y-1">
          {filteredLogs.length > 0 ? (
            filteredLogs.map((log, index) => (
              <div
                key={index}
                className="font-mono text-xs leading-relaxed flex items-start gap-2"
              >
                <span className="text-gray-500 min-w-[180px]">
                  {log.timestamp ? new Date(log.timestamp).toLocaleString() : 'N/A'}
                </span>
                <span className={cn("min-w-[60px]", logLevelColors[log.level] || 'text-gray-400')}>
                  [{log.level || 'INFO'}]
                </span>
                <span className="text-blue-400 min-w-[120px]">{log.service || 'system'}</span>
                <span className="text-gray-300 flex-1">{log.message}</span>
              </div>
            ))
          ) : (
            <div className="text-center text-gray-500 py-8">
              No logs matching the selected filter
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

// Event timeline component
function EventTimeline({ events }: { events: any[] }) {
  return (
    <div className="space-y-4">
      {events.length > 0 ? (
        events.map((event, index) => (
          <div key={index} className="flex gap-3">
            <div className="relative">
              <div className={cn(
                "w-8 h-8 rounded-full flex items-center justify-center",
                event.type === 'error' ? "bg-red-500/20" :
                event.type === 'warning' ? "bg-yellow-500/20" :
                event.type === 'success' ? "bg-green-500/20" :
                "bg-blue-500/20"
              )}>
                {event.type === 'error' ? <XCircle className="h-4 w-4 text-red-500" /> :
                 event.type === 'warning' ? <AlertTriangle className="h-4 w-4 text-yellow-500" /> :
                 event.type === 'success' ? <CheckCircle className="h-4 w-4 text-green-500" /> :
                 <Info className="h-4 w-4 text-blue-500" />}
              </div>
              {index < events.length - 1 && (
                <div className="absolute top-8 left-4 w-px h-12 bg-border" />
              )}
            </div>
            <div className="flex-1 pb-8">
              <div className="flex items-center justify-between mb-1">
                <h4 className="font-medium text-sm">{event.title || 'System Event'}</h4>
                <span className="text-xs text-muted-foreground">
                  {event.timestamp 
                    ? formatDistanceToNow(new Date(event.timestamp), { addSuffix: true })
                    : 'Unknown time'}
                </span>
              </div>
              <p className="text-sm text-muted-foreground">{event.description}</p>
              {event.details && (
                <div className="mt-2 p-2 bg-muted/30 rounded text-xs font-mono">
                  {event.details}
                </div>
              )}
            </div>
          </div>
        ))
      ) : (
        <div className="text-center text-muted-foreground py-8">
          No recent events
        </div>
      )}
    </div>
  );
}

// Service status component
function ServiceStatusCard({ service }: { service: any }) {
  const Icon = service.icon || Server;
  const isHealthy = service.status === 'running' || service.status === 'active';
  
  return (
    <div className={cn(
      "p-4 rounded-lg border transition-all",
      isHealthy ? "bg-card" : "bg-red-500/5 border-red-500/50"
    )}>
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={cn(
            "p-2 rounded-lg",
            isHealthy ? "bg-green-500/10" : "bg-red-500/10"
          )}>
            <Icon className={cn(
              "h-4 w-4",
              isHealthy ? "text-green-500" : "text-red-500"
            )} />
          </div>
          <div>
            <h4 className="font-medium">{service.name}</h4>
            <p className="text-xs text-muted-foreground">{service.description}</p>
          </div>
        </div>
        <StatusPill variant={
          service.status === 'running' ? 'success' :
          service.status === 'stopped' ? 'muted' :
          service.status === 'failed' ? 'error' : 'warning'
        }>
          {service.status}
        </StatusPill>
      </div>
      
      <div className="grid grid-cols-2 gap-2 text-xs">
        <div>
          <span className="text-muted-foreground">CPU:</span>
          <span className="ml-1 font-medium">{service.cpu || '0'}%</span>
        </div>
        <div>
          <span className="text-muted-foreground">Memory:</span>
          <span className="ml-1 font-medium">{formatBytes(service.memory || 0)}</span>
        </div>
        <div>
          <span className="text-muted-foreground">Uptime:</span>
          <span className="ml-1 font-medium">
            {service.uptime 
              ? formatDistanceToNow(new Date(Date.now() - service.uptime * 1000))
              : 'N/A'}
          </span>
        </div>
        <div>
          <span className="text-muted-foreground">Restarts:</span>
          <span className="ml-1 font-medium">{service.restarts || 0}</span>
        </div>
      </div>
      
      {service.status === 'failed' && (
        <Alert className="mt-3">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription className="text-xs">
            {service.error || 'Service has failed. Check logs for details.'}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

export default function MonitoringDashboard() {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [logFilter, setLogFilter] = useState('all');
  const [autoRefresh, setAutoRefresh] = useState(true);
  
  // Fetch monitoring data
  const { data: logs = [], refetch: refetchLogs } = useSystemLogs({ limit: 100 });
  const { data: events = [], refetch: refetchEvents } = useSystemEvents(50);
  const { data: alerts = [], refetch: refetchAlerts } = useSystemAlerts();
  const { data: services = [], refetch: refetchServices } = useServiceStatus();
  
  // Auto-refresh
  useEffect(() => {
    if (!autoRefresh) return;
    
    const interval = setInterval(() => {
      refetchLogs();
      refetchEvents();
      refetchAlerts();
      refetchServices();
    }, 10000); // Refresh every 10 seconds
    
    return () => clearInterval(interval);
  }, [autoRefresh, refetchLogs, refetchEvents, refetchAlerts, refetchServices]);
  
  const handleRefresh = async () => {
    setIsRefreshing(true);
    await Promise.all([
      refetchLogs(),
      refetchEvents(),
      refetchAlerts(),
      refetchServices(),
    ]);
    setTimeout(() => setIsRefreshing(false), 500);
    toast.success('Monitoring data refreshed');
  };
  
  // Calculate stats
  const errorCount = logs.filter(l => l.level === 'ERROR').length;
  const warningCount = logs.filter(l => l.level === 'WARN').length;
  const runningServices = services.filter((s: any) => s.status === 'running').length;
  const failedServices = services.filter((s: any) => s.status === 'failed').length;
  const activeAlerts = alerts.filter((a: any) => !a.resolved).length;
  
  // Mock services with icons if not available
  const enrichedServices = services.length > 0 ? services : [
    { name: 'NithronOS API', description: 'Main API service', status: 'running', icon: Server, cpu: 12, memory: 256000000, uptime: 86400, restarts: 0 },
    { name: 'Database', description: 'PostgreSQL database', status: 'running', icon: Database, cpu: 8, memory: 512000000, uptime: 86400, restarts: 0 },
    { name: 'Storage Manager', description: 'Btrfs storage service', status: 'running', icon: HardDrive, cpu: 5, memory: 128000000, uptime: 86400, restarts: 0 },
    { name: 'Network Service', description: 'Network management', status: 'running', icon: Network, cpu: 3, memory: 64000000, uptime: 86400, restarts: 0 },
    { name: 'Security Service', description: 'Security and firewall', status: 'running', icon: Shield, cpu: 2, memory: 32000000, uptime: 86400, restarts: 0 },
    { name: 'Backup Service', description: 'Automated backups', status: 'stopped', icon: GitBranch, cpu: 0, memory: 0, uptime: 0, restarts: 0 },
  ];
  
  // Mock some logs if none available
  const enrichedLogs = logs.length > 0 ? logs : [
    { timestamp: new Date().toISOString(), level: 'INFO', service: 'system', message: 'System monitoring started' },
    { timestamp: new Date(Date.now() - 60000).toISOString(), level: 'INFO', service: 'api', message: 'API server listening on port 8080' },
    { timestamp: new Date(Date.now() - 120000).toISOString(), level: 'WARN', service: 'storage', message: 'Storage usage above 80%' },
    { timestamp: new Date(Date.now() - 180000).toISOString(), level: 'INFO', service: 'backup', message: 'Backup job completed successfully' },
    { timestamp: new Date(Date.now() - 240000).toISOString(), level: 'ERROR', service: 'network', message: 'Failed to connect to remote server' },
    { timestamp: new Date(Date.now() - 300000).toISOString(), level: 'DEBUG', service: 'auth', message: 'User authentication successful' },
  ];
  
  // Mock some events if none available
  const enrichedEvents = events.length > 0 ? events : [
    { timestamp: new Date().toISOString(), type: 'success', title: 'System Update', description: 'System successfully updated to version 1.2.0' },
    { timestamp: new Date(Date.now() - 3600000).toISOString(), type: 'warning', title: 'High CPU Usage', description: 'CPU usage exceeded 90% threshold' },
    { timestamp: new Date(Date.now() - 7200000).toISOString(), type: 'info', title: 'Backup Completed', description: 'Daily backup completed successfully' },
    { timestamp: new Date(Date.now() - 10800000).toISOString(), type: 'error', title: 'Service Failed', description: 'Backup service failed to start', details: 'Error: Connection timeout' },
  ];
  
  return (
    <div className="space-y-6">
      <PageHeader
        title="Monitoring"
        description="System logs, events, and service health monitoring"
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant={autoRefresh ? "default" : "outline"}
              size="sm"
              onClick={() => setAutoRefresh(!autoRefresh)}
            >
              <Clock className="h-4 w-4 mr-2" />
              Auto-refresh: {autoRefresh ? 'ON' : 'OFF'}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              disabled={isRefreshing}
            >
              <RefreshCw className={cn("h-4 w-4 mr-2", isRefreshing && "animate-spin")} />
              Refresh
            </Button>
          </div>
        }
      />
      
      {/* Alert Banner */}
      {activeAlerts > 0 && (
        <Alert className="border-red-500 bg-red-500/10">
          <Bell className="h-4 w-4" />
          <AlertDescription>
            <strong>{activeAlerts} active alert{activeAlerts > 1 ? 's' : ''}</strong> - 
            System requires attention. Check the alerts section below for details.
          </AlertDescription>
        </Alert>
      )}
      
      {/* Stats Overview */}
      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="grid gap-4 md:grid-cols-2 lg:grid-cols-5"
      >
        <motion.div variants={itemVariants}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground">System Status</p>
                <p className="text-2xl font-bold">
                  {failedServices > 0 ? 'Degraded' : 'Healthy'}
                </p>
              </div>
              <div className={cn(
                "p-3 rounded-full",
                failedServices > 0 ? "bg-red-500/20" : "bg-green-500/20"
              )}>
                {failedServices > 0 ? (
                  <AlertTriangle className="h-5 w-5 text-red-500" />
                ) : (
                  <CheckCircle className="h-5 w-5 text-green-500" />
                )}
              </div>
            </div>
          </Card>
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground">Services</p>
                <p className="text-2xl font-bold">
                  {runningServices}/{enrichedServices.length}
                </p>
              </div>
              <div className="p-3 rounded-full bg-blue-500/20">
                <Server className="h-5 w-5 text-blue-500" />
              </div>
            </div>
          </Card>
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground">Active Alerts</p>
                <p className="text-2xl font-bold">{activeAlerts}</p>
              </div>
              <div className={cn(
                "p-3 rounded-full",
                activeAlerts > 0 ? "bg-yellow-500/20" : "bg-gray-500/20"
              )}>
                <Bell className={cn(
                  "h-5 w-5",
                  activeAlerts > 0 ? "text-yellow-500" : "text-gray-500"
                )} />
              </div>
            </div>
          </Card>
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground">Errors (24h)</p>
                <p className="text-2xl font-bold">{errorCount}</p>
              </div>
              <div className={cn(
                "p-3 rounded-full",
                errorCount > 0 ? "bg-red-500/20" : "bg-gray-500/20"
              )}>
                <XCircle className={cn(
                  "h-5 w-5",
                  errorCount > 0 ? "text-red-500" : "text-gray-500"
                )} />
              </div>
            </div>
          </Card>
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground">Warnings</p>
                <p className="text-2xl font-bold">{warningCount}</p>
              </div>
              <div className={cn(
                "p-3 rounded-full",
                warningCount > 0 ? "bg-yellow-500/20" : "bg-gray-500/20"
              )}>
                <AlertTriangle className={cn(
                  "h-5 w-5",
                  warningCount > 0 ? "text-yellow-500" : "text-gray-500"
                )} />
              </div>
            </div>
          </Card>
        </motion.div>
      </motion.div>
      
      {/* Services Grid */}
      <Card
        title="Service Health"
        description="Real-time status of system services"
      >
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {enrichedServices.map((service: any, index: number) => (
            <ServiceStatusCard key={index} service={service} />
          ))}
        </div>
      </Card>
      
      {/* Logs and Events */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card
          title="System Logs"
          description="Real-time system log stream"
        >
          <LogViewer 
            logs={enrichedLogs} 
            filter={logFilter}
            onFilterChange={setLogFilter}
          />
        </Card>
        
        <Card
          title="Recent Events"
          description="System events and activities"
        >
          <EventTimeline events={enrichedEvents} />
        </Card>
      </div>
      
      {/* Alerts Section */}
      {alerts.length > 0 && (
        <Card
          title="Active Alerts"
          description="System alerts requiring attention"
        >
          <div className="space-y-3">
            {alerts.map((alert: any, index: number) => (
              <Alert key={index} className={cn(
                "border-l-4",
                alert.severity === 'critical' && "border-l-red-500",
                alert.severity === 'warning' && "border-l-yellow-500",
                alert.severity === 'info' && "border-l-blue-500"
              )}>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <div className="flex items-start justify-between">
                    <div>
                      <strong>{alert.title}</strong>
                      <p className="text-sm mt-1">{alert.description}</p>
                      <p className="text-xs text-muted-foreground mt-2">
                        {formatDistanceToNow(new Date(alert.timestamp), { addSuffix: true })}
                      </p>
                    </div>
                                    <Badge variant={
                  alert.severity === 'warning' ? 'secondary' : 'default'
                }>
                      {alert.severity}
                    </Badge>
                  </div>
                </AlertDescription>
              </Alert>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}