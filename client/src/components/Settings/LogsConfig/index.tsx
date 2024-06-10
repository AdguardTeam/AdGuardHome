import React, { Component } from 'react';
import { withTranslation } from 'react-i18next';

import Card from '../../ui/Card';

import Form from './Form';
import { HOUR } from '../../../helpers/constants';

interface LogsConfigProps {
    interval: number;
    customInterval?: number;
    enabled: boolean;
    anonymize_client_ip: boolean;
    processing: boolean;
    ignored: unknown[];
    processingClear: boolean;
    setLogsConfig: (...args: unknown[]) => unknown;
    clearLogs: (...args: unknown[]) => unknown;
    t: (...args: unknown[]) => string;
}

class LogsConfig extends Component<LogsConfigProps> {
    handleFormSubmit = (values: any) => {
        const { t, interval: prevInterval } = this.props;
        const { interval, customInterval, ...rest } = values;

        const newInterval = customInterval ? customInterval * HOUR : interval;

        const data = {
            ...rest,
            ignored: values.ignored ? values.ignored.split('\n') : [],
            interval: newInterval,
        };

        if (newInterval < prevInterval) {
            // eslint-disable-next-line no-alert
            if (window.confirm(t('query_log_retention_confirm'))) {
                this.props.setLogsConfig(data);
            }
        } else {
            this.props.setLogsConfig(data);
        }
    };

    handleClear = () => {
        const { t, clearLogs } = this.props;
        // eslint-disable-next-line no-alert
        if (window.confirm(t('query_log_confirm_clear'))) {
            clearLogs();
        }
    };

    render() {
        const {
            t,

            enabled,

            interval,

            processing,

            processingClear,

            anonymize_client_ip,

            ignored,

            customInterval,
        } = this.props;

        return (
            <Card title={t('query_log_configuration')} bodyType="card-body box-body--settings" id="logs-config">
                <div className="form">
                    <Form
                        initialValues={{
                            enabled,
                            interval,
                            customInterval,
                            anonymize_client_ip,
                            ignored: ignored?.join('\n'),
                        }}
                        onSubmit={this.handleFormSubmit}
                        processing={processing}
                        processingClear={processingClear}
                        handleClear={this.handleClear}
                    />
                </div>
            </Card>
        );
    }
}

export default withTranslation()(LogsConfig);
