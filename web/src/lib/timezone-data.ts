// Timezone data with regions/countries
export interface RegionData {
  name: string
  timezones: Array<{
    id: string
    name: string
    offset?: string
  }>
}

export const TIMEZONE_REGIONS: Record<string, RegionData> = {
  'North America': {
    name: 'North America',
    timezones: [
      { id: 'America/New_York', name: 'Eastern Time (New York)', offset: 'UTC-5' },
      { id: 'America/Chicago', name: 'Central Time (Chicago)', offset: 'UTC-6' },
      { id: 'America/Denver', name: 'Mountain Time (Denver)', offset: 'UTC-7' },
      { id: 'America/Phoenix', name: 'Mountain Time - Arizona', offset: 'UTC-7' },
      { id: 'America/Los_Angeles', name: 'Pacific Time (Los Angeles)', offset: 'UTC-8' },
      { id: 'America/Anchorage', name: 'Alaska Time', offset: 'UTC-9' },
      { id: 'Pacific/Honolulu', name: 'Hawaii Time', offset: 'UTC-10' },
      { id: 'America/Toronto', name: 'Eastern Time (Toronto)', offset: 'UTC-5' },
      { id: 'America/Vancouver', name: 'Pacific Time (Vancouver)', offset: 'UTC-8' },
      { id: 'America/Mexico_City', name: 'Central Time (Mexico)', offset: 'UTC-6' },
    ],
  },
  'South America': {
    name: 'South America',
    timezones: [
      { id: 'America/Sao_Paulo', name: 'Brazil (São Paulo)', offset: 'UTC-3' },
      { id: 'America/Buenos_Aires', name: 'Argentina (Buenos Aires)', offset: 'UTC-3' },
      { id: 'America/Santiago', name: 'Chile (Santiago)', offset: 'UTC-3' },
      { id: 'America/Bogota', name: 'Colombia (Bogotá)', offset: 'UTC-5' },
      { id: 'America/Lima', name: 'Peru (Lima)', offset: 'UTC-5' },
      { id: 'America/Caracas', name: 'Venezuela (Caracas)', offset: 'UTC-4' },
    ],
  },
  'Europe': {
    name: 'Europe',
    timezones: [
      { id: 'Europe/London', name: 'United Kingdom (London)', offset: 'UTC+0' },
      { id: 'Europe/Paris', name: 'France (Paris)', offset: 'UTC+1' },
      { id: 'Europe/Berlin', name: 'Germany (Berlin)', offset: 'UTC+1' },
      { id: 'Europe/Rome', name: 'Italy (Rome)', offset: 'UTC+1' },
      { id: 'Europe/Madrid', name: 'Spain (Madrid)', offset: 'UTC+1' },
      { id: 'Europe/Amsterdam', name: 'Netherlands (Amsterdam)', offset: 'UTC+1' },
      { id: 'Europe/Brussels', name: 'Belgium (Brussels)', offset: 'UTC+1' },
      { id: 'Europe/Vienna', name: 'Austria (Vienna)', offset: 'UTC+1' },
      { id: 'Europe/Warsaw', name: 'Poland (Warsaw)', offset: 'UTC+1' },
      { id: 'Europe/Moscow', name: 'Russia (Moscow)', offset: 'UTC+3' },
      { id: 'Europe/Istanbul', name: 'Turkey (Istanbul)', offset: 'UTC+3' },
      { id: 'Europe/Athens', name: 'Greece (Athens)', offset: 'UTC+2' },
      { id: 'Europe/Helsinki', name: 'Finland (Helsinki)', offset: 'UTC+2' },
      { id: 'Europe/Stockholm', name: 'Sweden (Stockholm)', offset: 'UTC+1' },
      { id: 'Europe/Oslo', name: 'Norway (Oslo)', offset: 'UTC+1' },
      { id: 'Europe/Copenhagen', name: 'Denmark (Copenhagen)', offset: 'UTC+1' },
      { id: 'Europe/Dublin', name: 'Ireland (Dublin)', offset: 'UTC+0' },
      { id: 'Europe/Lisbon', name: 'Portugal (Lisbon)', offset: 'UTC+0' },
      { id: 'Europe/Zurich', name: 'Switzerland (Zurich)', offset: 'UTC+1' },
    ],
  },
  'Asia': {
    name: 'Asia',
    timezones: [
      { id: 'Asia/Tokyo', name: 'Japan (Tokyo)', offset: 'UTC+9' },
      { id: 'Asia/Shanghai', name: 'China (Shanghai/Beijing)', offset: 'UTC+8' },
      { id: 'Asia/Hong_Kong', name: 'Hong Kong', offset: 'UTC+8' },
      { id: 'Asia/Singapore', name: 'Singapore', offset: 'UTC+8' },
      { id: 'Asia/Seoul', name: 'South Korea (Seoul)', offset: 'UTC+9' },
      { id: 'Asia/Taipei', name: 'Taiwan (Taipei)', offset: 'UTC+8' },
      { id: 'Asia/Bangkok', name: 'Thailand (Bangkok)', offset: 'UTC+7' },
      { id: 'Asia/Jakarta', name: 'Indonesia (Jakarta)', offset: 'UTC+7' },
      { id: 'Asia/Manila', name: 'Philippines (Manila)', offset: 'UTC+8' },
      { id: 'Asia/Kolkata', name: 'India (Kolkata/Delhi)', offset: 'UTC+5:30' },
      { id: 'Asia/Dubai', name: 'UAE (Dubai)', offset: 'UTC+4' },
      { id: 'Asia/Tel_Aviv', name: 'Israel (Tel Aviv)', offset: 'UTC+2' },
      { id: 'Asia/Riyadh', name: 'Saudi Arabia (Riyadh)', offset: 'UTC+3' },
      { id: 'Asia/Karachi', name: 'Pakistan (Karachi)', offset: 'UTC+5' },
      { id: 'Asia/Dhaka', name: 'Bangladesh (Dhaka)', offset: 'UTC+6' },
      { id: 'Asia/Ho_Chi_Minh', name: 'Vietnam (Ho Chi Minh)', offset: 'UTC+7' },
      { id: 'Asia/Kuala_Lumpur', name: 'Malaysia (Kuala Lumpur)', offset: 'UTC+8' },
    ],
  },
  'Africa': {
    name: 'Africa',
    timezones: [
      { id: 'Africa/Cairo', name: 'Egypt (Cairo)', offset: 'UTC+2' },
      { id: 'Africa/Johannesburg', name: 'South Africa (Johannesburg)', offset: 'UTC+2' },
      { id: 'Africa/Lagos', name: 'Nigeria (Lagos)', offset: 'UTC+1' },
      { id: 'Africa/Nairobi', name: 'Kenya (Nairobi)', offset: 'UTC+3' },
      { id: 'Africa/Casablanca', name: 'Morocco (Casablanca)', offset: 'UTC+1' },
      { id: 'Africa/Algiers', name: 'Algeria (Algiers)', offset: 'UTC+1' },
      { id: 'Africa/Tunis', name: 'Tunisia (Tunis)', offset: 'UTC+1' },
    ],
  },
  'Oceania': {
    name: 'Oceania',
    timezones: [
      { id: 'Australia/Sydney', name: 'Australia - Sydney/Melbourne', offset: 'UTC+10' },
      { id: 'Australia/Brisbane', name: 'Australia - Brisbane', offset: 'UTC+10' },
      { id: 'Australia/Perth', name: 'Australia - Perth', offset: 'UTC+8' },
      { id: 'Australia/Adelaide', name: 'Australia - Adelaide', offset: 'UTC+9:30' },
      { id: 'Australia/Darwin', name: 'Australia - Darwin', offset: 'UTC+9:30' },
      { id: 'Australia/Hobart', name: 'Australia - Tasmania', offset: 'UTC+10' },
      { id: 'Pacific/Auckland', name: 'New Zealand (Auckland)', offset: 'UTC+12' },
      { id: 'Pacific/Fiji', name: 'Fiji', offset: 'UTC+12' },
    ],
  },
  'UTC': {
    name: 'UTC/GMT',
    timezones: [
      { id: 'UTC', name: 'Coordinated Universal Time', offset: 'UTC+0' },
      { id: 'GMT', name: 'Greenwich Mean Time', offset: 'UTC+0' },
    ],
  },
}

// Helper to get all timezones as a flat list
export function getAllTimezones(): string[] {
  const timezones: string[] = []
  Object.values(TIMEZONE_REGIONS).forEach(region => {
    region.timezones.forEach(tz => {
      if (!timezones.includes(tz.id)) {
        timezones.push(tz.id)
      }
    })
  })
  return timezones.sort()
}

// Helper to find a timezone's region and display name
export function getTimezoneInfo(timezoneId: string): { region: string; name: string } | null {
  for (const [regionKey, regionData] of Object.entries(TIMEZONE_REGIONS)) {
    const tz = regionData.timezones.find(t => t.id === timezoneId)
    if (tz) {
      return { region: regionKey, name: tz.name }
    }
  }
  return null
}
