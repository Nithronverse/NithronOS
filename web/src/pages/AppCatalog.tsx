import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
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
  Loader2
} from 'lucide-react';
import { appsApi } from '../api/apps';
import type { CatalogEntry, InstalledApp } from '../api/apps.types';
import { cn } from '../lib/utils';

export function AppCatalog() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'catalog' | 'installed'>('catalog');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [officialOnly, setOfficialOnly] = useState(false);

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

      // Official filter (for now, all are considered official)
      if (officialOnly && !app.id.startsWith('official-')) {
        // In production, we'd check a flag or source
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
      // Refetch installed apps
      await installedData?.refetch();
    } catch (error) {
      console.error(`Failed to ${action} app:`, error);
    }
  };

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Package className="w-8 h-8 text-blue-500" />
          <h1 className="text-3xl font-bold">App Catalog</h1>
        </div>
        <button
          onClick={() => appsApi.syncCatalogs()}
          className="btn btn-secondary"
        >
          <RotateCw className="w-4 h-4 mr-2" />
          Sync Catalogs
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 bg-gray-800 rounded-lg w-fit">
        <button
          onClick={() => setActiveTab('catalog')}
          className={cn(
            'px-4 py-2 rounded-md transition-colors',
            activeTab === 'catalog'
              ? 'bg-gray-700 text-white'
              : 'text-gray-400 hover:text-white'
          )}
        >
          Catalog
        </button>
        <button
          onClick={() => setActiveTab('installed')}
          className={cn(
            'px-4 py-2 rounded-md transition-colors relative',
            activeTab === 'installed'
              ? 'bg-gray-700 text-white'
              : 'text-gray-400 hover:text-white'
          )}
        >
          Installed
          {installedApps.length > 0 && (
            <span className="ml-2 px-2 py-0.5 text-xs bg-blue-600 rounded-full">
              {installedApps.length}
            </span>
          )}
        </button>
      </div>

      {/* Filters */}
      {activeTab === 'catalog' && (
        <div className="space-y-4">
          {/* Search and toggles */}
          <div className="flex gap-4 items-center">
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search apps..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="input pl-10 w-full"
              />
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={officialOnly}
                onChange={(e) => setOfficialOnly(e.target.checked)}
                className="checkbox"
              />
              <span className="text-sm">Only Official</span>
            </label>
          </div>

          {/* Category chips */}
          <div className="flex gap-2 flex-wrap">
            <button
              onClick={() => setSelectedCategory(null)}
              className={cn(
                'px-3 py-1 rounded-full text-sm transition-colors',
                selectedCategory === null
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
              )}
            >
              All
            </button>
            {categories.map(cat => (
              <button
                key={cat}
                onClick={() => setSelectedCategory(cat)}
                className={cn(
                  'px-3 py-1 rounded-full text-sm transition-colors',
                  selectedCategory === cat
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
                )}
              >
                {cat}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Content */}
      {activeTab === 'catalog' ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {catalogLoading ? (
            <div className="col-span-full flex justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : filteredEntries.length === 0 ? (
            <div className="col-span-full text-center py-12">
              <Package className="w-12 h-12 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400">No apps found</p>
            </div>
          ) : (
            filteredEntries.map((app: CatalogEntry) => (
              <div
                key={app.id}
                className="bg-gray-800 rounded-lg p-4 hover:bg-gray-750 transition-colors"
              >
                {/* App Icon */}
                <div className="w-12 h-12 bg-gray-700 rounded-lg flex items-center justify-center mb-3">
                  <Package className="w-6 h-6 text-gray-400" />
                </div>

                {/* App Info */}
                <h3 className="font-semibold text-lg mb-1">{app.name}</h3>
                <p className="text-sm text-gray-400 mb-3 line-clamp-2">
                  {app.description}
                </p>

                {/* Categories */}
                <div className="flex gap-1 flex-wrap mb-3">
                  {app.categories?.slice(0, 2).map((cat: string) => (
                    <span
                      key={cat}
                      className="text-xs px-2 py-1 bg-gray-700 rounded-full"
                    >
                      {cat}
                    </span>
                  ))}
                </div>

                {/* Actions */}
                {installedAppIds.has(app.id) ? (
                  <button
                    onClick={() => navigate(`/apps/${app.id}`)}
                    className="btn btn-secondary btn-sm w-full"
                  >
                    <CheckCircle className="w-4 h-4 mr-2" />
                    Installed
                  </button>
                ) : (
                  <button
                    onClick={() => navigate(`/apps/install/${app.id}`)}
                    className="btn btn-primary btn-sm w-full"
                  >
                    <Download className="w-4 h-4 mr-2" />
                    Install
                  </button>
                )}
              </div>
            ))
          )}
        </div>
      ) : (
        <div className="space-y-4">
          {installedLoading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : installedApps.length === 0 ? (
            <div className="text-center py-12">
              <Package className="w-12 h-12 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400 mb-4">No apps installed yet</p>
              <button
                onClick={() => setActiveTab('catalog')}
                className="btn btn-primary"
              >
                Browse Catalog
              </button>
            </div>
          ) : (
            installedApps.map((app: InstalledApp) => (
              <div
                key={app.id}
                className="bg-gray-800 rounded-lg p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-4">
                  {/* App Icon */}
                  <div className="w-12 h-12 bg-gray-700 rounded-lg flex items-center justify-center">
                    <Package className="w-6 h-6 text-gray-400" />
                  </div>

                  {/* App Info */}
                  <div>
                    <h3 className="font-semibold text-lg">{app.name}</h3>
                    <div className="flex items-center gap-4 text-sm">
                      <span className="text-gray-400">v{app.version}</span>
                      <div className={cn('flex items-center gap-1', getStatusColor(app.status))}>
                        {getStatusIcon(app.status)}
                        <span className="capitalize">{app.status}</span>
                      </div>
                      {app.health && (
                        <div className={cn(
                          'flex items-center gap-1',
                          app.health.status === 'healthy' ? 'text-green-500' :
                          app.health.status === 'unhealthy' ? 'text-red-500' :
                          'text-gray-400'
                        )}>
                          <Shield className="w-4 h-4" />
                          <span className="capitalize">{app.health.status}</span>
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2">
                  {app.urls?.length > 0 && (
                    <a
                      href={app.urls[0]}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="btn btn-secondary btn-sm"
                    >
                      <ExternalLink className="w-4 h-4" />
                    </a>
                  )}
                  {app.status === 'stopped' ? (
                    <button
                      onClick={() => handleAppAction(app.id, 'start')}
                      className="btn btn-secondary btn-sm"
                    >
                      <Play className="w-4 h-4" />
                    </button>
                  ) : app.status === 'running' ? (
                    <>
                      <button
                        onClick={() => handleAppAction(app.id, 'restart')}
                        className="btn btn-secondary btn-sm"
                      >
                        <RotateCw className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleAppAction(app.id, 'stop')}
                        className="btn btn-secondary btn-sm"
                      >
                        <Square className="w-4 h-4" />
                      </button>
                    </>
                  ) : null}
                  <button
                    onClick={() => navigate(`/apps/${app.id}`)}
                    className="btn btn-primary btn-sm"
                  >
                    Details
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
