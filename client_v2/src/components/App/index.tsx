import React, { useEffect } from 'react';

import { HashRouter, Route } from 'react-router-dom';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import { Sidebar, Icons, Footer, Header } from 'panel/common/ui';

import Toasts from '../Toasts';
import i18n from '../../i18n';

import { THEMES } from '../../helpers/constants';
import { setHtmlLangAttr, setUITheme } from '../../helpers/helpers';
import { changeLanguage, getDnsStatus, getTimerStatus } from '../../actions';

import { RootState } from '../../initialState';
import Expo from '../Expo';

import s from './styles.module.pcss';

type RouteConfig = {
    path: string;
    component: React.ComponentType;
    exact: boolean;
};

const ROUTES: RouteConfig[] = [];

if (process.env.NODE_ENV === 'development') {
    ROUTES.push({
        path: '/expo',
        component: Expo,
        exact: true,
    });
}

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
            }
        }

        i18n.on('languageChanged', (lang) => {
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
