import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import { renderSelectField, renderRadioField, toNumber } from '../../../helpers/form';
import { QUERY_LOG_INTERVALS_DAYS } from '../../../helpers/constants';

const getIntervalFields = (processing, t, handleChange, toNumber) =>
    QUERY_LOG_INTERVALS_DAYS.map((interval) => {
        const title =
            interval === 1 ? t('interval_24_hour') : t('interval_days', { count: interval });

        return (
            <Field
                key={interval}
                name="interval"
                type="radio"
                component={renderRadioField}
                value={interval}
                placeholder={title}
                onChange={handleChange}
                normalize={toNumber}
                disabled={processing}
            />
        );
    });

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
                            component={renderSelectField}
                            placeholder={t('query_log_enable')}
                            onChange={handleChange}
                            disabled={processing}
                        />
                    </div>
                </div>
                <div className="col-12">
                    <label className="form__label">
                        <Trans>query_log_retention</Trans>
                    </label>
                </div>
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <div className="custom-controls-stacked">
                            {getIntervalFields(processing, t, handleChange, toNumber)}
                        </div>
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
    withNamespaces(),
    reduxForm({
        form: 'logConfigForm',
    }),
])(Form);
