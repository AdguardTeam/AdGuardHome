import React from 'react';

import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { CheckboxField, toNumber } from '../../../helpers/form';
import { FILTERS_INTERVALS_HOURS, FILTERS_RELATIVE_LINK, FORM_NAME } from '../../../helpers/constants';

const getTitleForInterval = (interval: any, t: any) => {
    if (interval === 0) {
        return t('disabled');
    }
    if (interval === 72 || interval === 168) {
        return t('interval_days', { count: interval / 24 });
    }

    return t('interval_hours', { count: interval });
};

const getIntervalSelect = (processing: any, t: any, handleChange: any, toNumber: any) => (
    <Field
        name="interval"
        className="custom-select"
        component="select"
        onChange={handleChange}
        normalize={toNumber}
        disabled={processing}>
        {FILTERS_INTERVALS_HOURS.map((interval) => (
            <option value={interval} key={interval}>
                {getTitleForInterval(interval, t)}
            </option>
        ))}
    </Field>
);

interface FormProps {
    handleSubmit: (...args: unknown[]) => string;
    handleChange?: (...args: unknown[]) => unknown;
    change: (...args: unknown[]) => unknown;
    submitting: boolean;
    invalid: boolean;
    processing: boolean;
    t: (...args: unknown[]) => string;
}

const Form = (props: FormProps) => {
    const { handleSubmit, handleChange, processing, t } = props;

    const components = {
        a: <a href={FILTERS_RELATIVE_LINK} rel="noopener noreferrer" />,
    };

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Field
                            name="enabled"
                            type="checkbox"
                            modifier="checkbox--settings"
                            component={CheckboxField}
                            placeholder={t('block_domain_use_filters_and_hosts')}
                            subtitle={<Trans components={components}>filters_block_toggle_hint</Trans>}
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

export default flow([withTranslation(), reduxForm({ form: FORM_NAME.FILTER_CONFIG })])(Form);
