import React, { useEffect, ComponentType } from 'react';

import { HashRouter, Route } from 'react-router-dom';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import { Sidebar } from 'panel/common/ui/Sidebar';
import { Icons } from 'panel/common/ui/Icons';
import { Footer } from 'panel/common/ui/Footer';
import { Header } from 'panel/common/ui/Header';
import { Settings } from 'panel/components/Settings';
import { LocalesType } from 'panel/common/intl';
import { Encryption } from 'panel/components/Encryption';
import { Blocklists } from 'panel/components/FilterLists/Blocklists';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';

import { Allowlists } from 'panel/components/FilterLists/Allowlists';
import { DNSRewrites } from 'panel/components/FilterLists/DNSRewrites';
import { SetupGuide } from 'panel/components/SetupGuide';
import { Dashboard } from 'panel/components/Dashboard';
import Toasts from '../Toasts';
import i18n from '../../i18n';
import { THEMES } from '../../helpers/constants';
import { setHtmlLangAttr, setUITheme } from '../../helpers/helpers';
import { changeLanguage, getDnsStatus, getTimerStatus } from '../../actions';
import { RootState } from '../../initialState';

import s from './styles.module.pcss';
import { DnsSettings } from '../DnsSettings';

type RouteConfig = {
    path: string;
    component: ComponentType;
    exact: boolean;
};

const ROUTES: RouteConfig[] = [
    {
        path: '/dashboard',
        component: Dashboard,
        exact: true,
    },
    {
        path: '/settings',
        component: Settings,
        exact: true,
    },
    {
        path: '/encryption',
        component: Encryption,
        exact: true,
    },
    {
        path: '/dns',
        component: DnsSettings,
        exact: true,
    },
    {
        path: '/blocklists',
        component: Blocklists,
        exact: true,
    },
    {
        path: '/allowlists',
        component: Allowlists,
        exact: true,
    },
    {
        path: '/dns_rewrites',
        component: DNSRewrites,
        exact: true,
    },
    {
        path: '/guide',
        component: SetupGuide,
        exact: true,
    },
];

const App = () => {
    const dispatch = useDispatch();
    const { language, isCoreRunning, processing, theme } = useSelector<RootState, RootState['dashboard']>(
        (state) => state.dashboard,
        shallowEqual,
    );

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
                i18n.changeLanguage(language);
                setHtmlLangAttr(language);
                LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, language);
            }
        }

        i18n.on('languageChanged', (lang: LocalesType) => {
            dispatch(changeLanguage(lang));
        });
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
        <HashRouter hashType="noslash">
            <Header />

            <div className={s.wrapper}>
                <Sidebar />

                <div className={s.bodyWrapper}>
                    {!processing &&
                        isCoreRunning &&
                        ROUTES.map((route, index) => (
                            <Route key={index} exact={route.exact} path={route.path} component={route.component} />
                        ))}
                </div>
            </div>

            <Footer />

            <Toasts />

            <Icons />
        </HashRouter>
    );
};

export default App;
