import React, { Component } from 'react';
import PropTypes from 'prop-types';
import debounce from 'lodash/debounce';

import { DEBOUNCE_FILTER_TIMEOUT, RESPONSE_FILTER } from '../../../helpers/constants';
import { isValidQuestionType } from '../../../helpers/helpers';
import Form from './Form';

class Filters extends Component {
    getFilters = (filtered) => {
        const {
            domain, client, type, response,
        } = filtered;

        return {
            filter_domain: domain || '',
            filter_client: client || '',
            filter_question_type: isValidQuestionType(type) ? type.toUpperCase() : '',
            filter_response_status: response === RESPONSE_FILTER.FILTERED ? response : '',
        };
    };

    handleFormChange = debounce((values) => {
        const filter = this.getFilters(values);
        this.props.setLogsFilter(filter);
    }, DEBOUNCE_FILTER_TIMEOUT);

    render() {
        const { filter } = this.props;

        return (
            <Form
                initialValues={filter}
                onChange={this.handleFormChange}
            />
        );
    }
}

Filters.propTypes = {
    filter: PropTypes.object.isRequired,
    setLogsFilter: PropTypes.func.isRequired,
};

export default Filters;
