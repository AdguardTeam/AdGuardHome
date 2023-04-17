import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';

import Card from '../../ui/Card';
import Form from './Form';
import { HOUR } from '../../../helpers/constants';

class StatsConfig extends Component {
    handleFormSubmit = ({
        enabled, interval, ignored, customInterval,
    }) => {
        const { t, interval: prevInterval } = this.props;
        const newInterval = customInterval ? customInterval * HOUR : interval;

        const config = {
            enabled,
            interval: newInterval,
            ignored: ignored ? ignored.split('\n') : [],
        };

        if (config.interval < prevInterval) {
            if (window.confirm(t('statistics_retention_confirm'))) {
                this.props.setStatsConfig(config);
            }
        } else {
            this.props.setStatsConfig(config);
        }
    };

    handleReset = () => {
        const { t, resetStats } = this.props;
        // eslint-disable-next-line no-alert
        if (window.confirm(t('statistics_clear_confirm'))) {
            resetStats();
        }
    };

    render() {
        const {
            t,
            interval,
            customInterval,
            processing,
            processingReset,
            ignored,
            enabled,
        } = this.props;

        return (
            <Card
                title={t('statistics_configuration')}
                bodyType="card-body box-body--settings"
                id="stats-config"
            >
                <div className="form">
                    <Form
                        initialValues={{
                            interval,
                            customInterval,
                            enabled,
                            ignored: ignored.join('\n'),
                        }}
                        onSubmit={this.handleFormSubmit}
                        processing={processing}
                        processingReset={processingReset}
                        handleReset={this.handleReset}
                    />
                </div>
            </Card>
        );
    }
}

StatsConfig.propTypes = {
    interval: PropTypes.number.isRequired,
    customInterval: PropTypes.number,
    ignored: PropTypes.array.isRequired,
    enabled: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    processingReset: PropTypes.bool.isRequired,
    setStatsConfig: PropTypes.func.isRequired,
    resetStats: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(StatsConfig);
