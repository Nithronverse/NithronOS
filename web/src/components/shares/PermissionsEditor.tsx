import { useState, useMemo } from 'react'
import {
	Shield,
	UserPlus,
	Users,
	User,
	X,
	AlertCircle,
	Info
} from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'

import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from '@/components/ui/dialog'
import {
	Command,
	CommandEmpty,
	CommandGroup,
	CommandInput,
	CommandItem,
	CommandList,
} from '@/components/ui/command'
import { useUsers, useGroups } from '@/hooks/use-shares'

interface PermissionsEditorProps {
	owners: string[]
	readers: string[]
	onOwnersChange: (owners: string[]) => void
	onReadersChange: (readers: string[]) => void
	guestEnabled?: boolean
}

type PrincipalType = 'user' | 'group'
type PermissionLevel = 'owner' | 'reader'

export function PermissionsEditor({
	owners,
	readers,
	onOwnersChange,
	onReadersChange,
	guestEnabled = false,
}: PermissionsEditorProps) {
	const { data: users = [] } = useUsers()
	const { data: groups = [] } = useGroups()
	const [pickerOpen, setPickerOpen] = useState(false)
	const [pickerTarget, setPickerTarget] = useState<PermissionLevel | null>(null)
	const [searchQuery, setSearchQuery] = useState('')

	// Parse principals into structured format
	const parsePrincipal = (principal: string) => {
		const [type, name] = principal.split(':')
		return { type: type as PrincipalType, name }
	}

	const ownersParsed = owners.map(parsePrincipal)
	const readersParsed = readers.map(parsePrincipal)

	// Get all available principals not already assigned
	const availablePrincipals = useMemo(() => {
		const assigned = new Set([...owners, ...readers])
		const principals: Array<{ type: PrincipalType; name: string; display: string }> = []

		users.forEach(user => {
			const key = `user:${user.username}`
			if (!assigned.has(key)) {
				principals.push({
					type: 'user',
					name: user.username,
					display: user.username,
				})
			}
		})

		groups.forEach(group => {
			const key = `group:${group.name}`
			if (!assigned.has(key)) {
				principals.push({
					type: 'group',
					name: group.name,
					display: group.name,
				})
			}
		})

		return principals
	}, [users, groups, owners, readers])

	// Filter principals based on search
	const filteredPrincipals = availablePrincipals.filter(p =>
		p.display.toLowerCase().includes(searchQuery.toLowerCase())
	)

	const handleAddPrincipal = (type: PrincipalType, name: string) => {
		const principal = `${type}:${name}`
		if (pickerTarget === 'owner') {
			onOwnersChange([...owners, principal])
		} else if (pickerTarget === 'reader') {
			onReadersChange([...readers, principal])
		}
	}

	const handleRemoveOwner = (principal: string) => {
		onOwnersChange(owners.filter(o => o !== principal))
	}

	const handleRemoveReader = (principal: string) => {
		onReadersChange(readers.filter(r => r !== principal))
	}



	return (
		<div className="space-y-6">
			{/* Permissions Summary */}
			<Card className="p-6">
				<div className="space-y-4">
					<div className="flex items-center gap-2">
						<Shield className="h-5 w-5 text-muted-foreground" />
						<h3 className="font-medium">Permissions Summary</h3>
					</div>
					
					<div className="grid gap-4 md:grid-cols-2">
						<div className="space-y-2">
							<p className="text-sm font-medium text-green-600">Owners (Read/Write)</p>
							<p className="text-sm text-muted-foreground">
								Full access to create, modify, and delete files
							</p>
						</div>
						<div className="space-y-2">
							<p className="text-sm font-medium text-blue-600">Readers (Read Only)</p>
							<p className="text-sm text-muted-foreground">
								Can view and download files but cannot modify
							</p>
						</div>
					</div>

					{guestEnabled && (
						<Alert>
							<AlertCircle className="h-4 w-4" />
							<AlertDescription>
								Guest access is enabled. Anyone on the network can access this share without authentication.
								Explicit permissions will still apply for authenticated users.
							</AlertDescription>
						</Alert>
					)}
				</div>
			</Card>

			{/* Owners Section */}
			<Card className="p-6">
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<div>
							<h3 className="font-medium">Owners</h3>
							<p className="text-sm text-muted-foreground">Users and groups with full read/write access</p>
						</div>
						<Button
							size="sm"
							variant="outline"
							onClick={() => {
								setPickerTarget('owner')
								setPickerOpen(true)
							}}
						>
							<UserPlus className="h-4 w-4 mr-2" />
							Add Owner
						</Button>
					</div>

					{ownersParsed.length > 0 ? (
						<div className="flex flex-wrap gap-2">
							{ownersParsed.map((principal, index) => (
								<Badge
									key={index}
									variant="secondary"
									className="pl-2 pr-1 py-1 bg-green-500/10 text-green-700 hover:bg-green-500/20"
								>
									{principal.type === 'user' ? (
										<User className="h-3 w-3 mr-1" />
									) : (
										<Users className="h-3 w-3 mr-1" />
									)}
									{principal.name}
									<button
										className="h-4 w-4 ml-1 hover:bg-green-500/20 rounded"
										onClick={() => handleRemoveOwner(owners[index])}
									>
										<X className="h-3 w-3" />
									</button>
								</Badge>
							))}
						</div>
					) : (
						<p className="text-sm text-muted-foreground">No owners assigned</p>
					)}
				</div>
			</Card>

			{/* Readers Section */}
			<Card className="p-6">
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<div>
							<h3 className="font-medium">Readers</h3>
							<p className="text-sm text-muted-foreground">Users and groups with read-only access</p>
						</div>
						<Button
							size="sm"
							variant="outline"
							onClick={() => {
								setPickerTarget('reader')
								setPickerOpen(true)
							}}
						>
							<UserPlus className="h-4 w-4 mr-2" />
							Add Reader
						</Button>
					</div>

					{readersParsed.length > 0 ? (
						<div className="flex flex-wrap gap-2">
							{readersParsed.map((principal, index) => (
								<Badge
									key={index}
									variant="secondary"
									className="pl-2 pr-1 py-1 bg-blue-500/10 text-blue-700 hover:bg-blue-500/20"
								>
									{principal.type === 'user' ? (
										<User className="h-3 w-3 mr-1" />
									) : (
										<Users className="h-3 w-3 mr-1" />
									)}
									{principal.name}
									<button
										className="h-4 w-4 ml-1 hover:bg-blue-500/20 rounded"
										onClick={() => handleRemoveReader(readers[index])}
									>
										<X className="h-3 w-3" />
									</button>
								</Badge>
							))}
						</div>
					) : (
						<p className="text-sm text-muted-foreground">No readers assigned</p>
					)}
				</div>
			</Card>

			{/* Warnings */}
			{owners.length === 0 && readers.length === 0 && !guestEnabled && (
				<Alert variant="destructive">
					<AlertCircle className="h-4 w-4" />
					<AlertDescription>
						No permissions configured. This share will not be accessible to anyone.
						Add at least one owner or reader, or enable guest access.
					</AlertDescription>
				</Alert>
			)}

			{guestEnabled && readers.length > 0 && (
				<Alert>
					<Info className="h-4 w-4" />
					<AlertDescription>
						Guest access is enabled along with explicit readers. 
						Guests will have read-only access, while configured readers will have their specified permissions.
					</AlertDescription>
				</Alert>
			)}

			{/* Principal Picker Dialog */}
			<Dialog open={pickerOpen} onOpenChange={setPickerOpen}>
				<DialogContent className="max-w-md">
					<DialogHeader>
						<DialogTitle>
							Add {pickerTarget === 'owner' ? 'Owner' : 'Reader'}
						</DialogTitle>
						<DialogDescription>
							Select users or groups to grant {pickerTarget === 'owner' ? 'read/write' : 'read-only'} access
						</DialogDescription>
					</DialogHeader>

					<Command>
						<CommandInput 
							placeholder="Search users and groups..." 
							value={searchQuery}
							onValueChange={setSearchQuery}
						/>
						<CommandList>
							<CommandEmpty>No users or groups found.</CommandEmpty>
							
							{filteredPrincipals.filter(p => p.type === 'user').length > 0 && (
								<CommandGroup heading="Users">
									{filteredPrincipals
										.filter(p => p.type === 'user')
										.map((principal) => (
											<CommandItem
												key={`user:${principal.name}`}
												onSelect={() => {
													handleAddPrincipal(principal.type, principal.name)
													setPickerOpen(false)
													setSearchQuery('')
												}}
											>
												<User className="h-4 w-4 mr-2" />
												{principal.display}
											</CommandItem>
										))}
								</CommandGroup>
							)}

							{filteredPrincipals.filter(p => p.type === 'group').length > 0 && (
								<CommandGroup heading="Groups">
									{filteredPrincipals
										.filter(p => p.type === 'group')
										.map((principal) => (
											<CommandItem
												key={`group:${principal.name}`}
												onSelect={() => {
													handleAddPrincipal(principal.type, principal.name)
													setPickerOpen(false)
													setSearchQuery('')
												}}
											>
												<Users className="h-4 w-4 mr-2" />
												{principal.display}
											</CommandItem>
										))}
								</CommandGroup>
							)}
						</CommandList>
					</Command>
				</DialogContent>
			</Dialog>
		</div>
	)
}
