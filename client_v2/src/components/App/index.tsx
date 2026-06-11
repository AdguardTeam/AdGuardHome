import React, { useEffect, ComponentType } from 'react';

import { HashRouter, Route, Routes, Navigate } from 'react-router-dom';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import { Sidebar } from 'panel/common/ui/Sidebar';
import { Icons } from 'panel/common/ui/Icons';
import { Footer } from 'panel/common/ui/Footer';
import { Header } from 'panel/common/ui/Header';
import { Settings } from 'panel/components/Settings';
import intl, { LocalesType } from 'panel/common/intl';
import { Encryption } from 'panel/components/Encryption';
import { Blocklists } from 'panel/components/FilterLists/Blocklists';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';

import { Allowlists } from 'panel/components/FilterLists/Allowlists';
import { DNSRewrites } from 'panel/components/FilterLists/DNSRewrites';
import { SetupGuide } from 'panel/components/SetupGuide';
import { Dashboard } from 'panel/components/Dashboard';
import { Dhcp } from 'panel/components/Dhcp';
import { QueryLog } from 'panel/components/QueryLog';
import Toasts from '../Toasts';
import { THEMES } from '../../helpers/constants';
import { setHtmlLangAttr, setUITheme } from '../../helpers/helpers';
import { getDnsStatus, getTimerStatus } from '../../actions';
import { RootState } from '../../initialState';

import s from './styles.module.pcss';
import { DnsSettings } from '../DnsSettings';
import { UserRules } from '../UserRules';
import { BlockedServices } from '../BlockedServices';
import { Clients } from '../Clients/Clients';
import { InactivitySchedule } from '../BlockedServices/InactivitySchedule';
import { AddClient } from '../Clients/AddClient';
import { Protection } from '../Clients/AddClient/blocks/Protection/Protection';
import { ClientBlockedServices } from '../Clients/AddClient/blocks/ClientBlockedServices';
import { ClientSchedule } from '../Clients/AddClient/blocks/ClientSchedule';

type RouteConfig = {
    path: string;
    component: ComponentType;
};

const SetupGuideRoute = () => <SetupGuide />;
const BlockedServicesRoute = () => <BlockedServices />;
const InactivityScheduleRoute = () => <InactivitySchedule />;
const ClientScheduleRoute = () => <ClientSchedule />;
const ClientBlockedServicesRoute = () => <ClientBlockedServices />;
const ProtectionRoute = () => <Protection />;
const AddClientRoute = () => <AddClient />;

const ROUTES: RouteConfig[] = [
    {
        path: '/dashboard',
        component: Dashboard,
    },
    {
        path: '/settings',
        component: Settings,
    },
    {
        path: '/encryption',
        component: Encryption,
    },
    {
        path: '/dns',
        component: DnsSettings,
    },
    {
        path: '/blocklists',
        component: Blocklists,
    },
    {
        path: '/allowlists',
        component: Allowlists,
    },
    {
        path: '/user_rules',
        component: UserRules,
    },
    {
        path: '/dns_rewrites',
        component: DNSRewrites,
    },
    {
        path: '/dhcp',
        component: Dhcp,
    },
    {
        path: '/guide',
        component: SetupGuideRoute,
    },
    {
        path: '/logs',
        component: QueryLog,
    },
    {
        path: '/blocked_services/schedule',
        component: InactivityScheduleRoute,
    },
    {
        path: '/blocked_services',
        component: BlockedServicesRoute,
    },
    {
        path: '/clients/add/blocked_services/schedule',
        component: ClientScheduleRoute,
    },
    {
        path: '/clients/add/blocked_services',
        component: ClientBlockedServicesRoute,
    },
    {
        path: '/clients/add/protection',
        component: ProtectionRoute,
    },
    {
        path: '/clients/add',
        component: AddClientRoute,
    },
    {
        path: '/clients/edit/:clientName/blocked_services/schedule',
        component: ClientScheduleRoute,
    },
    {
        path: '/clients/edit/:clientName/blocked_services',
        component: ClientBlockedServicesRoute,
    },
    {
        path: '/clients/edit/:clientName/protection',
        component: ProtectionRoute,
    },
    {
        path: '/clients/edit/:clientName',
        component: AddClientRoute,
    },
    {
        path: '/clients',
        component: Clients,
    },
];

const App = () => {
    const dispatch = useDispatch();
    const { language, isCoreRunning, processing, theme } = useSelector<
        RootState,
        RootState['dashboard']
    >((state) => state.dashboard, shallowEqual);

    useEffect(() => {
        dispatch(getDnsStatus());

        const handleVisibilityChange = () => {
            if (document.visibilityState === 'visible') {
                dispatch(getTimerStatus());
            }
        };

        document.addEventListener('visibilitychange', handleVisibilityChange);

        return () => {
            document.removeEventListener('visibilitychange', handleVisibilityChange);
        };
    }, []);

    const setLanguage = () => {
        if (!processing) {
            if (language) {
                intl.changeLanguage(language as LocalesType);
                setHtmlLangAttr(language);
                LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, language);
            }
        }
    };

    useEffect(() => {
        setLanguage();
    }, [language]);

    const handleAutoTheme = (e: any, accountTheme: any) => {
        if (accountTheme !== THEMES.auto) {
            return;
        }

        if (e.matches) {
            setUITheme(THEMES.dark);
        } else {
            setUITheme(THEMES.light);
        }
    };

    useEffect(() => {
        if (theme !== THEMES.auto) {
            setUITheme(theme);

            return;
        }

        const colorSchemeMedia = window.matchMedia('(prefers-color-scheme: dark)');
        setUITheme(theme);

        if (colorSchemeMedia.addEventListener !== undefined) {
            colorSchemeMedia.addEventListener('change', (e) => {
                handleAutoTheme(e, theme);
            });
        } else {
            // Deprecated addListener for older versions of Safari.
            colorSchemeMedia.addListener((e) => {
                handleAutoTheme(e, theme);
            });
        }
    }, [theme]);

    return (
        <HashRouter>
            <Header />

            <div className={s.wrapper}>
                <Sidebar />

                <div className={s.bodyWrapper}>
                    {!processing && isCoreRunning && (
                        <Routes>
                            {ROUTES.map((route) => (
                                <Route
                                    key={route.path}
                                    path={route.path}
                                    element={<route.component />}
                                />
                            ))}
                            <Route path="/" element={<Navigate to="/dashboard" replace />} />
                        </Routes>
                    )}
                </div>
            </div>

            <Footer />

            <Toasts />

            <Icons />
        </HashRouter>
    );
};

export default App;
