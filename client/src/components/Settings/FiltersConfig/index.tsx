import React from 'react';
import { withTranslation } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';

import Form from './Form';

import { getObjDiff } from '../../../helpers/helpers';

interface FiltersConfigProps {
    initialValues: object;
    processing: boolean;
    setFiltersConfig: (...args: unknown[]) => unknown;
    t: (...args: unknown[]) => string;
}

const FiltersConfig = (props: FiltersConfigProps) => {
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

export default withTranslation()(FiltersConfig);
