import React, { useEffect, Fragment } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import debounce from 'lodash/debounce';

import intl, { LocalesType } from 'panel/common/intl'; // путь подстрой под свой
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import s from 'panel/common/ui/Header/Header.module.pcss';
import { Logo } from 'panel/common/ui/Sidebar';
import { InstallInterface, InstallState, RootState } from 'panel/initialState';
import * as actionCreators from '../../actions/install';

import { getWebAddress, setHtmlLangAttr } from '../../helpers/helpers';
import { INSTALL_TOTAL_STEPS, ALL_INTERFACES_IP, DEBOUNCE_TIMEOUT } from '../../helpers/constants';

import Greeting from './Greeting';
import { ConfigType, DnsConfig, Settings, WebConfig } from './Settings';
import { InterfaceSettings } from './InterfaceSettings'
import { DnsSettings } from './DnsSettings'
import { Devices } from './Devices';
import { Submit } from './Submit';
import { Progress } from './Progress';
import { Auth } from './Auth';
import Toasts from '../../components/Toasts';

import './Setup.css';
import twosky from '../../../../.twosky.json';
const LANGUAGES = twosky[1].languages;
import { changeLanguage as changeLanguageAction } from '../../actions';
import { LanguageDropdown } from '../../common/ui/LanguageDropdown/LanguageDropdown';

export const Setup = () => {
    const dispatch = useDispatch();

    const install = useSelector((state: InstallState) => state.install);
    const { processingDefault, step, web, dns, staticIp, interfaces } = install;

    useEffect(() => {
        dispatch(actionCreators.getDefaultAddresses());
    }, []);

    const handleFormSubmit = (values: any) => {
        const config = { ...values };
        delete config.staticIp;

        if (web.port && dns.port) {
            dispatch(
                actionCreators.setAllSettings({
                    web,
                    dns,
                    ...config,
                }),
            );
        }
    };

    const checkConfig = debounce((values) => {
        const { web, dns } = values;

        if (values && web.port && dns.port) {
            dispatch(actionCreators.checkConfig({ web, dns, set_static_ip: false }));
        }
    }, DEBOUNCE_TIMEOUT);

    const handleFix = (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => {
        dispatch(actionCreators.checkConfig({ web, dns, set_static_ip }));
    };

    const openDashboard = (ip: string, port: number) => {
        let address = getWebAddress(ip, port);
        if (ip === ALL_INTERFACES_IP) {
            address = getWebAddress(window.location.hostname, port);
        }
        window.location.replace(address);
    };

    const handleNextStep = () => {
        if (step < INSTALL_TOTAL_STEPS) {
            dispatch(actionCreators.nextStep());
        }
    };

    const currentLanguage =
        useSelector((state: RootState) => (state.dashboard ? state.dashboard.language : '')) || intl.getUILanguage();

    const changeLanguage = async (newLang: LocalesType) => {
        setHtmlLangAttr(newLang);

        try {
            await dispatch(changeLanguageAction(newLang));
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, newLang);
            window.location.reload();
        } catch (error) {
            console.error('Failed to save language preference:', error);
        }
    };

    const renderPage = (step: number, config: ConfigType, interfaces: InstallInterface[]) => {
        switch (step) {
            case 1:
                return <Greeting />;
            case 2:
                return <Auth onAuthSubmit={handleFormSubmit} />;
            case 3:
                return (
                    <InterfaceSettings
                        config={config}
                        initialValues={config}
                        interfaces={interfaces}
                        handleSubmit={handleNextStep}
                        validateForm={checkConfig}
                        handleFix={handleFix}
                    />
                );
            case 4:
                return (
                    <DnsSettings
                        config={config}
                        initialValues={config}
                        interfaces={interfaces}
                        handleSubmit={handleNextStep}
                        validateForm={checkConfig}
                        handleFix={handleFix}
                    />
                );
            case 5:
                return <Submit openDashboard={openDashboard} webConfig={web} />;
            default:
                return false;
        }
    };

    if (processingDefault) {
        return null;
    }

    return (
        <>
            <div className="setup">

                <div className="setup__header">
                    <div className="setup__header-content">
                        <div className={s.linkWrapper}>
                            <Logo id="header" />
                        </div>
                        <Progress step={step} />
                        <LanguageDropdown
                        value={currentLanguage}
                        languages={LANGUAGES}
                        onChange={(lang) => changeLanguage(lang as LocalesType)}
                        className={s.dropdown}
                        position="bottomRight" />
                    </div>
                </div>

                <div className="setup__container">
                    {renderPage(step, { web, dns, staticIp }, interfaces)}
                </div>
            </div>

            <Toasts />
        </>
    );
};
