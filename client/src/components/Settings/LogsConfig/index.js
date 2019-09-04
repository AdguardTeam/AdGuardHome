import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';
import Card from '../../ui/Card';
import Form from './Form';

class LogsConfig extends Component {
    handleFormChange = debounce((values) => {
        this.props.setLogsConfig(values);
    }, DEBOUNCE_TIMEOUT);

    handleLogsClear = () => {
        const { t, clearLogs } = this.props;
        // eslint-disable-next-line no-alert
        if (window.confirm(t('query_log_confirm_clear'))) {
            clearLogs();
        }
    };

    render() {
        const {
            t, enabled, interval, processing, processingClear,
        } = this.props;

        return (
            <Card
                title={t('query_log_configuration')}
                bodyType="card-body box-body--settings"
                id="logs_config"
            >
                <div className="form">
                    <Form
                        initialValues={{
                            enabled,
                            interval,
                        }}
                        onSubmit={this.handleFormChange}
                        onChange={this.handleFormChange}
                        processing={processing}
                    />

                    <button
                        type="button"
                        className="btn btn-outline-secondary btn-sm"
                        onClick={this.handleLogsClear}
                        disabled={processingClear}
                    >
                        <Trans>query_log_clear</Trans>
                    </button>
                </div>
            </Card>
        );
    }
}

LogsConfig.propTypes = {
    interval: PropTypes.number.isRequired,
    enabled: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    processingClear: PropTypes.bool.isRequired,
    setLogsConfig: PropTypes.func.isRequired,
    clearLogs: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(LogsConfig);
