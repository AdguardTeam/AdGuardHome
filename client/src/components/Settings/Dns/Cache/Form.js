import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import { shallowEqual, useSelector } from 'react-redux';
import {
    biggerOrEqualZero,
    maxValue,
    renderInputField,
    required,
    toNumber,
} from '../../../../helpers/form';
import { FORM_NAME } from '../../../../helpers/constants';

const maxValue3600 = maxValue(3600);

const getInputFields = ({ required, maxValue3600 }) => [{
    name: 'cache_size',
    title: 'cache_size',
    description: 'cache_size_desc',
    placeholder: 'enter_cache_size',
    validate: required,
},
{
    name: 'cache_ttl_min',
    title: 'cache_ttl_min_override',
    description: 'cache_ttl_min_override_desc',
    placeholder: 'enter_cache_ttl_min_override',
    max: 3600,
    validate: maxValue3600,
},
{
    name: 'cache_ttl_max',
    title: 'cache_ttl_max_override',
    description: 'cache_ttl_max_override_desc',
    placeholder: 'enter_cache_ttl_max_override',
}];

const Form = ({
    handleSubmit, submitting, invalid,
}) => {
    const { t } = useTranslation();

    const { processingSetConfig } = useSelector((state) => state.dnsConfig, shallowEqual);
    const {
        cache_ttl_max, cache_ttl_min,
    } = useSelector((state) => state.form[FORM_NAME.CACHE].values, shallowEqual);

    const minExceedsMax = cache_ttl_min > cache_ttl_max;

    const INPUTS_FIELDS = getInputFields({
        required,
        maxValue3600,
    });

    return <form onSubmit={handleSubmit}>
        <div className="row">
            {INPUTS_FIELDS.map(({
                name, title, description, placeholder, validate, max,
            }) => <div className="col-12" key={name}>
                <div className="col-7 p-0">
                    <div className="form__group form__group--settings">
                        <label htmlFor={name}
                               className="form__label form__label--with-desc">{t(title)}</label>
                        <div className="form__desc form__desc--top">{t(description)}</div>
                        <Field
                            name={name}
                            type="number"
                            component={renderInputField}
                            placeholder={t(placeholder)}
                            disabled={processingSetConfig}
                            normalize={toNumber}
                            className="form-control"
                            validate={[biggerOrEqualZero].concat(validate || [])}
                            min={0}
                            max={max}
                        />
                    </div>
                </div>
            </div>)}
            {minExceedsMax
            && <span className="text-danger pl-3 pb-3">{t('min_exceeds_max_value')}</span>}
        </div>
        <button
            type="submit"
            className="btn btn-success btn-standard btn-large"
            disabled={submitting || invalid || processingSetConfig || minExceedsMax}
        >
            <Trans>save_btn</Trans>
        </button>
    </form>;
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
};

export default reduxForm({ form: FORM_NAME.CACHE })(Form);
