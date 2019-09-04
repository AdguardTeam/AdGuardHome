import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';
import Card from '../../ui/Card';
import Form from './Form';

class StatsConfig extends Component {
    handleFormChange = debounce((values) => {
        this.props.setStatsConfig(values);
    }, DEBOUNCE_TIMEOUT);

    handleReset = () => {
        const { t, resetStats } = this.props;
        // eslint-disable-next-line no-alert
        if (window.confirm(t('statistics_clear_confirm'))) {
            resetStats();
        }
    };

    render() {
        const {
            t, interval, processing, processingReset,
        } = this.props;

        return (
            <Card title={t('statistics_configuration')} bodyType="card-body box-body--settings">
                <div className="form">
                    <Form
                        initialValues={{
                            interval,
                        }}
                        onSubmit={this.handleFormChange}
                        onChange={this.handleFormChange}
                        processing={processing}
                    />

                    <button
                        type="button"
                        className="btn btn-outline-secondary btn-sm"
                        onClick={this.handleReset}
                        disabled={processingReset}
                    >
                        <Trans>statistics_clear</Trans>
                    </button>
                </div>
            </Card>
        );
    }
}

StatsConfig.propTypes = {
    interval: PropTypes.number.isRequired,
    processing: PropTypes.bool.isRequired,
    processingReset: PropTypes.bool.isRequired,
    setStatsConfig: PropTypes.func.isRequired,
    resetStats: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(StatsConfig);
