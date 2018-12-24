import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';

import Form from './Form';
import Leases from './Leases';
import Card from '../../ui/Card';

class Dhcp extends Component {
    handleFormSubmit = (values) => {
        this.props.setDhcpConfig(values);
    };

    handleRefresh = () => {
        this.props.findActiveDhcp();
    }

    getToggleDhcpButton = () => {
        const { config } = this.props.dhcp;
        const buttonText = config.enabled ? 'dhcp_disable' : 'dhcp_enable';
        const buttonClass = config.enabled ? 'btn-gray' : 'btn-success';

        return (
            <button type="button" className={`btn btn-standart mr-2 ${buttonClass}`} onClick={() => this.props.toggleDhcp(config)}>
                <Trans>{buttonText}</Trans>
            </button>
        );
    }

    render() {
        const { t, dhcp } = this.props;
        const statusButtonClass = classnames({
            'btn btn-primary btn-standart': true,
            'btn btn-primary btn-standart btn-loading': dhcp.processingStatus,
        });

        return (
            <Fragment>
                {!dhcp.processing &&
                    <Card title={ t('dhcp_title') } subtitle={ t('dhcp_description') } bodyType="card-body box-body--settings">
                        <div className="row">
                            <div className="col">
                                <div className="card-actions mt-0 mb-3">
                                    {this.getToggleDhcpButton()}
                                    <button
                                        className={statusButtonClass}
                                        type="button"
                                        onClick={this.handleRefresh}
                                    >
                                        <Trans>refresh_status</Trans>
                                    </button>
                                </div>
                                {dhcp.active && !dhcp.active.found &&
                                    <div className="text-secondary">
                                        <Trans>dhcp_not_found</Trans>
                                    </div>
                                }
                                <hr/>
                                <Form
                                    onSubmit={this.handleFormSubmit}
                                    initialValues={dhcp.config}
                                    enabled={dhcp.config.enabled}
                                    interfaces={dhcp.interfaces}
                                    processing={dhcp.processingInterfaces}
                                />
                            </div>
                        </div>
                    </Card>
                }
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
