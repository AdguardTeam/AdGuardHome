import { Client, RootState } from 'panel/initialState';

import { CheckResultData, ResultActionKind, RewriteEntry } from './types';

type SafeSearchConfig = Record<string, boolean> & { enabled: boolean };

type BlockedServicesList = {
    ids?: string[];
    schedule?: Client['blocked_services_schedule'];
};

export const CLIENT_SCOPED_ACTIONS: ResultActionKind[] = [
    'disable-parental',
    'disable-safebrowsing',
    'disable-safesearch',
];

export const getPrimaryRule = (result?: CheckResultData | null) => result?.rules?.[0];

const normalizeClientIdentifier = (value: string) => value.trim().toLowerCase();

export const findPersistentClient = (clients: Client[], identifier?: string) => {
    if (!identifier) {
        return undefined;
    }

    const normalizedIdentifier = normalizeClientIdentifier(identifier);
    const matches = clients.filter((client) => {
        if (normalizeClientIdentifier(client.name) === normalizedIdentifier) {
            return true;
        }

        return client.ids.some(
            (clientId) => normalizeClientIdentifier(clientId) === normalizedIdentifier,
        );
    });

    return matches.length === 1 ? matches[0] : undefined;
};

const getSafeSearchConfig = (
    config?: Partial<SafeSearchConfig>,
    fallbackEnabled = false,
): SafeSearchConfig => ({
    ...(config || {}),
    enabled: config?.enabled ?? fallbackEnabled,
});

export const getEffectiveClientProtectionSettings = ({
    client,
    globalFilteringEnabled,
    settingsList,
}: {
    client: Client;
    globalFilteringEnabled: boolean;
    settingsList?: RootState['settings']['settingsList'];
}) => {
    if (client.use_global_settings && !settingsList) {
        return null;
    }

    const effectiveSafeSearch = client.use_global_settings
        ? getSafeSearchConfig(settingsList?.safesearch as Partial<SafeSearchConfig> | undefined)
        : getSafeSearchConfig(
              client.safe_search as Partial<SafeSearchConfig> | undefined,
              client.safesearch_enabled,
          );

    return {
        filtering_enabled: client.use_global_settings
            ? globalFilteringEnabled
            : client.filtering_enabled,
        parental_enabled: client.use_global_settings
            ? Boolean(settingsList?.parental.enabled)
            : client.parental_enabled,
        safebrowsing_enabled: client.use_global_settings
            ? Boolean(settingsList?.safebrowsing.enabled)
            : client.safebrowsing_enabled,
        safe_search: effectiveSafeSearch,
        safesearch_enabled: effectiveSafeSearch.enabled,
    };
};

export const getEffectiveBlockedServices = (
    client: Client,
    globalBlockedServices: BlockedServicesList,
) => {
    const effectiveIds = client.use_global_blocked_services
        ? globalBlockedServices.ids
        : client.blocked_services;

    if (!Array.isArray(effectiveIds)) {
        return null;
    }

    return {
        blocked_services: [...effectiveIds],
        blocked_services_schedule: client.use_global_blocked_services
            ? (globalBlockedServices.schedule ?? client.blocked_services_schedule)
            : client.blocked_services_schedule,
    };
};

const matchesDomain = (hostname: string, pattern: string): boolean => {
    const h = hostname.toLowerCase();
    const p = pattern.toLowerCase();

    if (p === h) {
        return true;
    }

    return p.startsWith('*.') && h.endsWith(p.slice(1));
};

export const findMatchedRewrite = (
    rewrites: RootState['rewrites']['list'],
    checkResult?: CheckResultData | null,
): RewriteEntry | null => {
    if (!checkResult?.hostname) {
        return null;
    }

    const matches = rewrites.filter((entry) => matchesDomain(checkResult.hostname!, entry.domain));

    return matches.length === 1 ? matches[0] : null;
};

export const findMatchedBlockedService = (
    allServices: RootState['services']['allServices'],
    checkResult?: CheckResultData | null,
) => {
    const currentRule = getPrimaryRule(checkResult)?.text;
    const normalizedServiceName = checkResult?.service_name?.trim().toLowerCase();

    return allServices.find((service) => {
        if (normalizedServiceName) {
            const matchesName = service.name?.toLowerCase() === normalizedServiceName;
            const matchesId = service.id?.toLowerCase() === normalizedServiceName;

            if (matchesName || matchesId) {
                return true;
            }
        }

        return Boolean(currentRule) && service.rules?.includes(currentRule);
    });
};
