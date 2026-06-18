import { updateBlockedServices } from './blocked_services.ts';

const baseUrl = process.env.ADGUARD_URL || 'http://localhost:3000';

await updateBlockedServices(baseUrl, { ids: [], schedule: { time_zone: 'Local' } });
console.log('Unblocked all services.');
