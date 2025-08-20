import type { components } from '@/api-types';
type Share = components['schemas']['Share'];
type Props = { items: Share[]; onDelete:(id:string)=>Promise<void> };
export default function ShareTable({ items, onDelete }:Props) {
  if (!items?.length) return <div className="text-sm text-muted-foreground">No shares yet.</div>;
  return (
    <table className="w-full text-sm">
      <thead><tr><th>Name</th><th>Type</th><th>Path</th><th>RO</th><th>Users</th><th/></tr></thead>
      <tbody>
        {items.map(s=>(
          <tr key={s.id} className="border-b border-border/50">
            <td>{s.name}</td>
            <td className="uppercase">{s.type}</td>
            <td className="font-mono">{s.path}</td>
            <td>{s.ro ? 'Yes':'No'}</td>
            <td>{s.users?.length ?? 0}</td>
            <td><button className="text-red-500 hover:underline" onClick={()=>onDelete(s.id || '')}>Delete</button></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}


