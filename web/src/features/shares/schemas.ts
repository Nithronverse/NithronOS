import { z } from 'zod';
export const ShareType = z.enum(['smb','nfs']);
export const ShareName = z.string().regex(/^[A-Za-z0-9_-]{1,32}$/,'1–32 chars: letters, numbers, _ or -');
export const PathUnder = (roots:string[]) => z.string().refine(p=>roots.some(r=>p===r || p.startsWith(r+'/')), { message:'Path must be under a mounted pool' });
export const ShareForm = (roots:string[]) => z.object({
  type: ShareType,
  name: ShareName,
  ro: z.boolean().default(false),
  path: PathUnder(roots),
  users: z.array(z.string()).default([]),
});
export type ShareFormInput = z.input<ReturnType<typeof ShareForm>>;


