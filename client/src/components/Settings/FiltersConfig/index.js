import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';
import Form from './Form';

class FiltersConfig extends Component {
    handleFormChange = debounce((values) => {
        this.props.setFiltersConfig(values);
    }, DEBOUNCE_TIMEOUT);

    render() {
        const { interval, enabled, processing } = this.props;

        return (
            <Form
                initialValues={{ interval, enabled }}
                onSubmit={this.handleFormChange}
                onChange={this.handleFormChange}
                processing={processing}
            />
        );
    }
}

FiltersConfig.propTypes = {
    interval: PropTypes.number.isRequired,
    enabled: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    setFiltersConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(FiltersConfig);
