import React, { useEffect } from 'react';

import { HashRouter, Route, Routes } from 'react-router-dom';
import LoadingBar from 'react-redux-loading-bar';

import '../ui/Tabler.css';
import '../ui/ReactTable.css';
import './index.css';

import { shallowEqual } from 'react-redux';
import { useDispatch, useSelector } from '../../store/hooks';
import Toasts from '../Toasts';
import Footer from '../ui/Footer';
import Status from '../ui/Status';
import UpdateTopline from '../ui/UpdateTopline';
import UpdateOverlay from '../ui/UpdateOverlay';
import EncryptionTopline from '../ui/EncryptionTopline';
import Icons from '../ui/Icons';
import i18n from '../../i18n';

import Loading from '../ui/Loading';
import { FILTERS_URLS, MENU_URLS, SETTINGS_URLS, THEMES } from '../../helpers/constants';

import { getLogsUrlParams, setHtmlLangAttr, setUITheme } from '../../helpers/helpers';

import Header from '../Header';

import { getDnsStatus, getTimerStatus } from '../../actions';

import Dashboard from '../../containers/Dashboard';
import SetupGuide from '../../containers/SetupGuide';
import Settings from '../../containers/Settings';
import Dns from '../../containers/Dns';
import Encryption from '../../containers/Encryption';

import Dhcp from '../Settings/Dhcp';
import Clients from '../../containers/Clients';
import DnsBlocklist from '../../containers/DnsBlocklist';
import DnsAllowlist from '../../containers/DnsAllowlist';
import DnsRewrites from '../../containers/DnsRewrites';
import CustomRules from '../../containers/CustomRules';

import Services from '../Filters/Services';

import Logs from '../Logs';
import ProtectionTimer from '../ProtectionTimer';
import { RootState } from '../../initialState';

const ROUTES = [
    {
        path: MENU_URLS.root,
        element: <Dashboard />,
    },
    {
        path: `${MENU_URLS.logs}${getLogsUrlParams(':search?', ':response_status?')}`,
        element: <Logs />,
    },
    {
        path: MENU_URLS.logs,
        element: <Logs />,
    },
    {
        path: MENU_URLS.guide,
        element: <SetupGuide />,
    },
    {
        path: SETTINGS_URLS.settings,
        element: <Settings />,
    },
    {
        path: SETTINGS_URLS.dns,
        element: <Dns />,
    },
    {
        path: SETTINGS_URLS.encryption,
        element: <Encryption />,
    },
    {
        path: SETTINGS_URLS.dhcp,
        element: <Dhcp />,
    },
    {
        path: SETTINGS_URLS.clients,
        element: <Clients />,
    },
    {
        path: FILTERS_URLS.dns_blocklists,
        element: <DnsBlocklist />,
    },
    {
        path: FILTERS_URLS.dns_allowlists,
        element: <DnsAllowlist />,
    },
    {
        path: FILTERS_URLS.dns_rewrites,
        element: <DnsRewrites />,
    },
    {
        path: FILTERS_URLS.custom_rules,
        element: <CustomRules />,
    },
    {
        path: FILTERS_URLS.blocked_services,
        element: <Services />,
    },
];

const App = () => {
    const dispatch = useDispatch();
    const { language, isCoreRunning, isUpdateAvailable, processing, theme } = useSelector<
        RootState,
        RootState['dashboard']
    >((state) => state.dashboard, shallowEqual);

    const { processing: processingEncryption } = useSelector<RootState, RootState['encryption']>(
        (state) => state.encryption,
        shallowEqual,
    );

    const updateAvailable = isCoreRunning && isUpdateAvailable;

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
        if (processing || !language) {
            return;
        }

        i18n.changeLanguage(language);
        setHtmlLangAttr(language);
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

    const reloadPage = () => {
        window.location.reload();
    };

    return (
        <HashRouter>
            {updateAvailable && (
                <>
                    <UpdateTopline />

                    <UpdateOverlay />
                </>
            )}

            {!processingEncryption && <EncryptionTopline />}

            <LoadingBar className="loading-bar" updateTime={1000} />

            <Header />

            <ProtectionTimer />

            <div className="container container--wrap pb-5 pt-5">
                {processing && <Loading />}

                {!isCoreRunning && (
                    <div className="row row-cards">
                        <div className="col-lg-12">
                            <Status reloadPage={reloadPage} message="dns_start" />

                            <Loading />
                        </div>
                    </div>
                )}
                {!processing && isCoreRunning && (
                    <Routes>
                        {ROUTES.map((route, index) => (
                            <Route key={index} path={route.path} element={route.element} />
                        ))}
                    </Routes>
                )}
            </div>

            <Footer />

            <Toasts />

            <Icons />
        </HashRouter>
    );
};

export default App;
