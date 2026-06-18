import { adguardGet, adguardPut, type AdGuardRequestContext } from './api.ts';

export interface ProfileSettings {
  theme?: 'auto' | 'light' | 'dark';
  language?: string;
}

export interface VersionInfo {
  version: string;
}

export async function updateProfile(
  context: AdGuardRequestContext,
  settings: ProfileSettings,
): Promise<void> {
  await adguardPut({
    ...context,
    path: '/control/profile/update',
    body: settings,
  });
}

export async function getVersionInfo(context: AdGuardRequestContext): Promise<VersionInfo> {
  return adguardGet<VersionInfo>({
    ...context,
    path: '/control/version.json',
  });
}
