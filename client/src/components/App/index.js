import React, { useEffect } from 'react';
import { HashRouter, Route } from 'react-router-dom';
import LoadingBar from 'react-redux-loading-bar';
import { hot } from 'react-hot-loader/root';

import 'react-table/react-table.css';
import '../ui/Tabler.css';
import '../ui/ReactTable.css';
import './index.css';

import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import propTypes from 'prop-types';
import Toasts from '../Toasts';
import Footer from '../ui/Footer';
import Status from '../ui/Status';
import UpdateTopline from '../ui/UpdateTopline';
import UpdateOverlay from '../ui/UpdateOverlay';
import EncryptionTopline from '../ui/EncryptionTopline';
import Icons from '../ui/Icons';
import i18n from '../../i18n';
import Loading from '../ui/Loading';
import { FILTERS_URLS, MENU_URLS, SETTINGS_URLS } from '../../helpers/constants';
import { getLogsUrlParams, setHtmlLangAttr } from '../../helpers/helpers';
import Header from '../Header';
import { changeLanguage, getDnsStatus } from '../../actions';

import Dashboard from '../../containers/Dashboard';
import Logs from '../../containers/Logs';
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


const ROUTES = [
    {
        path: MENU_URLS.root,
        component: Dashboard,
        exact: true,
    },
    {
        path: [`${MENU_URLS.logs}${getLogsUrlParams(':search?', ':response_status?')}`, MENU_URLS.logs],
        component: Logs,
    },
    {
        path: MENU_URLS.guide,
        component: SetupGuide,
    },
    {
        path: SETTINGS_URLS.settings,
        component: Settings,
    },
    {
        path: SETTINGS_URLS.dns,
        component: Dns,
    },
    {
        path: SETTINGS_URLS.encryption,
        component: Encryption,
    },
    {
        path: SETTINGS_URLS.dhcp,
        component: Dhcp,
    },
    {
        path: SETTINGS_URLS.clients,
        component: Clients,
    },
    {
        path: FILTERS_URLS.dns_blocklists,
        component: DnsBlocklist,
    },
    {
        path: FILTERS_URLS.dns_allowlists,
        component: DnsAllowlist,
    },
    {
        path: FILTERS_URLS.dns_rewrites,
        component: DnsRewrites,
    },
    {
        path: FILTERS_URLS.custom_rules,
        component: CustomRules,
    },
    {
        path: FILTERS_URLS.blocked_services,
        component: Services,
    },
];

const renderRoute = ({ path, component, exact }, idx) => <Route
        key={idx}
        exact={exact}
        path={path}
        component={component}
/>;

const App = () => {
    const dispatch = useDispatch();
    const {
        language,
        isCoreRunning,
        isUpdateAvailable,
        processing,
    } = useSelector((state) => state.dashboard, shallowEqual);

    const { processing: processingEncryption } = useSelector((
        state,
    ) => state.encryption, shallowEqual);

    const updateAvailable = isCoreRunning && isUpdateAvailable;

    useEffect(() => {
        dispatch(getDnsStatus());
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

    const reloadPage = () => {
        window.location.reload();
    };

    return <HashRouter hashType="noslash">
        {updateAvailable && <>
            <UpdateTopline />
            <UpdateOverlay />
        </>}
        {!processingEncryption && <EncryptionTopline />}
        <LoadingBar className="loading-bar" updateTime={1000} />
        <Header />
        <div className="container container--wrap pb-5">
            {processing && <Loading />}
            {!isCoreRunning && <div className="row row-cards">
                <div className="col-lg-12">
                    <Status reloadPage={reloadPage} message="dns_start" />
                    <Loading />
                </div>
            </div>}
            {!processing && isCoreRunning && ROUTES.map(renderRoute)}
        </div>
        <Footer />
        <Toasts />
        <Icons />
    </HashRouter>;
};

renderRoute.propTypes = {
    path: propTypes.oneOfType([propTypes.string, propTypes.arrayOf(propTypes.string)]).isRequired,
    component: propTypes.element.isRequired,
    exact: propTypes.bool,
};

export default hot(App);
