import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';

import Form from './Form';
import Leases from './Leases';
import Interface from './Interface';
import Card from '../../ui/Card';

class Dhcp extends Component {
    handleFormSubmit = (values) => {
        this.props.setDhcpConfig(values);
    };

    handleFormChange = (value) => {
        this.props.setDhcpConfig(value);
    }

    handleToggle = (config) => {
        this.props.toggleDhcp(config);
        this.props.findActiveDhcp(config.interface_name);
    }

    getToggleDhcpButton = () => {
        const { config, active, processingDhcp } = this.props.dhcp;
        const activeDhcpFound = active && active.found;
        const filledConfig = Object.keys(config).every((key) => {
            if (key === 'enabled') {
                return true;
            }

            return config[key];
        });

        if (config.enabled) {
            return (
                <button
                    type="button"
                    className="btn btn-standart mr-2 btn-gray"
                    onClick={() => this.props.toggleDhcp(config)}
                    disabled={processingDhcp}
                >
                    <Trans>dhcp_disable</Trans>
                </button>
            );
        }

        return (
            <button
                type="button"
                className="btn btn-standart mr-2 btn-success"
                onClick={() => this.handleToggle(config)}
                disabled={!filledConfig || activeDhcpFound || processingDhcp}
            >
                <Trans>dhcp_enable</Trans>
            </button>
        );
    }

    getActiveDhcpMessage = () => {
        const { active } = this.props.dhcp;

        if (active) {
            if (active.error) {
                return (
                    <div className="text-danger">
                        {active.error}
                    </div>
                );
            }

            return (
                <Fragment>
                    {active.found ? (
                        <div className="text-danger">
                            <Trans>dhcp_found</Trans>
                        </div>
                    ) : (
                        <div className="text-secondary">
                            <Trans>dhcp_not_found</Trans>
                        </div>
                    )}
                </Fragment>
            );
        }

        return '';
    }

    render() {
        const { t, dhcp } = this.props;
        const statusButtonClass = classnames({
            'btn btn-primary btn-standart': true,
            'btn btn-primary btn-standart btn-loading': dhcp.processingStatus,
        });

        return (
            <Fragment>
                <Card title={ t('dhcp_title') } subtitle={ t('dhcp_description') } bodyType="card-body box-body--settings">
                    <div className="dhcp">
                        {!dhcp.processing &&
                            <Fragment>
                                <Interface
                                    onChange={this.handleFormChange}
                                    initialValues={dhcp.config}
                                    interfaces={dhcp.interfaces}
                                    processing={dhcp.processingInterfaces}
                                    enabled={dhcp.config.enabled}
                                />
                                <Form
                                    onSubmit={this.handleFormSubmit}
                                    initialValues={dhcp.config}
                                    interfaces={dhcp.interfaces}
                                    processing={dhcp.processingInterfaces}
                                />
                                <hr/>
                                <div className="card-actions mb-3">
                                    {this.getToggleDhcpButton()}
                                    <button
                                        type="button"
                                        className={statusButtonClass}
                                        onClick={() =>
                                            this.props.findActiveDhcp(dhcp.config.interface_name)
                                        }
                                        disabled={!dhcp.config.interface_name}
                                    >
                                        <Trans>check_dhcp_servers</Trans>
                                    </button>
                                </div>
                                {this.getActiveDhcpMessage()}
                            </Fragment>
                        }
                    </div>
                </Card>
                {!dhcp.processing && dhcp.config.enabled &&
                    <Card title={ t('dhcp_leases') } bodyType="card-body box-body--settings">
                        <div className="row">
                            <div className="col">
                                <Leases leases={dhcp.leases} />
                            </div>
                        </div>
                    </Card>
                }
            </Fragment>
        );
    }
}

Dhcp.propTypes = {
    dhcp: PropTypes.object,
    toggleDhcp: PropTypes.func,
    getDhcpStatus: PropTypes.func,
    setDhcpConfig: PropTypes.func,
    findActiveDhcp: PropTypes.func,
    handleSubmit: PropTypes.func,
    t: PropTypes.func,
};

export default withNamespaces()(Dhcp);
