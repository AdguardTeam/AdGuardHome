import React from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';
import Form from './Form';
import { getObjDiff } from '../../../helpers/helpers';

const FiltersConfig = (props) => {
    const { initialValues, processing } = props;

    const handleFormChange = debounce((values) => {
        const diff = getObjDiff(initialValues, values);

        if (Object.values(diff).length > 0) {
            props.setFiltersConfig(values);
        }
    }, DEBOUNCE_TIMEOUT);

    return (
        <Form
            initialValues={initialValues}
            onSubmit={handleFormChange}
            onChange={handleFormChange}
            processing={processing}
        />
    );
};

FiltersConfig.propTypes = {
    initialValues: PropTypes.object.isRequired,
    processing: PropTypes.bool.isRequired,
    setFiltersConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(FiltersConfig);
