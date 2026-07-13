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
    DnsPrivateReverse: 'DnsPrivateReverse',
    SettingsPage: 'SettingsPage',
    Clients: 'Clients',
    DnsBlocklists: 'DnsBlocklists',
    DnsAllowlists: 'DnsAllowlists',
    DnsRewrites: 'DnsRewrites',
    BlockedServices: 'BlockedServices',
    InactivitySchedule: 'InactivitySchedule',
    UserRules: 'UserRules',
    QueryLog: 'QueryLog',
    ClientsAdd: 'ClientsAdd',
    ClientsProtection: 'ClientsProtection',
    ClientsBlockedServices: 'ClientsBlockedServices',
    ClientsSchedule: 'ClientsSchedule',
    ClientsEdit: 'ClientsEdit',
    ClientsEditProtection: 'ClientsEditProtection',
    ClientsEditBlockedServices: 'ClientsEditBlockedServices',
    ClientsEditSchedule: 'ClientsEditSchedule',
} as const;

export type RoutePathKey = keyof typeof RoutePath;

/** Query param key used to pass a target element ID for scroll-to-section navigation. */
export const SCROLL_QUERY_KEY = 'section';

export const Paths: Record<RoutePathKey, string> = {
    Dashboard: pathBuilder('dashboard'),
    Logs: pathBuilder('logs'),
    Guide: pathBuilder('guide'),
    Encryption: pathBuilder('encryption'),
    Dhcp: pathBuilder('dhcp'),
    Dns: pathBuilder('dns'),
    DnsPrivateReverse: pathBuilder('dns/private-reverse'),
    SettingsPage: pathBuilder('settings'),
    Clients: pathBuilder('clients'),
    DnsBlocklists: pathBuilder('blocklists'),
    DnsAllowlists: pathBuilder('allowlists'),
    DnsRewrites: pathBuilder('dns_rewrites'),
    BlockedServices: pathBuilder('blocked_services'),
    InactivitySchedule: pathBuilder('blocked_services/schedule'),
    UserRules: pathBuilder('user_rules'),
    QueryLog: pathBuilder('logs'),
    ClientsAdd: pathBuilder('clients/add'),
    ClientsProtection: pathBuilder('clients/add/protection'),
    ClientsBlockedServices: pathBuilder('clients/add/blocked_services'),
    ClientsSchedule: pathBuilder('clients/add/blocked_services/schedule'),
    ClientsEdit: pathBuilder('clients/edit/:clientName'),
    ClientsEditProtection: pathBuilder('clients/edit/:clientName/protection'),
    ClientsEditBlockedServices: pathBuilder('clients/edit/:clientName/blocked_services'),
    ClientsEditSchedule: pathBuilder('clients/edit/:clientName/blocked_services/schedule'),
};

export type LinkParams = Partial<Record<string, string | number>>;

export const linkPathBuilder = (
    route: RoutePathKey,
    params?: LinkParams,
    query?: Partial<Record<string, string | number | boolean>>,
    hash?: string,
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

    if (hash) {
        path += `#${hash}`;
    }

    return path;
};
