M1 QA checklist

- Create/import on virtual disks (loopback or sparse files)
- Add/remove device with balance; verify progress and final layout
- Scrub runs, status visible in UI; monthly schedule configured
- SMART scan shows alerts when fixtures injected (temp high, reallocated)
- fstrim timer ran this week (journal shows nos-fstrim.service); Schedules page shows last TRIM
- Support bundle downloads and contains redacted configs, logs, SMART snapshots, and recent tx logs


