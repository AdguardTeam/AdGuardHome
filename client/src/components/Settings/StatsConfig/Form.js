import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { renderRadioField, toNumber } from '../../../helpers/form';
import { FORM_NAME, STATS_INTERVALS_DAYS } from '../../../helpers/constants';
import '../FormButton.css';

const getIntervalFields = (processing, t, toNumber) => STATS_INTERVALS_DAYS.map((interval) => {
    const title = interval === 1 ? t('interval_24_hour') : t('interval_days', { count: interval });

    return (
        <Field
            key={interval}
            name="interval"
            type="radio"
            component={renderRadioField}
            value={interval}
            placeholder={title}
            normalize={toNumber}
            disabled={processing}
        />
    );
});

const Form = (props) => {
    const {
        handleSubmit, processing, submitting, invalid, handleReset, processingReset, t,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <label className="form__label form__label--with-desc">
                <Trans>statistics_retention</Trans>
            </label>
            <div className="form__desc form__desc--top">
                <Trans>statistics_retention_desc</Trans>
            </div>
            <div className="form__group form__group--settings mt-2">
                <div className="custom-controls-stacked">
                    {getIntervalFields(processing, t, toNumber)}
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
                    onClick={() => handleReset()}
                    disabled={processingReset}
                >
                    <Trans>statistics_clear</Trans>
                </button>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    handleReset: PropTypes.func.isRequired,
    change: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    processingReset: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default flow([
    withTranslation(),
    reduxForm({ form: FORM_NAME.STATS_CONFIG }),
])(Form);
