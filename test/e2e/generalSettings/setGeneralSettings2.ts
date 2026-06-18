import {
  updateStatsConfig,
  type StatsConfig,
} from '../shared/adguard/stats.ts';

export async function setStatisticsSettings(options: {
  baseUrl: string;
  config: StatsConfig;
  headers?: Record<string, string>;
}) {
  await updateStatsConfig({
    baseUrl: options.baseUrl,
    headers: options.headers,
  }, options.config);
}
