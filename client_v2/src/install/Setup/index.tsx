import React, { useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import debounce from 'lodash/debounce';

import intl, { LocalesType } from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import s from 'panel/common/ui/Header/Header.module.pcss';
import { Logo } from 'panel/common/ui/Sidebar';
import { InstallInterface, InstallState, RootState } from 'panel/initialState';
import { SetupGuide } from 'panel/components/SetupGuide/SetupGuide';
import * as actionCreators from '../../actions/install';
import { stripZoneId } from './helpers';

import { getInterfaceIp, getWebAddress, setHtmlLangAttr } from '../../helpers/helpers';
import { INSTALL_TOTAL_STEPS, ALL_INTERFACES_IP, DEBOUNCE_TIMEOUT } from '../../helpers/constants';

import Greeting from './Greeting';
import type { ConfigType, DnsConfig, WebConfig } from './types';
import { InterfaceSettings } from './InterfaceSettings';
import { DnsSettings } from './DnsSettings';
import Controls from './Controls';
import { Submit } from './Submit';
import { Progress } from './Progress';
import { Auth } from './Auth';
import Toasts from '../../components/Toasts';

import styles from './styles.module.pcss';
import twosky from '../../../../.twosky.json';
import { changeLanguage as changeLanguageAction } from '../../actions';
import { LanguageDropdown } from '../../common/ui/LanguageDropdown/LanguageDropdown';

const LANGUAGES = twosky[1].languages;

const getDnsAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `[${normalizedIp}]:${port}`;
    }

    return `${normalizedIp}:${port}`;
};

const getInstallDnsAddresses = (dns: { ip: string; port: number }, interfaces: InstallInterface[]) => {
    if (!dns?.ip || !dns?.port) {
        return [];
    }

    if (dns.ip === ALL_INTERFACES_IP) {
        return (interfaces || [])
            .filter((iface: InstallInterface) => iface?.ip_addresses?.length > 0)
            .map((iface: InstallInterface) => getInterfaceIp(iface))
            .filter(Boolean)
            .map((ip: string) => getDnsAddressWithPort(ip, dns.port));
    }

    return [getDnsAddressWithPort(dns.ip, dns.port)];
};

export const Setup = () => {
    const dispatch = useDispatch();

    const install = useSelector((state: InstallState) => state.install);
    const { processingDefault, step, web, dns, staticIp, interfaces, auth } = install;
    const dnsAddresses = useSelector((state: RootState) => state.dashboard?.dnsAddresses || []);
    const installDnsAddresses = getInstallDnsAddresses(dns, interfaces);
    const resolvedDnsAddresses = dnsAddresses.length > 0 ? dnsAddresses : installDnsAddresses;

    useEffect(() => {
        dispatch(actionCreators.getDefaultAddresses());
    }, []);

    const handleNextStep = () => {
        if (step <= INSTALL_TOTAL_STEPS) {
            dispatch(actionCreators.nextStep());
        }
    };

    const handleAuthSubmit = (values: any) => {
        dispatch(actionCreators.setAuthData(values));
        handleNextStep();
    };

    const handleFinalSubmit = () => {
        const config: any = {
            web,
            dns,
            ...auth,
        };

        if (web.port && dns.port) {
            dispatch(actionCreators.setAllSettings(config));
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
        window.location.replace(`${address}#dashboard`);
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
                return <Auth onAuthSubmit={handleAuthSubmit} />;
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
                return (
                    <>
                        <SetupGuide dnsAddresses={resolvedDnsAddresses} isStep footer={<Controls />} />
                    </>
                );
            case 6:
                return (
                    <Submit
                        openDashboard={openDashboard}
                        webConfig={web}
                        onSubmit={handleFinalSubmit}
                    />
                );
            default:
                return false;
        }
    };

    if (processingDefault) {
        return null;
    }

    return (
        <>
            <div className={styles.setup}>
                <div className={styles.header}>
                    <div className={styles.headerContent}>
                        <div className={styles.logoWrap}>
                            <Logo id="header" />
                        </div>
                        <Progress step={step} />
                        <div className={styles.languageWrap}>
                            <LanguageDropdown
                                value={currentLanguage}
                                languages={LANGUAGES}
                                onChange={(lang) => changeLanguage(lang as LocalesType)}
                                className={s.dropdown}
                                position="bottomRight"
                            />
                        </div>
                    </div>
                </div>

                <div className={styles.container}>{renderPage(step, { web, dns, staticIp }, interfaces)}</div>
            </div>

            <Toasts />
        </>
    );
};
