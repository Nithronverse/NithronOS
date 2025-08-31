import { useState, useMemo } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Package, 
  Download, 
  Search, 
  Shield,
  Play,
  Square,
  RotateCw,
  ExternalLink,
  CheckCircle,
  AlertCircle,
  XCircle,
  Loader2,
  Grid3x3,
  List,
  ChevronRight,
  Server,
  Heart,
  Zap,
  Database,
  Code,
  Film,
  Gamepad,
  Home,
  Lock,
  Cloud,
  Info
} from 'lucide-react';
import { appsApi } from '../api/apps';
import type { CatalogEntry, InstalledApp } from '../api/apps.types';
import { cn } from '../lib/utils';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card-enhanced';
import { toast } from '@/components/ui/toast';
import { Badge } from '@/components/ui/badge';

// Category icons mapping
const categoryIcons: Record<string, any> = {
  'Media': Film,
  'Gaming': Gamepad,
  'Productivity': Zap,
  'Development': Code,
  'Database': Database,
  'Smart Home': Home,
  'Security': Lock,
  'Cloud': Cloud,
  'Health': Heart,
  'Utilities': Server,
};

// Animation variants
const containerVariants: any = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.05
    }
  }
};

const itemVariants: any = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      type: "spring",
      stiffness: 100
    }
  }
};

export function AppCatalog() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<'catalog' | 'installed'>('catalog');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [officialOnly, setOfficialOnly] = useState(false);
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Fetch catalog
  const { data: catalog, isLoading: catalogLoading } = useQuery({
    queryKey: ['apps', 'catalog'],
    queryFn: appsApi.getCatalog,
    refetchInterval: 60000, // Refresh every minute
  });

  // Fetch installed apps
  const { data: installedData, isLoading: installedLoading } = useQuery({
    queryKey: ['apps', 'installed'],
    queryFn: appsApi.getInstalledApps,
    refetchInterval: 10000, // Refresh every 10 seconds
  });

  // Sync catalogs mutation
  const syncCatalogsMutation = useMutation({
    mutationFn: async () => {
      setIsRefreshing(true);
      // Call sync API
      await appsApi.syncCatalogs();
      // Wait a bit for sync to complete
      await new Promise(resolve => setTimeout(resolve, 2000));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps', 'catalog'] });
      toast.success('Catalogs synced successfully');
      setIsRefreshing(false);
    },
    onError: (error) => {
      toast.error('Failed to sync catalogs');
      console.error('Sync error:', error);
      setIsRefreshing(false);
    }
  });

  const installedApps = installedData?.items || [];
  const installedAppIds = new Set(installedApps.map((app: InstalledApp) => app.id));

  // Get unique categories
  const categories = useMemo(() => {
    if (!catalog) return [];
    const cats = new Set<string>();
    catalog.entries.forEach((app: CatalogEntry) => {
      app.categories?.forEach((cat: string) => cats.add(cat));
    });
    return Array.from(cats).sort();
  }, [catalog]);

  // Filter catalog entries
  const filteredEntries = useMemo(() => {
    if (!catalog) return [];
    
    return catalog.entries.filter((app: CatalogEntry) => {
      // Search filter
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        if (
          !app.name.toLowerCase().includes(query) &&
          !app.description.toLowerCase().includes(query) &&
          !app.id.toLowerCase().includes(query)
        ) {
          return false;
        }
      }

      // Category filter
      if (selectedCategory && !app.categories?.includes(selectedCategory)) {
        return false;
      }

      // Official filter
      if (officialOnly && !(app as any).verified) {
        return false;
      }

      return true;
    });
  }, [catalog, searchQuery, selectedCategory, officialOnly]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'text-green-500';
      case 'stopped':
        return 'text-gray-500';
      case 'error':
      case 'unhealthy':
        return 'text-red-500';
      case 'starting':
      case 'stopping':
      case 'upgrading':
        return 'text-yellow-500';
      default:
        return 'text-gray-400';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="w-4 h-4" />;
      case 'stopped':
        return <XCircle className="w-4 h-4" />;
      case 'error':
      case 'unhealthy':
        return <AlertCircle className="w-4 h-4" />;
      case 'starting':
      case 'stopping':
      case 'upgrading':
        return <Loader2 className="w-4 h-4 animate-spin" />;
      default:
        return null;
    }
  };

  const handleAppAction = async (appId: string, action: 'start' | 'stop' | 'restart') => {
    try {
      switch (action) {
        case 'start':
          await appsApi.startApp(appId);
          break;
        case 'stop':
          await appsApi.stopApp(appId);
          break;
        case 'restart':
          await appsApi.restartApp(appId);
          break;
      }
      await queryClient.invalidateQueries({ queryKey: ['apps', 'installed'] });
      toast.success(`App ${action}ed successfully`);
    } catch (error) {
      console.error(`Failed to ${action} app:`, error);
      toast.error(`Failed to ${action} app`);
    }
  };

  const handleInstall = (appId: string) => {
    navigate(`/apps/install/${appId}`);
  };

  const AppCard = ({ app, isInstalled }: { app: CatalogEntry; isInstalled: boolean }) => {
    const CategoryIcon = app.categories?.[0] ? categoryIcons[app.categories[0]] || Package : Package;
    
    return (
      <motion.div variants={itemVariants}>
        <Card className="group hover:shadow-xl transition-all duration-300 overflow-hidden h-full">
          <div className="relative">
            {/* App Header with Gradient Background */}
            <div className={cn(
              "h-32 relative overflow-hidden",
              "bg-gradient-to-br",
              isInstalled ? "from-green-600 to-green-800" : "from-blue-600 to-purple-800"
            )}>
              <div className="absolute inset-0 bg-black/20" />
              <div className="absolute inset-0 flex items-center justify-center">
                <CategoryIcon className="w-16 h-16 text-white/80" />
              </div>
              {(app as any).verified && (
                <div className="absolute top-2 right-2">
                  <Badge className="bg-yellow-500 text-black">
                    <Shield className="w-3 h-3 mr-1" />
                    Verified
                  </Badge>
                </div>
              )}
              {isInstalled && (
                <div className="absolute top-2 left-2">
                  <Badge className="bg-green-500">
                    <CheckCircle className="w-3 h-3 mr-1" />
                    Installed
                  </Badge>
                </div>
              )}
            </div>
            
            {/* App Info */}
            <div className="p-4 space-y-3">
              <div>
                <h3 className="font-semibold text-lg line-clamp-1">{app.name}</h3>
                <p className="text-sm text-muted-foreground line-clamp-2 mt-1">
                  {app.description}
                </p>
              </div>
              
              {/* Categories */}
              <div className="flex flex-wrap gap-1">
                {app.categories?.map((cat) => (
                  <Badge key={cat} variant="secondary" className="text-xs">
                    {cat}
                  </Badge>
                ))}
              </div>
              
              {/* Version and Size */}
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>v{app.version}</span>
                {(app as any).repository && (
                  <a
                    href={(app as any).repository}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center hover:text-primary transition-colors"
                  >
                    <ExternalLink className="w-3 h-3 mr-1" />
                    Source
                  </a>
                )}
              </div>
              
              {/* Action Button */}
              <Button
                className="w-full"
                variant={isInstalled ? "secondary" : "default"}
                onClick={() => isInstalled ? navigate(`/apps/${app.id}`) : handleInstall(app.id)}
              >
                {isInstalled ? (
                  <>
                    <Info className="w-4 h-4 mr-2" />
                    Manage
                  </>
                ) : (
                  <>
                    <Download className="w-4 h-4 mr-2" />
                    Install
                  </>
                )}
              </Button>
            </div>
          </div>
        </Card>
      </motion.div>
    );
  };

  const InstalledAppCard = ({ app }: { app: InstalledApp }) => {
    const CategoryIcon = Package;
    
    return (
      <motion.div variants={itemVariants}>
        <Card className="group hover:shadow-xl transition-all duration-300 overflow-hidden h-full">
          <div className="relative">
            {/* Status Bar */}
            <div className={cn(
              "h-2",
              app.status === 'running' && "bg-green-500",
              app.status === 'stopped' && "bg-gray-500",
              app.status === 'error' && "bg-red-500",
              (app.status === 'starting' || app.status === 'stopping') && "bg-yellow-500 animate-pulse"
            )} />
            
            {/* App Content */}
            <div className="p-4 space-y-3">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/10 rounded-lg">
                    <CategoryIcon className="w-8 h-8 text-primary" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-lg">{app.name}</h3>
                    <div className={cn("flex items-center gap-1 text-sm", getStatusColor(app.status))}>
                      {getStatusIcon(app.status)}
                      <span className="capitalize">{app.status}</span>
                    </div>
                  </div>
                </div>
              </div>
              
              {/* App Info */}
              <div className="space-y-2 text-sm text-muted-foreground">
                <div className="flex items-center justify-between">
                  <span>Version</span>
                  <span className="font-medium text-foreground">{app.version}</span>
                </div>
                {app.ports && app.ports.length > 0 && (
                  <div className="flex items-center justify-between">
                    <span>Ports</span>
                    <span className="font-medium text-foreground">
                      {app.ports.join(', ')}
                    </span>
                  </div>
                )}
              </div>
              
              {/* Action Buttons */}
              <div className="flex gap-2">
                {app.status === 'stopped' ? (
                  <Button
                    size="sm"
                    variant="outline"
                    className="flex-1"
                    onClick={() => handleAppAction(app.id, 'start')}
                  >
                    <Play className="w-4 h-4 mr-1" />
                    Start
                  </Button>
                ) : app.status === 'running' ? (
                  <>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1"
                      onClick={() => handleAppAction(app.id, 'stop')}
                    >
                      <Square className="w-4 h-4 mr-1" />
                      Stop
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1"
                      onClick={() => handleAppAction(app.id, 'restart')}
                    >
                      <RotateCw className="w-4 h-4 mr-1" />
                      Restart
                    </Button>
                  </>
                ) : null}
                <Button
                  size="sm"
                  variant="default"
                  onClick={() => navigate(`/apps/${app.id}`)}
                >
                  <ChevronRight className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </div>
        </Card>
      </motion.div>
    );
  };

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-3 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl">
            <Package className="w-8 h-8 text-white" />
          </div>
          <div>
            <h1 className="text-3xl font-bold">App Catalog</h1>
            <p className="text-muted-foreground">
              Discover and install applications for your NithronOS
            </p>
          </div>
        </div>
        <Button
          onClick={() => syncCatalogsMutation.mutate()}
          disabled={isRefreshing || syncCatalogsMutation.isPending}
          variant="outline"
        >
          <RotateCw className={cn(
            "w-4 h-4 mr-2",
            (isRefreshing || syncCatalogsMutation.isPending) && "animate-spin"
          )} />
          Sync Catalogs
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex items-center justify-between">
        <div className="flex gap-1 p-1 bg-muted rounded-lg">
          <button
            onClick={() => setActiveTab('catalog')}
            className={cn(
              'px-4 py-2 rounded-md transition-all font-medium',
              activeTab === 'catalog'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Package className="w-4 h-4 inline-block mr-2" />
            Catalog
          </button>
          <button
            onClick={() => setActiveTab('installed')}
            className={cn(
              'px-4 py-2 rounded-md transition-all font-medium relative',
              activeTab === 'installed'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Download className="w-4 h-4 inline-block mr-2" />
            Installed
            {installedApps.length > 0 && (
              <Badge className="ml-2" variant="secondary">
                {installedApps.length}
              </Badge>
            )}
          </button>
        </div>
        
        {/* View Mode Toggle */}
        <div className="flex gap-1 p-1 bg-muted rounded-lg">
          <button
            onClick={() => setViewMode('grid')}
            className={cn(
              'p-2 rounded transition-all',
              viewMode === 'grid'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Grid3x3 className="w-4 h-4" />
          </button>
          <button
            onClick={() => setViewMode('list')}
            className={cn(
              'p-2 rounded transition-all',
              viewMode === 'list'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <List className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Filters */}
      {activeTab === 'catalog' && (
        <div className="space-y-4">
          {/* Search and toggles */}
          <div className="flex gap-4 items-center">
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search apps..."
                className="w-full pl-10 pr-4 py-2 bg-background border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            
            <Button
              variant={officialOnly ? "default" : "outline"}
              size="sm"
              onClick={() => setOfficialOnly(!officialOnly)}
            >
              <Shield className="w-4 h-4 mr-2" />
              Verified Only
            </Button>
          </div>

          {/* Category Pills */}
          {categories.length > 0 && (
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => setSelectedCategory(null)}
                className={cn(
                  'px-4 py-2 rounded-full text-sm font-medium transition-all',
                  selectedCategory === null
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted hover:bg-muted/80'
                )}
              >
                All Categories
              </button>
              {categories.map((cat) => {
                const Icon = categoryIcons[cat] || Package;
                return (
                  <button
                    key={cat}
                    onClick={() => setSelectedCategory(cat)}
                    className={cn(
                      'px-4 py-2 rounded-full text-sm font-medium transition-all flex items-center gap-2',
                      selectedCategory === cat
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted hover:bg-muted/80'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {cat}
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Content */}
      <AnimatePresence mode="wait">
        {activeTab === 'catalog' ? (
          <motion.div
            key="catalog"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            {catalogLoading ? (
              <div className="flex items-center justify-center py-20">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
              </div>
            ) : filteredEntries.length === 0 ? (
              <Card className="p-12 text-center">
                <Package className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No apps found</h3>
                <p className="text-muted-foreground">
                  {searchQuery || selectedCategory
                    ? 'Try adjusting your filters'
                    : 'No apps available in the catalog'}
                </p>
              </Card>
            ) : (
              <motion.div
                variants={containerVariants}
                initial="hidden"
                animate="visible"
                className={cn(
                  viewMode === 'grid'
                    ? "grid gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
                    : "space-y-4"
                )}
              >
                {filteredEntries.map((app) => (
                  <AppCard
                    key={app.id}
                    app={app}
                    isInstalled={installedAppIds.has(app.id)}
                  />
                ))}
              </motion.div>
            )}
          </motion.div>
        ) : (
          <motion.div
            key="installed"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            {installedLoading ? (
              <div className="flex items-center justify-center py-20">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
              </div>
            ) : installedApps.length === 0 ? (
              <Card className="p-12 text-center">
                <Package className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No apps installed</h3>
                <p className="text-muted-foreground mb-6">
                  Install apps from the catalog to get started
                </p>
                <Button onClick={() => setActiveTab('catalog')}>
                  Browse Catalog
                </Button>
              </Card>
            ) : (
              <motion.div
                variants={containerVariants}
                initial="hidden"
                animate="visible"
                className={cn(
                  viewMode === 'grid'
                    ? "grid gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
                    : "space-y-4"
                )}
              >
                {installedApps.map((app) => (
                  <InstalledAppCard key={app.id} app={app} />
                ))}
              </motion.div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}