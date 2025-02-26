import React, { useEffect, Fragment } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import debounce from 'lodash/debounce';

import * as actionCreators from '../../actions/install';

import { getWebAddress } from '../../helpers/helpers';
import { INSTALL_TOTAL_STEPS, ALL_INTERFACES_IP, DEBOUNCE_TIMEOUT } from '../../helpers/constants';

import Loading from '../../components/ui/Loading';
import Greeting from './Greeting';
import { ConfigType, DnsConfig, Settings, WebConfig } from './Settings';
import { Devices } from './Devices';
import { Submit } from './Submit';
import { Progress } from './Progress';
import { Auth } from './Auth';
import Toasts from '../../components/Toasts';
import Footer from '../../components/ui/Footer';
import Icons from '../../components/ui/Icons';
import { Logo } from '../../components/ui/svg/logo';

import './Setup.css';
import '../../components/ui/Tabler.css';
import { InstallInterface, InstallState } from '../../initialState';

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

    const renderPage = (step: number, config: ConfigType, interfaces: InstallInterface[]) => {
        switch (step) {
            case 1:
                return <Greeting />;
            case 2:
                return (
                    <Settings
                        config={config}
                        initialValues={config}
                        interfaces={interfaces}
                        handleSubmit={handleNextStep}
                        validateForm={checkConfig}
                        handleFix={handleFix}
                    />
                );
            case 3:
                return <Auth onAuthSubmit={handleFormSubmit} />;
            case 4:
                return <Devices interfaces={interfaces} dnsConfig={dns} />;
            case 5:
                return <Submit openDashboard={openDashboard} webConfig={web} />;
            default:
                return false;
        }
    };

    if (processingDefault) {
        return <Loading />;
    }

    return (
        <>
            <div className="setup">
                <div className="setup__container">
                    <Logo className="setup__logo" />
                    {renderPage(step, { web, dns, staticIp }, interfaces)}
                    <Progress step={step} />
                </div>
            </div>

            <Footer />

            <Toasts />

            <Icons />
        </>
    );
};
