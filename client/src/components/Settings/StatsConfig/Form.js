import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import {
    change, Field, formValueSelector, reduxForm,
} from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import { connect } from 'react-redux';

import {
    renderRadioField,
    toNumber,
    CheckboxField,
    renderTextareaField,
    toFloatNumber,
    renderInputField,
} from '../../../helpers/form';
import {
    FORM_NAME,
    STATS_INTERVALS_DAYS,
    DAY,
    RETENTION_CUSTOM,
    RETENTION_CUSTOM_INPUT,
    CUSTOM_INTERVAL,
    RETENTION_RANGE,
} from '../../../helpers/constants';
import '../FormButton.css';

const getIntervalTitle = (intervalMs, t) => {
    switch (intervalMs) {
        case RETENTION_CUSTOM:
            return t('settings_custom');
        case DAY:
            return t('interval_24_hour');
        default:
            return t('interval_days', { count: intervalMs / DAY });
    }
};

let Form = (props) => {
    const {
        handleSubmit,
        processing,
        submitting,
        invalid,
        handleReset,
        processingReset,
        t,
        interval,
        customInterval,
        dispatch,
    } = props;

    useEffect(() => {
        if (STATS_INTERVALS_DAYS.includes(interval)) {
            dispatch(change(FORM_NAME.STATS_CONFIG, CUSTOM_INTERVAL, null));
        }
    }, [interval]);

    return (
        <form onSubmit={handleSubmit}>
            <div className="form__group form__group--settings">
                <Field
                    name="enabled"
                    type="checkbox"
                    component={CheckboxField}
                    placeholder={t('statistics_enable')}
                    disabled={processing}
                />
            </div>
            <label className="form__label form__label--with-desc">
                <Trans>statistics_retention</Trans>
            </label>
            <div className="form__desc form__desc--top">
                <Trans>statistics_retention_desc</Trans>
            </div>
            <div className="form__group form__group--settings mt-2">
                <div className="custom-controls-stacked">
                    <Field
                        key={RETENTION_CUSTOM}
                        name="interval"
                        type="radio"
                        component={renderRadioField}
                        value={STATS_INTERVALS_DAYS.includes(interval)
                            ? RETENTION_CUSTOM
                            : interval
                        }
                        placeholder={getIntervalTitle(RETENTION_CUSTOM, t)}
                        normalize={toFloatNumber}
                        disabled={processing}
                    />
                    {!STATS_INTERVALS_DAYS.includes(interval) && (
                        <div className="form__group--input">
                            <div className="form__desc form__desc--top">
                                {t('custom_retention_input')}
                            </div>
                            <Field
                                key={RETENTION_CUSTOM_INPUT}
                                name={CUSTOM_INTERVAL}
                                type="number"
                                className="form-control"
                                component={renderInputField}
                                disabled={processing}
                                normalize={toFloatNumber}
                                min={RETENTION_RANGE.MIN}
                                max={RETENTION_RANGE.MAX}
                            />
                        </div>
                    )}
                    {STATS_INTERVALS_DAYS.map((interval) => (
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
                    ))}
                </div>
            </div>
            <label className="form__label form__label--with-desc">
                <Trans>ignore_domains_title</Trans>
            </label>
            <div className="form__desc form__desc--top">
                <Trans>ignore_domains_desc_stats</Trans>
            </div>
            <div className="form__group form__group--settings">
                <Field
                    name="ignored"
                    type="textarea"
                    className="form-control form-control--textarea font-monospace text-input"
                    component={renderTextareaField}
                    placeholder={t('ignore_domains')}
                    disabled={processing}
                />
            </div>
            <div className="mt-5">
                <button
                    type="submit"
                    className="btn btn-success btn-standard btn-large"
                    disabled={
                        submitting
                        || invalid
                        || processing
                        || (!STATS_INTERVALS_DAYS.includes(interval) && !customInterval)
                    }
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
    interval: PropTypes.number,
    customInterval: PropTypes.number,
    dispatch: PropTypes.func.isRequired,
};

const selector = formValueSelector(FORM_NAME.STATS_CONFIG);

Form = connect((state) => {
    const interval = selector(state, 'interval');
    const customInterval = selector(state, CUSTOM_INTERVAL);
    return {
        interval,
        customInterval,
    };
})(Form);

export default flow([
    withTranslation(),
    reduxForm({ form: FORM_NAME.STATS_CONFIG }),
])(Form);
