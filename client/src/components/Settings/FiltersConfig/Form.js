import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { renderCheckboxField, toNumber } from '../../../helpers/form';
import { FILTERS_INTERVALS_HOURS, FORM_NAME } from '../../../helpers/constants';

const getTitleForInterval = (interval, t) => {
    if (interval === 0) {
        return t('disabled');
    }
    if (interval === 72 || interval === 168) {
        return t('interval_days', { count: interval / 24 });
    }

    return t('interval_hours', { count: interval });
};

const getIntervalSelect = (processing, t, handleChange, toNumber) => (
    <Field
        name="interval"
        className="custom-select"
        component="select"
        onChange={handleChange}
        normalize={toNumber}
        disabled={processing}
    >
        {FILTERS_INTERVALS_HOURS.map((interval) => (
            <option value={interval} key={interval}>
                {getTitleForInterval(interval, t)}
            </option>
        ))}
    </Field>
);

const Form = (props) => {
    const {
        handleSubmit, handleChange, processing, t,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Field
                            name="enabled"
                            type="checkbox"
                            modifier="checkbox--settings"
                            component={renderCheckboxField}
                            placeholder={t('block_domain_use_filters_and_hosts')}
                            subtitle={t('filters_block_toggle_hint')}
                            onChange={handleChange}
                            disabled={processing}
                        />
                    </div>
                </div>
                <div className="col-12 col-md-5">
                    <div className="form__group form__group--inner mb-5">
                        <label className="form__label">
                            <Trans>filters_interval</Trans>
                        </label>
                        {getIntervalSelect(processing, t, handleChange, toNumber)}
                    </div>
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    handleChange: PropTypes.func,
    change: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default flow([
    withTranslation(),
    reduxForm({ form: FORM_NAME.FILTER_CONFIG }),
])(Form);
