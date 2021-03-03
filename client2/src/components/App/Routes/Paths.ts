import qs from 'qs';
import { Locale } from 'Localization';

const BasicPath = '/';
const pathBuilder = (path: string) => (`${BasicPath}${path}`);

export enum RoutePath {
    Dashboard = 'Dashboard',
    FiltersBlocklist = 'FiltersBlocklist',
    FiltersAllowlist = 'FiltersAllowlist',
    FiltersRewrites = 'FiltersRewrites',
    FiltersServices = 'FiltersServices',
    FiltersCustom = 'FiltersCustom',
    QueryLog = 'QueryLog',
    SetupGuide = 'SetupGuide',
    SettingsGeneral = 'SettingsGeneral',
    SettingsDns = 'SettingsDns',
    SettingsEncryption = 'SettingsEncryption',
    SettingsClients = 'SettingsClients',
    SettingsDhcp = 'SettingsDhcp',
    Login = 'Login',
    ForgotPassword = 'ForgotPassword',
}

export const Paths: Record<RoutePath, string> = {
    Dashboard: pathBuilder('dashboard'),
    FiltersBlocklist: pathBuilder('filters/blocklists'),
    FiltersAllowlist: pathBuilder('filters/allowlists'),
    FiltersRewrites: pathBuilder('filters/rewrites'),
    FiltersServices: pathBuilder('filters/services'),
    FiltersCustom: pathBuilder('filters/custom'),
    QueryLog: pathBuilder('logs'),
    SetupGuide: pathBuilder('guide'),
    SettingsGeneral: pathBuilder('settings/general'),
    SettingsDns: pathBuilder('settings/dns'),
    SettingsEncryption: pathBuilder('settings/encryption'),
    SettingsClients: pathBuilder('settings/clients'),
    SettingsDhcp: pathBuilder('settings/dhcp'),
    Login: pathBuilder(''),
    ForgotPassword: pathBuilder('forgot_password'),
};

export enum LinkParamsKeys {}
export enum QueryParams {}
export type LinkParams = Partial<Record<LinkParamsKeys, string | number>>;

export const linkPathBuilder = (
    route: RoutePath,
    params?: LinkParams,
    lang?: Locale,
    query?: Partial<Record<QueryParams, string | number | boolean>>,
) => {
    let path = Paths[route]; //  .replace(BasicPath, `/${lang}`);
    if (params) {
        Object.keys(params).forEach((key: unknown) => {
            path = path.replace(`:${key}`, String(params[key as LinkParamsKeys]));
        });
    }
    if (query) {
        path += `?${qs.stringify(query)}`;
    }
    return path;
};
