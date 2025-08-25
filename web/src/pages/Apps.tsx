import { useState } from 'react'
import { motion } from 'framer-motion'
import { 
  Package,
  Download,
  Play,
  Square,
  RefreshCw,
  Settings,
  Trash2,
  Star,
  Shield,
  Search,
  Grid3x3,
  List
} from 'lucide-react'
import { ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { useApps, useMarketplace } from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import { pushToast } from '@/components/ui/toast'
import type { App } from '@/lib/api-client'

// Mock marketplace apps
const mockMarketplaceApps = [
  { 
    id: 'm1', 
    name: 'Plex Media Server', 
    version: '1.32.5', 
    category: 'Media',
    description: 'Stream your media to all your devices',
    status: 'available' as const,
    autoUpdate: false,
    official: true,
    rating: 4.8
  },
  { 
    id: 'm2', 
    name: 'Nextcloud', 
    version: '27.1.0', 
    category: 'Productivity',
    description: 'Self-hosted cloud storage and collaboration',
    status: 'available' as const,
    autoUpdate: false,
    official: true,
    rating: 4.6
  },
  { 
    id: 'm3', 
    name: 'Home Assistant', 
    version: '2023.10', 
    category: 'Home Automation',
    description: 'Open source home automation platform',
    status: 'available' as const,
    autoUpdate: false,
    official: false,
    rating: 4.9
  },
]

// Categories for filtering
const categories = ['All', 'Media', 'Productivity', 'Security', 'Network', 'Home Automation']

// Installed app columns
const installedColumns: ColumnDef<App>[] = [
  {
    accessorKey: 'name',
    header: 'Application',
    cell: ({ row }) => (
      <div className="flex items-center gap-3">
        <div className="p-2 rounded-lg bg-muted">
          <Package className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">{row.original.version}</div>
        </div>
      </div>
    ),
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.original.status
      return (
        <StatusPill variant={
          status === 'running' ? 'success' :
          status === 'stopped' ? 'muted' : 'error'
        }>
          {status}
        </StatusPill>
      )
    },
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: ({ row }) => {
      const isRunning = row.original.status === 'running'
      return (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
          >
            {isRunning ? <Square className="h-4 w-4" /> : <Play className="h-4 w-4" />}
          </Button>
          <Button variant="ghost" size="sm">
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm">
            <Settings className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" className="text-destructive">
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      )
    },
  },
]

export function Apps() {
  const [activeTab, setActiveTab] = useState('installed')
  const [selectedCategory, setSelectedCategory] = useState('All')
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  
  const { data: apps, isLoading: appsLoading } = useApps()
  const { data: marketplace, isLoading: marketplaceLoading } = useMarketplace()

  // Use mock data if API fails
  const installedApps = apps || [
    { id: '1', name: 'Plex', version: '1.32.5', status: 'running' as const, autoUpdate: true },
    { id: '2', name: 'Nextcloud', version: '27.0.2', status: 'running' as const, autoUpdate: false },
    { id: '3', name: 'Home Assistant', version: '2023.9.3', status: 'stopped' as const, autoUpdate: true },
  ]

  const marketplaceApps = marketplace || mockMarketplaceApps

  // Filter marketplace apps
  const filteredMarketplace = marketplaceApps.filter((app: any) => {
    const matchesCategory = selectedCategory === 'All' || app.category === selectedCategory
    const matchesSearch = app.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          app.description?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesCategory && matchesSearch
  })

  const handleInstall = async (app: any) => {
    pushToast(`Installing ${app.name}...`, 'success')
    setTimeout(() => {
      pushToast(`${app.name} installed successfully`, 'success')
    }, 2000)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Applications"
        description="Manage installed apps and browse the marketplace"
        actions={
          <Button
            size="sm"
            onClick={() => setActiveTab('marketplace')}
          >
            <Download className="h-4 w-4 mr-2" />
            Marketplace
          </Button>
        }
      />

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="installed">
            <Package className="h-4 w-4 mr-2" />
            Installed ({installedApps.length})
          </TabsTrigger>
          <TabsTrigger value="marketplace">
            <Download className="h-4 w-4 mr-2" />
            Marketplace
          </TabsTrigger>
        </TabsList>

        <TabsContent value="installed">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Card
              title="Installed Applications"
              description="Apps currently installed on your system"
              isLoading={appsLoading}
            >
              {installedApps.length > 0 ? (
                <DataTable
                  columns={installedColumns}
                  data={installedApps}
                  searchKey="applications"
                />
              ) : (
                <EmptyState
                  variant="no-data"
                  icon={Package}
                  title="No apps installed"
                  description="Browse the marketplace to find and install applications"
                  action={{
                    label: "Browse Marketplace",
                    onClick: () => setActiveTab('marketplace')
                  }}
                />
              )}
            </Card>
          </motion.div>
        </TabsContent>

        <TabsContent value="marketplace">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="space-y-4"
          >
            {/* Filters and search */}
            <Card>
              <div className="flex flex-col sm:flex-row gap-4">
                <div className="flex-1">
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <input
                      type="text"
                      placeholder="Search applications..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="w-full pl-10 pr-3 py-2 rounded-md border border-input bg-background text-sm"
                    />
                  </div>
                </div>
                <div className="flex gap-2">
                  <select
                    value={selectedCategory}
                    onChange={(e) => setSelectedCategory(e.target.value)}
                    className="px-3 py-2 rounded-md border border-input bg-background text-sm"
                  >
                    {categories.map(cat => (
                      <option key={cat} value={cat}>{cat}</option>
                    ))}
                  </select>
                  <div className="flex rounded-md border border-input">
                    <button
                      onClick={() => setViewMode('grid')}
                      className={cn(
                        "p-2 transition-colors",
                        viewMode === 'grid' ? "bg-primary text-primary-foreground" : "hover:bg-muted"
                      )}
                    >
                      <Grid3x3 className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => setViewMode('list')}
                      className={cn(
                        "p-2 transition-colors",
                        viewMode === 'list' ? "bg-primary text-primary-foreground" : "hover:bg-muted"
                      )}
                    >
                      <List className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            </Card>

            {/* Apps grid/list */}
            <Card
              title="Available Applications"
              description={`${filteredMarketplace.length} apps available`}
              isLoading={marketplaceLoading}
            >
              {filteredMarketplace.length > 0 ? (
                viewMode === 'grid' ? (
                  <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {filteredMarketplace.map((app: any) => (
                      <motion.div
                        key={app.id}
                        whileHover={{ scale: 1.02 }}
                        className="p-4 rounded-lg border border-border hover:border-primary/50 transition-colors cursor-pointer"
                      >
                        <div className="flex items-start gap-3">
                          <div className="p-2 rounded-lg bg-muted">
                            <Package className="h-6 w-6 text-muted-foreground" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-start justify-between">
                              <h4 className="font-medium truncate">{app.name}</h4>
                              {app.official && (
                                <Shield className="h-4 w-4 text-blue-500 shrink-0" />
                              )}
                            </div>
                            <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                              {app.description}
                            </p>
                            <div className="flex items-center gap-3 mt-3">
                              <div className="flex items-center gap-1">
                                <Star className="h-3 w-3 text-yellow-500 fill-yellow-500" />
                                <span className="text-xs">{app.rating}</span>
                              </div>
                              <span className="text-xs text-muted-foreground">
                                {app.category}
                              </span>
                            </div>
                          </div>
                        </div>
                        <Button
                          size="sm"
                          className="w-full mt-3"
                          onClick={() => handleInstall(app)}
                        >
                          <Download className="h-4 w-4 mr-2" />
                          Install
                        </Button>
                      </motion.div>
                    ))}
                  </div>
                ) : (
                  <EmptyState
                    variant="filtered"
                    title="List view coming soon"
                    description="Grid view is currently available"
                  />
                )
              ) : (
                <EmptyState
                  variant="filtered"
                  title="No apps found"
                  description="Try adjusting your search or filters"
                />
              )}
            </Card>
          </motion.div>
        </TabsContent>
      </Tabs>
    </div>
  )
}