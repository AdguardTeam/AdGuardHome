import { useLocation } from '@solidjs/router';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { Paths, RoutePath } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';
import { apiClient } from 'panel/api/Api';
import { AccordionSection } from './AccordionSection';

import s from './styles.module.pcss';

type Props = {
    accountSubMenu: boolean;
    setAccountSubMenu: (value: boolean) => void;
    burgerMenuId?: string;
    closeSubMenu?: () => void;
    rightSideDropdown?: boolean;
    headerMenu?: boolean;
};

export const Menu = (props: Props) => {
    const location = useLocation();

    const isActive = (path: string | string[], full = false) => {
        const currentPath = location.pathname;
        const paths = Array.isArray(path) ? path : [path];

        return paths.some((p) => {
            if (currentPath === p) {
                return true;
            }

            if (full) {
                return false;
            }

            const normalizedPath = p.endsWith('/') ? p : `${p}/`;
            return currentPath.startsWith(normalizedPath);
        });
    };

    return (
        <div
            class={cn(s.menuWrapper, {
                [s.headerMenu]: props.headerMenu,
            })}
        >
            <nav class={s.topMenuWrapper}>
                <div class={s.menuLinkWrapper}>
                    <Link
                        class={cn(s.menuLink, {
                            [s.activeLink]: isActive(Paths.Dashboard, true),
                        })}
                        to={RoutePath.Dashboard}
                    >
                        <Icon class={s.linkIcon} icon="dashboard" />
                        <span class={theme.common.textOverflow}>
                            {intl.getMessage('dashboard')}
                        </span>
                    </Link>
                </div>
                <AccordionSection
                    title={intl.getMessage('settings')}
                    icon="settings"
                    items={[
                        {
                            label: intl.getMessage('settings_general_short'),
                            path: Paths.SettingsPage,
                            routePath: RoutePath.SettingsPage,
                        },
                        { label: 'DNS', path: Paths.Dns, routePath: RoutePath.Dns },
                        {
                            label: intl.getMessage('protocols'),
                            path: Paths.Encryption,
                            routePath: RoutePath.Encryption,
                        },
                        {
                            label: intl.getMessage('clients'),
                            path: Paths.Clients,
                            routePath: RoutePath.Clients,
                        },
                        { label: 'DHCP', path: Paths.Dhcp, routePath: RoutePath.Dhcp },
                    ]}
                    isActive={isActive}
                />
                <AccordionSection
                    title={intl.getMessage('filters')}
                    icon="tune"
                    items={[
                        {
                            label: intl.getMessage('blocklists_title'),
                            path: Paths.DnsBlocklists,
                            routePath: RoutePath.DnsBlocklists,
                        },
                        {
                            label: intl.getMessage('allowlists'),
                            path: Paths.DnsAllowlists,
                            routePath: RoutePath.DnsAllowlists,
                        },
                        {
                            label: intl.getMessage('dns_rewrites'),
                            path: Paths.DnsRewrites,
                            routePath: RoutePath.DnsRewrites,
                        },
                        {
                            label: intl.getMessage('blocked_services'),
                            path: Paths.BlockedServices,
                            routePath: RoutePath.BlockedServices,
                        },
                        {
                            label: intl.getMessage('user_rules_title'),
                            path: Paths.UserRules,
                            routePath: RoutePath.UserRules,
                        },
                    ]}
                    isActive={isActive}
                />
                <div class={cn(s.menuLinkWrapper)}>
                    <Link
                        class={cn(s.menuLink, {
                            [s.activeLink]: isActive(Paths.Logs),
                        })}
                        to={RoutePath.Logs}
                    >
                        <Icon class={s.linkIcon} icon="log" />
                        <span class={theme.common.textOverflow}>Logs</span>
                    </Link>
                </div>
                <div class={cn(s.menuLinkWrapper)}>
                    <Link
                        class={cn(s.menuLink, { [s.activeLink]: isActive(Paths.Guide) })}
                        to={RoutePath.Guide}
                    >
                        <Icon class={s.linkIcon} icon="faq" />
                        <span class={theme.common.textOverflow}>
                            {intl.getMessage('setup_guide')}
                        </span>
                    </Link>
                </div>
            </nav>
            <div class={s.referenceWrapper}>
                <div class={cn(s.menuLinkWrapper)}>
                    <a
                        href={apiClient.getLogoutUrl()}
                        target="_blank"
                        rel="noopener noreferrer"
                        class={s.menuLink}
                        id="sign_out"
                    >
                        <Icon class={s.linkIcon} icon="logout" />
                        <span class={theme.common.textOverflow}>{intl.getMessage('logout')}</span>
                    </a>
                </div>
            </div>
        </div>
    );
};
