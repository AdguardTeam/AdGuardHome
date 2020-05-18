import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import {
    renderInputField,
    renderRadioField,
    renderSelectField,
    required,
    ipv4,
    ipv6,
    biggerOrEqualZero,
    toNumber,
} from '../../../../helpers/form';
import { BLOCKING_MODES } from '../../../../helpers/constants';

const checkboxes = [{
    name: 'edns_cs_enabled',
    placeholder: 'edns_enable',
    subtitle: 'edns_cs_desc',
},
{
    name: 'dnssec_enabled',
    placeholder: 'dnssec_enable',
    subtitle: 'dnssec_enable_desc',
},
{
    name: 'disable_ipv6',
    placeholder: 'disable_ipv6',
    subtitle: 'disable_ipv6_desc',
}];

const customIps = [{
    description: 'blocking_ipv4_desc',
    name: 'blocking_ipv4',
    validateIp: ipv4,
},
{
    description: 'blocking_ipv6_desc',
    name: 'blocking_ipv6',
    validateIp: ipv6,
}];

const getFields = (processing, t) => Object.values(BLOCKING_MODES)
    .map(mode => (
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
}) =>
    <form onSubmit={handleSubmit}>
        <div className="row">
            <div className="col-12 col-sm-6">
                <div className="form__group form__group--settings">
                    <label htmlFor="ratelimit"
                           className="form__label form__label--with-desc">
                        <Trans>rate_limit</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        <Trans>rate_limit_desc</Trans>
                    </div>
                    <Field
                        name="ratelimit"
                        type="number"
                        component={renderInputField}
                        className="form-control"
                        placeholder={t('form_enter_rate_limit')}
                        normalize={toNumber}
                        validate={[required, biggerOrEqualZero]}
                    />
                </div>
            </div>
            {checkboxes.map(({ name, placeholder, subtitle }) =>
                <div className="col-12" key={name}>
                    <div className="form__group form__group--settings">
                        <Field
                            name={name}
                            type="checkbox"
                            component={renderSelectField}
                            placeholder={t(placeholder)}
                            disabled={processing}
                            subtitle={t(subtitle)}
                        />
                    </div>
                </div>)}
            <div className="col-12">
                <div className="form__group form__group--settings mb-4">
                    <label className="form__label form__label--with-desc">
                        <Trans>blocking_mode</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        {Object.values(BLOCKING_MODES)
                            .map(mode => (
                                <li key={mode}>
                                    <Trans>{`blocking_mode_${mode}`}</Trans>
                                </li>
                            ))}
                    </div>
                    <div className="custom-controls-stacked">
                        {getFields(processing, t)}
                    </div>
                </div>
            </div>
            {blockingMode === BLOCKING_MODES.custom_ip && (
                <Fragment>
                    {customIps.map(({
                        description,
                        name,
                        validateIp,
                    }) => <div className="col-12 col-sm-6" key={name}>
                        <div className="form__group form__group--settings">
                            <label className="form__label form__label--with-desc"
                                   htmlFor={name}><Trans>{name}</Trans>
                            </label>
                            <div className="form__desc form__desc--top">
                                <Trans>{description}</Trans>
                            </div>
                            <Field
                                name={name}
                                component={renderInputField}
                                className="form-control"
                                placeholder={t('form_enter_ip')}
                                validate={[validateIp, required]}
                            />
                        </div>
                    </div>)}
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
    </form>;

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
