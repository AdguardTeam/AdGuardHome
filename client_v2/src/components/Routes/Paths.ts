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
    CustomRules: 'CustomRules',
    DhcpLeases: 'DhcpLeases',
    QueryLog: 'QueryLog',
    ClientsAdd: 'ClientsAdd',
    ClientsProtection: 'ClientsProtection',
    ClientsBlockedServices: 'ClientsBlockedServices',
    ClientsFilterLists: 'ClientsFilterLists',
    ClientsSchedule: 'ClientsSchedule',
    ClientsEdit: 'ClientsEdit',
    ClientsEditProtection: 'ClientsEditProtection',
    ClientsEditBlockedServices: 'ClientsEditBlockedServices',
    ClientsEditFilterLists: 'ClientsEditFilterLists',
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
    CustomRules: pathBuilder('custom_rules'),
    DhcpLeases: pathBuilder('dhcp/leases'),
    QueryLog: pathBuilder('logs'),
    ClientsAdd: pathBuilder('clients/add'),
    ClientsProtection: pathBuilder('clients/add/protection'),
    ClientsBlockedServices: pathBuilder('clients/add/blocked_services'),
    ClientsFilterLists: pathBuilder('clients/add/filter_lists'),
    ClientsSchedule: pathBuilder('clients/add/blocked_services/schedule'),
    ClientsEdit: pathBuilder('clients/edit/:clientName'),
    ClientsEditProtection: pathBuilder('clients/edit/:clientName/protection'),
    ClientsEditBlockedServices: pathBuilder('clients/edit/:clientName/blocked_services'),
    ClientsEditFilterLists: pathBuilder('clients/edit/:clientName/filter_lists'),
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
