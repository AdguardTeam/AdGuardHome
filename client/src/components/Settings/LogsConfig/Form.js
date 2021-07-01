import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { CheckboxField, renderRadioField, toFloatNumber } from '../../../helpers/form';
import { FORM_NAME, QUERY_LOG_INTERVALS_DAYS } from '../../../helpers/constants';
import '../FormButton.css';

const getIntervalTitle = (interval, t) => {
    switch (interval) {
        case 0.25:
            return t('interval_6_hour');
        case 1:
            return t('interval_24_hour');
        default:
            return t('interval_days', { count: interval });
    }
};

const getIntervalFields = (processing, t, toNumber) => QUERY_LOG_INTERVALS_DAYS.map((interval) => (
    <Field
        key={interval}
        name="interval"
        type="radio"
        component={renderRadioField}
        value={interval}
        placeholder={getIntervalTitle(interval, t)}
        normalize={toNumber}
        disabled={processing}
    />
));

const Form = (props) => {
    const {
        handleSubmit, submitting, invalid, processing, processingClear, handleClear, t,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="form__group form__group--settings">
                <Field
                    name="enabled"
                    type="checkbox"
                    component={CheckboxField}
                    placeholder={t('query_log_enable')}
                    disabled={processing}
                />
            </div>
            <div className="form__group form__group--settings">
                <Field
                    name="anonymize_client_ip"
                    type="checkbox"
                    component={CheckboxField}
                    placeholder={t('anonymize_client_ip')}
                    subtitle={t('anonymize_client_ip_desc')}
                    disabled={processing}
                />
            </div>
            <label className="form__label">
                <Trans>query_log_retention</Trans>
            </label>
            <div className="form__group form__group--settings">
                <div className="custom-controls-stacked">
                    {getIntervalFields(processing, t, toFloatNumber)}
                </div>
            </div>
            <div className="mt-5">
                <button
                    type="submit"
                    className="btn btn-success btn-standard btn-large"
                    disabled={submitting || invalid || processing}
                >
                    <Trans>save_btn</Trans>
                </button>
                <button
                    type="button"
                    className="btn btn-outline-secondary btn-standard form__button"
                    onClick={() => handleClear()}
                    disabled={processingClear}
                >
                    <Trans>query_log_clear</Trans>
                </button>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    handleClear: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    processingClear: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default flow([
    withTranslation(),
    reduxForm({ form: FORM_NAME.LOG_CONFIG }),
])(Form);
