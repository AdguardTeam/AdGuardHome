import { adguardGet, adguardPost, type AdGuardRequestContext } from './api.ts';

export interface FilteringStatus {
  enabled: boolean;
  filters_count?: number;
  protection_enabled?: boolean;
}

export async function getFilteringStatus(context: AdGuardRequestContext): Promise<FilteringStatus> {
  return adguardGet<FilteringStatus>({
    ...context,
    path: '/control/filtering/status',
  });
}

export async function setProtection(
  context: AdGuardRequestContext,
  enabled: boolean,
  durationMs?: number,
): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/protection',
    body: {
      enabled,
      ...(durationMs === undefined ? {} : { duration: durationMs }),
    },
  });
}

export async function setCustomRules(
  context: AdGuardRequestContext,
  rules: string[],
): Promise<void> {
  await adguardPost({
    ...context,
    path: '/control/filtering/set_rules',
    body: { rules },
  });
}
