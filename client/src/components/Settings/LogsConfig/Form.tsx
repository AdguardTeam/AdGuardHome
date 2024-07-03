import React, { useEffect } from 'react';
import { change, Field, formValueSelector, reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import {
    CheckboxField,
    toFloatNumber,
    renderTextareaField,
    renderInputField,
    renderRadioField,
} from '../../../helpers/form';

import { trimLinesAndRemoveEmpty } from '../../../helpers/helpers';
import {
    FORM_NAME,
    QUERY_LOG_INTERVALS_DAYS,
    HOUR,
    DAY,
    RETENTION_CUSTOM,
    RETENTION_CUSTOM_INPUT,
    RETENTION_RANGE,
    CUSTOM_INTERVAL,
} from '../../../helpers/constants';
import '../FormButton.css';

const getIntervalTitle = (interval: any, t: any) => {
    switch (interval) {
        case RETENTION_CUSTOM:
            return t('settings_custom');
        case 6 * HOUR:
            return t('interval_6_hour');
        case DAY:
            return t('interval_24_hour');
        default:
            return t('interval_days', { count: interval / DAY });
    }
};

const getIntervalFields = (processing: any, t: any, toNumber: any) =>
    QUERY_LOG_INTERVALS_DAYS.map((interval) => (
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

interface FormProps {
    handleSubmit: (...args: unknown[]) => string;
    handleClear: (...args: unknown[]) => unknown;
    submitting: boolean;
    invalid: boolean;
    processing: boolean;
    processingClear: boolean;
    t: (...args: unknown[]) => string;
    interval?: number;
    customInterval?: number;
    dispatch: (...args: unknown[]) => unknown;
}

let Form = (props: FormProps) => {
    const {
        handleSubmit,
        submitting,
        invalid,
        processing,
        processingClear,
        handleClear,
        t,
        interval,
        customInterval,
        dispatch,
    } = props;

    useEffect(() => {
        if (QUERY_LOG_INTERVALS_DAYS.includes(interval)) {
            dispatch(change(FORM_NAME.LOG_CONFIG, CUSTOM_INTERVAL, null));
        }
    }, [interval]);

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
                    <Field
                        key={RETENTION_CUSTOM}
                        name="interval"
                        type="radio"
                        component={renderRadioField}
                        value={QUERY_LOG_INTERVALS_DAYS.includes(interval) ? RETENTION_CUSTOM : interval}
                        placeholder={getIntervalTitle(RETENTION_CUSTOM, t)}
                        normalize={toFloatNumber}
                        disabled={processing}
                    />
                    {!QUERY_LOG_INTERVALS_DAYS.includes(interval) && (
                        <div className="form__group--input">
                            <div className="form__desc form__desc--top">{t('custom_rotation_input')}</div>

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
                    {getIntervalFields(processing, t, toFloatNumber)}
                </div>
            </div>

            <label className="form__label form__label--with-desc">
                <Trans>ignore_domains_title</Trans>
            </label>

            <div className="form__desc form__desc--top">
                <Trans>ignore_domains_desc_query</Trans>
            </div>

            <div className="form__group form__group--settings">
                <Field
                    name="ignored"
                    type="textarea"
                    className="form-control form-control--textarea font-monospace text-input"
                    component={renderTextareaField}
                    placeholder={t('ignore_domains')}
                    disabled={processing}
                    normalizeOnBlur={trimLinesAndRemoveEmpty}
                />
            </div>

            <div className="mt-5">
                <button
                    type="submit"
                    className="btn btn-success btn-standard btn-large"
                    disabled={
                        submitting ||
                        invalid ||
                        processing ||
                        (!QUERY_LOG_INTERVALS_DAYS.includes(interval) && !customInterval)
                    }>
                    <Trans>save_btn</Trans>
                </button>

                <button
                    type="button"
                    className="btn btn-outline-secondary btn-standard form__button"
                    onClick={() => handleClear()}
                    disabled={processingClear}>
                    <Trans>query_log_clear</Trans>
                </button>
            </div>
        </form>
    );
};

const selector = formValueSelector(FORM_NAME.LOG_CONFIG);

Form = connect((state) => {
    const interval = selector(state, 'interval');
    const customInterval = selector(state, CUSTOM_INTERVAL);
    return {
        interval,
        customInterval,
    };
})(Form);

export default flow([withTranslation(), reduxForm({ form: FORM_NAME.LOG_CONFIG })])(Form);
