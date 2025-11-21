import React, { Component } from 'react';
import { withTranslation } from 'react-i18next';

import Card from '@/components/ui/Card';

import { HOUR } from '@/helpers/constants';
import { Form, FormValues } from './Form';

interface StatsConfigProps {
    interval: number;
    customInterval?: number;
    ignored: unknown[];
    enabled: boolean;
    processing: boolean;
    processingReset: boolean;
    setStatsConfig: (...args: unknown[]) => unknown;
    resetStats: (...args: unknown[]) => unknown;
    t: (...args: unknown[]) => string;
}

class StatsConfig extends Component<StatsConfigProps> {
    handleFormSubmit = ({ enabled, interval, ignored, customInterval }: FormValues) => {
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
            <Card title={t('statistics_configuration')} bodyType="card-body box-body--settings" id="stats-config">
                <div className="form">
                    <Form
                        initialValues={{
                            interval,
                            customInterval,
                            enabled,
                            ignored: ignored.join('\n'),
                        }}
                        processing={processing}
                        processingReset={processingReset}
                        onSubmit={this.handleFormSubmit}
                        onReset={this.handleReset}
                    />
                </div>
            </Card>
        );
    }
}

export default withTranslation()(StatsConfig);
