import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { Trans, withTranslation } from 'react-i18next';

import { DHCP_STATUS_RESPONSE } from '../../../helpers/constants';
import Form from './Form';
import Leases from './Leases';
import StaticLeases from './StaticLeases/index';
import Card from '../../ui/Card';
import Accordion from '../../ui/Accordion';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Dhcp extends Component {
    componentDidMount() {
        this.props.getDhcpStatus();
        this.props.getDhcpInterfaces();
    }

    handleFormSubmit = (values) => {
        if (values.interface_name) {
            this.props.setDhcpConfig(values);
        }
    };

    handleToggle = (config) => {
        this.props.toggleDhcp(config);
    };

    getToggleDhcpButton = () => {
        const {
            config, check, processingDhcp, processingConfig,
        } = this.props.dhcp;
        const otherDhcpFound = check?.otherServer
            && check.otherServer.found === DHCP_STATUS_RESPONSE.YES;
        const filledConfig = Object.keys(config).every((key) => {
            if (key === 'enabled' || key === 'icmp_timeout_msec') {
                return true;
            }

            return config[key];
        });

        if (config.enabled) {
            return (
                <button
                    type="button"
                    className="btn btn-standard mr-2 btn-gray"
                    onClick={() => this.props.toggleDhcp(config)}
                    disabled={processingDhcp || processingConfig}
                >
                    <Trans>dhcp_disable</Trans>
                </button>
            );
        }

        return (
            <button
                type="button"
                className="btn btn-standard mr-2 btn-success"
                onClick={() => this.handleToggle(config)}
                disabled={
                    !filledConfig || !check || otherDhcpFound || processingDhcp || processingConfig
                }
            >
                <Trans>dhcp_enable</Trans>
            </button>
        );
    };

    getActiveDhcpMessage = (t, check) => {
        const { found } = check.otherServer;

        if (found === DHCP_STATUS_RESPONSE.ERROR) {
            return (
                <div className="text-danger mb-2">
                    <Trans>dhcp_error</Trans>
                    <div className="mt-2 mb-2">
                        <Accordion label={t('error_details')}>
                            <span>{check.otherServer.error}</span>
                        </Accordion>
                    </div>
                </div>
            );
        }

        return (
            <div className="mb-2">
                {found === DHCP_STATUS_RESPONSE.YES ? (
                    <div className="text-danger">
                        <Trans>dhcp_found</Trans>
                    </div>
                ) : (
                    <div className="text-secondary">
                        <Trans>dhcp_not_found</Trans>
                    </div>
                )}
            </div>
        );
    };

    getDhcpWarning = (check) => {
        if (check.otherServer.found === DHCP_STATUS_RESPONSE.NO) {
            return '';
        }

        return (
            <div className="text-danger">
                <Trans>dhcp_warning</Trans>
            </div>
        );
    };

    getStaticIpWarning = (t, check, interfaceName) => {
        if (check.staticIP.static === DHCP_STATUS_RESPONSE.ERROR) {
            return (
                <Fragment>
                    <div className="text-danger mb-2">
                        <Trans>dhcp_static_ip_error</Trans>
                        <div className="mt-2 mb-2">
                            <Accordion label={t('error_details')}>
                                <span>{check.staticIP.error}</span>
                            </Accordion>
                        </div>
                    </div>
                    <hr className="mt-4 mb-4" />
                </Fragment>
            );
        } if (
            check.staticIP.static === DHCP_STATUS_RESPONSE.NO
            && check.staticIP.ip
            && interfaceName
        ) {
            return (
                <Fragment>
                    <div className="text-secondary mb-2">
                        <Trans
                            components={[<strong key="0">example</strong>]}
                            values={{
                                interfaceName,
                                ipAddress: check.staticIP.ip,
                            }}
                        >
                            dhcp_dynamic_ip_found
                        </Trans>
                    </div>
                    <hr className="mt-4 mb-4" />
                </Fragment>
            );
        }

        return '';
    };

    render() {
        const {
            t,
            dhcp,
            resetDhcp,
            findActiveDhcp,
            addStaticLease,
            removeStaticLease,
            toggleLeaseModal,
        } = this.props;
        const statusButtonClass = classnames({
            'btn btn-primary btn-standard': true,
            'btn btn-primary btn-standard btn-loading': dhcp.processingStatus,
        });
        const { enabled, interface_name, ...values } = dhcp.config;

        return (
            <Fragment>
                <PageTitle title={t('dhcp_settings')} />
                {(dhcp.processing || dhcp.processingInterfaces) && <Loading />}
                {!dhcp.processing && !dhcp.processingInterfaces && (
                    <Fragment>
                        <Card
                            title={t('dhcp_title')}
                            subtitle={t('dhcp_description')}
                            bodyType="card-body box-body--settings"
                        >
                            <div className="dhcp">
                                <Fragment>
                                    <Form
                                        onSubmit={this.handleFormSubmit}
                                        initialValues={{
                                            interface_name,
                                            ...values,
                                        }}
                                        interfaces={dhcp.interfaces}
                                        processingConfig={dhcp.processingConfig}
                                        processingInterfaces={dhcp.processingInterfaces}
                                        enabled={enabled}
                                        resetDhcp={resetDhcp}
                                    />
                                    <hr />
                                    <div className="card-actions mb-3">
                                        {this.getToggleDhcpButton()}
                                        <button
                                            type="button"
                                            className={statusButtonClass}
                                            onClick={() => findActiveDhcp(interface_name)}
                                            disabled={
                                                enabled || !interface_name || dhcp.processingConfig
                                            }
                                        >
                                            <Trans>check_dhcp_servers</Trans>
                                        </button>
                                    </div>
                                    {!enabled && dhcp.check && (
                                        <Fragment>
                                            {this.getStaticIpWarning(t, dhcp.check, interface_name)}
                                            {this.getActiveDhcpMessage(t, dhcp.check)}
                                            {this.getDhcpWarning(dhcp.check)}
                                        </Fragment>
                                    )}
                                </Fragment>
                            </div>
                        </Card>
                        {dhcp.config.enabled && (
                            <Card
                                title={t('dhcp_leases')}
                                bodyType="card-body box-body--settings"
                            >
                                <div className="row">
                                    <div className="col">
                                        <Leases leases={dhcp.leases} />
                                    </div>
                                </div>
                            </Card>
                        )}
                        <Card
                            title={t('dhcp_static_leases')}
                            bodyType="card-body box-body--settings"
                        >
                            <div className="row">
                                <div className="col-12">
                                    <StaticLeases
                                        staticLeases={dhcp.staticLeases}
                                        isModalOpen={dhcp.isModalOpen}
                                        addStaticLease={addStaticLease}
                                        removeStaticLease={removeStaticLease}
                                        toggleLeaseModal={toggleLeaseModal}
                                        processingAdding={dhcp.processingAdding}
                                        processingDeleting={dhcp.processingDeleting}
                                    />
                                </div>
                                <div className="col-12">
                                    <button
                                        type="button"
                                        className="btn btn-success btn-standard mt-3"
                                        onClick={() => toggleLeaseModal()}
                                    >
                                        <Trans>dhcp_add_static_lease</Trans>
                                    </button>
                                </div>
                            </div>
                        </Card>
                    </Fragment>
                )}
            </Fragment>
        );
    }
}

Dhcp.propTypes = {
    dhcp: PropTypes.object.isRequired,
    toggleDhcp: PropTypes.func.isRequired,
    getDhcpStatus: PropTypes.func.isRequired,
    setDhcpConfig: PropTypes.func.isRequired,
    findActiveDhcp: PropTypes.func.isRequired,
    addStaticLease: PropTypes.func.isRequired,
    removeStaticLease: PropTypes.func.isRequired,
    toggleLeaseModal: PropTypes.func.isRequired,
    getDhcpInterfaces: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
    resetDhcp: PropTypes.func.isRequired,
};

export default withTranslation()(Dhcp);
