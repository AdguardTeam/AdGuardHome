import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import { renderField, renderRadioField, required, ipv4, ipv6, isPositive, toNumber } from '../../../helpers/form';
import { BLOCKING_MODES } from '../../../helpers/constants';

const getFields = (processing, t) => Object.values(BLOCKING_MODES).map(mode => (
    <Field
        key={mode}
        name="blocking_mode"
        type="radio"
        component={renderRadioField}
        value={mode}
        placeholder={t(mode)}
        disabled={processing}
    />
));

let Form = ({
    handleSubmit, submitting, invalid, processing, blockingMode, t,
}) => (
    <form onSubmit={handleSubmit}>
        <div className="row">
            <div className="col-12 col-sm-6">
                <div className="form__group form__group--settings">
                    <label htmlFor="ratelimit">
                        <Trans>rate_limit</Trans>
                        </label>
                    <Field
                        name="ratelimit"
                        type="number"
                        component={renderField}
                        className="form-control"
                        placeholder={t('form_enter_rate_limit')}
                        normalize={toNumber}
                        validate={[required, isPositive]}
                    />
                </div>
            </div>
            <div className="col-12">
                <div className="form__group form__group--settings mb-3">
                    <label className="form__label">
                        <Trans>blocking_mode</Trans>
                    </label>
                    <div className="custom-controls-stacked">
                        {getFields(processing, t)}
                    </div>
                </div>
            </div>
            {blockingMode === BLOCKING_MODES.custom_ip && (
                <Fragment>
                    <div className="col-12 col-sm-6">
                        <div className="form__group form__group--settings">
                            <label htmlFor="blocking_ipv4">
                                <Trans>blocking_ipv4</Trans>
                            </label>
                            <Field
                                name="blocking_ipv4"
                                component={renderField}
                                className="form-control"
                                placeholder={t('form_enter_ip')}
                                validate={[ipv4, required]}
                            />
                        </div>
                    </div>
                    <div className="col-12 col-sm-6">
                        <div className="form__group form__group--settings">
                            <label htmlFor="ip_address">
                                <Trans>blocking_ipv6</Trans>
                            </label>
                            <Field
                                name="blocking_ipv6"
                                component={renderField}
                                className="form-control"
                                placeholder={t('form_enter_ip')}
                                validate={[ipv6, required]}
                            />
                        </div>
                    </div>
                </Fragment>
            )}
        </div>
        <button
            type="submit"
            className="btn btn-success btn-standard btn-large"
            disabled={submitting || invalid || processing}
        >
            <Trans>save_btn</Trans>
        </button>
    </form>
);

Form.propTypes = {
    blockingMode: PropTypes.string.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

const selector = formValueSelector('blockingModeForm');

Form = connect((state) => {
    const blockingMode = selector(state, 'blocking_mode');
    return {
        blockingMode,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'blockingModeForm',
    }),
])(Form);
