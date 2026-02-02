import React, { useEffect, useMemo } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import debounce from 'lodash/debounce';

import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { InstallInterface, InstallState, RootState } from 'panel/initialState';
import { SetupGuide } from 'panel/components/SetupGuide/SetupGuide';
import * as actionCreators from '../../actions/install';

import { getInterfaceIp, getWebAddress } from '../../helpers/helpers';
import { INSTALL_TOTAL_STEPS, ALL_INTERFACES_IP, DEBOUNCE_TIMEOUT } from '../../helpers/constants';

import Greeting from './Greeting';
import type { ConfigType, DnsConfig, WebConfig } from './types';
import { InterfaceSettings } from './InterfaceSettings';
import { DnsSettings } from './DnsSettings';
import Controls from './Controls';
import { Submit } from './Submit';
import { Progress } from './blocks/Progress';
import { Auth } from './Auth';
import Toasts from '../../components/Toasts';

import styles from './styles.module.pcss';
import twosky from '../../../../.twosky.json';
import { getDnsAddressWithPort } from './helpers/helpers';

const LANGUAGES = twosky[1].languages;

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

    const checkConfig = useMemo(
        () =>
            debounce((values) => {
                const { web, dns } = values;

                if (values && web.port && dns.port) {
                    dispatch(actionCreators.checkConfig({ web, dns, set_static_ip: false }));
                }
            }, DEBOUNCE_TIMEOUT),
        [dispatch],
    );

    useEffect(() => () => checkConfig.cancel(), [checkConfig]);

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
                <PublicHeader
                    languages={LANGUAGES}
                    dropdownClassName={styles.dropdown}
                    dropdownPosition="bottomRight"
                    center={<Progress step={step} />}
                />

                <div className={styles.container}>{renderPage(step, { web, dns, staticIp }, interfaces)}</div>
            </div>

            <Toasts />
        </>
    );
};
