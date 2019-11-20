import React, { Component } from 'react';
import PropTypes from 'prop-types';
import debounce from 'lodash/debounce';
import classnames from 'classnames';

import { DEBOUNCE_FILTER_TIMEOUT, RESPONSE_FILTER } from '../../../helpers/constants';
import { isValidQuestionType } from '../../../helpers/helpers';
import Form from './Form';
import Card from '../../ui/Card';

class Filters extends Component {
    getFilters = ({
        filter_domain, filter_question_type, filter_response_status, filter_client,
    }) => ({
        filter_domain: filter_domain || '',
        filter_question_type: isValidQuestionType(filter_question_type) ? filter_question_type.toUpperCase() : '',
        filter_response_status: filter_response_status === RESPONSE_FILTER.FILTERED ? filter_response_status : '',
        filter_client: filter_client || '',
    });

    handleFormChange = debounce((values) => {
        const filter = this.getFilters(values);
        this.props.setLogsFilter(filter);
    }, DEBOUNCE_FILTER_TIMEOUT);

    render() {
        const { filter, processingAdditionalLogs } = this.props;

        const cardBodyClass = classnames({
            'card-body': true,
            'card-body--loading': processingAdditionalLogs,
        });

        return (
            <Card bodyType={cardBodyClass}>
                <Form
                    initialValues={filter}
                    onChange={this.handleFormChange}
                />
            </Card>
        );
    }
}

Filters.propTypes = {
    filter: PropTypes.object.isRequired,
    setLogsFilter: PropTypes.func.isRequired,
    processingGetLogs: PropTypes.bool.isRequired,
    processingAdditionalLogs: PropTypes.bool.isRequired,
};

export default Filters;
