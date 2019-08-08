import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';
import Form from './Form';
import Card from '../../ui/Card';

class StatsConfig extends Component {
    handleFormChange = debounce((values) => {
        this.props.setStatsConfig(values);
    }, DEBOUNCE_TIMEOUT);

    render() {
        const {
            t,
            interval,
            processing,
        } = this.props;

        return (
            <Card
                title={t('stats_params')}
                bodyType="card-body box-body--settings"
            >
                <div className="form">
                    <Form
                        initialValues={{
                            interval,
                        }}
                        onSubmit={this.handleFormChange}
                        onChange={this.handleFormChange}
                        processing={processing}
                    />
                </div>
            </Card>
        );
    }
}

StatsConfig.propTypes = {
    interval: PropTypes.number.isRequired,
    processing: PropTypes.bool.isRequired,
    setStatsConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(StatsConfig);
