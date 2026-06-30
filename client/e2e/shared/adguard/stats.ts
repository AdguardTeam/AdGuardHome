import { adguardGet, adguardPost, adguardPut, type AdGuardRequestContext } from './api.ts';

export interface StatsConfig {
  enabled: boolean;
  interval: number;
  ignored: string[];
  ignored_enabled: boolean;
}

export interface RankedItem {
  [key: string]: number;
}

export interface StatsResponse {
  top_queried_domains: RankedItem[];
  top_blocked_domains: RankedItem[];
  top_clients: RankedItem[];
  num_dns_queries: number;
  num_blocked_filtering: number;
  avg_processing_time?: number;
}

export async function getStats(context: AdGuardRequestContext): Promise<StatsResponse> {
  return adguardGet<StatsResponse>({
    ...context,
    path: '/control/stats',
  });
}

export async function getStatsConfig(context: AdGuardRequestContext): Promise<StatsConfig> {
  return adguardGet<StatsConfig>({
    ...context,
    path: '/control/stats/config',
  });
}

export async function updateStatsConfig(
  context: AdGuardRequestContext,
  config: StatsConfig,
): Promise<void> {
  await adguardPut({
    ...context,
    path: '/control/stats/config/update',
    body: config,
  });
}

export async function clearStats(context: AdGuardRequestContext): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/stats_reset',
  });
}
