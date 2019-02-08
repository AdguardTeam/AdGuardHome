import React, { Component, Fragment } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';

import * as actionCreators from '../../actions/install';
import { INSTALL_FIRST_STEP, INSTALL_TOTAL_STEPS } from '../../helpers/constants';

import Loading from '../../components/ui/Loading';
import Greeting from './Greeting';
import Settings from './Settings';
import Auth from './Auth';
import Devices from './Devices';
import Submit from './Submit';
import Progress from './Progress';

import Toasts from '../../components/Toasts';
import Footer from '../../components/ui/Footer';
import logo from '../../components/ui/svg/logo.svg';

import './Setup.css';
import '../../components/ui/Tabler.css';

class Setup extends Component {
    componentDidMount() {
        this.props.getDefaultAddresses();
    }

    handleFormSubmit = (values) => {
        this.props.setAllSettings(values);
    };

    openDashboard = (address) => {
        window.location.replace(address);
    }

    nextStep = () => {
        if (this.props.install.step < INSTALL_TOTAL_STEPS) {
            this.props.nextStep();
        }
    }

    prevStep = () => {
        if (this.props.install.step > INSTALL_FIRST_STEP) {
            this.props.prevStep();
        }
    }

    renderPage(step, config, interfaces) {
        switch (step) {
            case 1:
                return <Greeting />;
            case 2:
                return (
                    <Settings
                        initialValues={config}
                        interfaces={interfaces}
                        webWarning={config.web.warning}
                        dnsWarning={config.dns.warning}
                        onSubmit={this.nextStep}
                    />
                );
            case 3:
                return (
                    <Auth onSubmit={this.handleFormSubmit} />
                );
            case 4:
                return <Devices interfaces={interfaces} />;
            case 5:
                return <Submit openDashboard={this.openDashboard} />;
            default:
                return false;
        }
    }

    render() {
        const {
            processingDefault,
            step,
            web,
            dns,
            interfaces,
        } = this.props.install;

        return (
            <Fragment>
                {processingDefault && <Loading />}
                {!processingDefault &&
                    <Fragment>
                        <div className="setup">
                            <div className="setup__container">
                                <img src={logo} className="setup__logo" alt="logo" />
                                {this.renderPage(step, { web, dns }, interfaces)}
                                <Progress step={step} />
                            </div>
                        </div>
                        <Footer />
                        <Toasts />
                    </Fragment>
                }
            </Fragment>
        );
    }
}

Setup.propTypes = {
    getDefaultAddresses: PropTypes.func.isRequired,
    setAllSettings: PropTypes.func.isRequired,
    nextStep: PropTypes.func.isRequired,
    prevStep: PropTypes.func.isRequired,
    install: PropTypes.object.isRequired,
    step: PropTypes.number,
    web: PropTypes.object,
    dns: PropTypes.object,
};

const mapStateToProps = (state) => {
    const { install, toasts } = state;
    const props = { install, toasts };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(Setup);
