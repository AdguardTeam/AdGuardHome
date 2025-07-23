import React from 'react';
import { useLocation } from 'react-router-dom';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { Paths, RoutePath } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';
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

export const Menu = ({ headerMenu }: Props) => {
    const location = useLocation();

    const isActive = (path: string | string[], full = false) => {
        const currentPath = location.pathname;

        if (Array.isArray(path)) {
            return path.some((p) => (full ? p === currentPath : currentPath === p || currentPath.startsWith(p + '/')));
        }

        if (full) {
            return path === currentPath;
        }

        return (
            currentPath === path ||
            (currentPath.startsWith(path) && (currentPath[path.length] === '/' || path.endsWith('/')))
        );
    };

    return (
        <div
            className={cn(s.menuWrapper, {
                [s.headerMenu]: headerMenu,
            })}>
            <nav className={s.topMenuWrapper}>
                <div className={s.menuLinkWrapper}>
                    <Link
                        className={cn(s.menuLink, {
                            [s.activeLink]: isActive(Paths.Dashboard, true),
                        })}
                        to={RoutePath.Dashboard}>
                        <Icon className={s.linkIcon} icon="dashboard" />
                        <span className={theme.common.textOverflow}>{intl.getMessage('dashboard')}</span>
                    </Link>
                </div>
                <AccordionSection
                    title={intl.getMessage('settings')}
                    icon="settings"
                    items={[
                        {
                            label: intl.getMessage('general_settings_short'),
                            path: Paths.SettingsPage,
                            routePath: RoutePath.SettingsPage,
                        },
                        { label: 'DNS', path: Paths.Dns, routePath: RoutePath.Dns },
                        {
                            label: intl.getMessage('encryption_title'),
                            path: Paths.Encryption,
                            routePath: RoutePath.Encryption,
                        },
                        { label: intl.getMessage('clients'), path: Paths.Clients, routePath: RoutePath.Clients },
                        { label: 'DHCP', path: Paths.Dhcp, routePath: RoutePath.Dhcp },
                    ]}
                    isActive={isActive}
                />
                <AccordionSection
                    title={intl.getMessage('filters')}
                    icon="tune"
                    items={[
                        {
                            label: intl.getMessage('blocklists'),
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
                        { label: intl.getMessage('user_rules'), path: Paths.UserRules, routePath: RoutePath.UserRules },
                    ]}
                    isActive={isActive}
                />
                <div className={cn(s.menuLinkWrapper)}>
                    <Link
                        className={cn(s.menuLink, {
                            [s.activeLink]: isActive(Paths.Logs),
                        })}
                        to={RoutePath.Logs}>
                        <Icon className={s.linkIcon} icon="log" />
                        <span className={theme.common.textOverflow}>Logs</span>
                    </Link>
                </div>
                <div className={cn(s.menuLinkWrapper)}>
                    <Link className={cn(s.menuLink, { [s.activeLink]: isActive(Paths.Guide) })} to={RoutePath.Guide}>
                        <Icon className={s.linkIcon} icon="faq" />
                        <span className={theme.common.textOverflow}>{intl.getMessage('setup_guide')}</span>
                    </Link>
                </div>
            </nav>
            <div className={s.referenceWrapper}>
                <div className={cn(s.menuLinkWrapper)}>
                    <a target="_blank" rel="noopener noreferrer" className={s.menuLink} href="">
                        <Icon className={s.linkIcon} icon="logout" />
                        <span className={theme.common.textOverflow}>{intl.getMessage('logout')}</span>
                    </a>
                </div>
            </div>
        </div>
    );
};
