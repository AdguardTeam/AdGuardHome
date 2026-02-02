import qs from 'qs';

const BasicPath = '/';
const pathBuilder = (path: string) => `${BasicPath}${path}`;

export const RoutePath = {
    Dashboard: 'Dashboard',
    Logs: 'Logs',
    Guide: 'Guide',
    Encryption: 'Encryption',
    Dhcp: 'Dhcp',
    Dns: 'Dns',
    SettingsPage: 'SettingsPage',
    Clients: 'Clients',
    DnsBlocklists: 'DnsBlocklists',
    DnsAllowlists: 'DnsAllowlists',
    DnsRewrites: 'DnsRewrites',
    CustomRules: 'CustomRules',
    BlockedServices: 'BlockedServices',
    UserRules: 'UserRules',
} as const;

export type RoutePathKey = keyof typeof RoutePath;

export const Paths: Record<RoutePathKey, string> = {
    Dashboard: pathBuilder('dashboard'),
    Logs: pathBuilder('logs'),
    Guide: pathBuilder('guide'),
    Encryption: pathBuilder('encryption'),
    Dhcp: pathBuilder('dhcp'),
    Dns: pathBuilder('dns'),
    SettingsPage: pathBuilder('settings'),
    Clients: pathBuilder('clients'),
    DnsBlocklists: pathBuilder('blocklists'),
    DnsAllowlists: pathBuilder('allowlists'),
    DnsRewrites: pathBuilder('dns_rewrites'),
    CustomRules: pathBuilder('custom_rules'),
    BlockedServices: pathBuilder('blocked_services'),
    UserRules: pathBuilder('user_rules'),
};

export type LinkParams = Partial<Record<string, string | number>>;

export const linkPathBuilder = (
    route: RoutePathKey,
    params?: LinkParams,
    query?: Partial<Record<string, string | number | boolean>>,
) => {
    let path = Paths[route];
    if (params) {
        Object.keys(params).forEach((key: string) => {
            path = path.replace(`:${key}`, String(params[key]));
        });
    }

    if (query) {
        path += `?${qs.stringify(query)}`;
    }

    return path;
};
