import React, { Component, Fragment } from 'react';
import { connect } from 'react-redux';
import debounce from 'lodash/debounce';

import * as actionCreators from '../../actions/install';

import { getWebAddress } from '../../helpers/helpers';
import { INSTALL_FIRST_STEP, INSTALL_TOTAL_STEPS, ALL_INTERFACES_IP, DEBOUNCE_TIMEOUT } from '../../helpers/constants';

import Loading from '../../components/ui/Loading';

import Greeting from './Greeting';

import Settings from './Settings';

import Auth from './Auth';

import Devices from './Devices';

import Submit from './Submit';

import Progress from './Progress';

import Toasts from '../../components/Toasts';

import Footer from '../../components/ui/Footer';

import Icons from '../../components/ui/Icons';

import { Logo } from '../../components/ui/svg/logo';

import './Setup.css';
import '../../components/ui/Tabler.css';

interface SetupProps {
    getDefaultAddresses: (...args: unknown[]) => unknown;
    setAllSettings: (...args: unknown[]) => unknown;
    checkConfig: (...args: unknown[]) => unknown;
    nextStep: (...args: unknown[]) => unknown;
    prevStep: (...args: unknown[]) => unknown;
    install: {
        step: number;
        processingDefault: boolean;
        web;
        dns;
        staticIp;
        interfaces;
    };
    step?: number;
    web?: object;
    dns?: object;
}

class Setup extends Component<SetupProps> {
    componentDidMount() {
        this.props.getDefaultAddresses();
    }

    handleFormSubmit = (values: any) => {
        const { staticIp, ...config } = values;

        this.props.setAllSettings(config);
    };

    handleFormChange = debounce((values) => {
        const { web, dns } = values;
        if (values && web.port && dns.port) {
            this.props.checkConfig({ web, dns, set_static_ip: false });
        }
    }, DEBOUNCE_TIMEOUT);

    handleFix = (web: any, dns: any, set_static_ip: any) => {
        this.props.checkConfig({ web, dns, set_static_ip });
    };

    openDashboard = (ip: any, port: any) => {
        let address = getWebAddress(ip, port);

        if (ip === ALL_INTERFACES_IP) {
            address = getWebAddress(window.location.hostname, port);
        }

        window.location.replace(address);
    };

    nextStep = () => {
        if (this.props.install.step < INSTALL_TOTAL_STEPS) {
            this.props.nextStep();
        }
    };

    prevStep = () => {
        if (this.props.install.step > INSTALL_FIRST_STEP) {
            this.props.prevStep();
        }
    };

    renderPage(step: any, config: any, interfaces: any) {
        switch (step) {
            case 1:
                return <Greeting />;
            case 2:
                return (
                    <Settings
                        config={config}
                        initialValues={config}
                        interfaces={interfaces}
                        onSubmit={this.nextStep}
                        onChange={this.handleFormChange}
                        validateForm={this.handleFormChange}
                        handleFix={this.handleFix}
                    />
                );
            case 3:
                return <Auth onSubmit={this.handleFormSubmit} />;
            case 4:
                return <Devices interfaces={interfaces} />;
            case 5:
                return <Submit openDashboard={this.openDashboard} />;
            default:
                return false;
        }
    }

    render() {
        const { processingDefault, step, web, dns, staticIp, interfaces } = this.props.install;

        return (
            <Fragment>
                {processingDefault && <Loading />}
                {!processingDefault && (
                    <Fragment>
                        <div className="setup">
                            <div className="setup__container">
                                <Logo className="setup__logo" />
                                {this.renderPage(step, { web, dns, staticIp }, interfaces)}
                                <Progress step={step} />
                            </div>
                        </div>

                        <Footer />

                        <Toasts />

                        <Icons />
                    </Fragment>
                )}
            </Fragment>
        );
    }
}

const mapStateToProps = (state: any) => {
    const { install, toasts } = state;
    const props = { install, toasts };
    return props;
};

export default connect(mapStateToProps, actionCreators)(Setup);
