import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import { shallowEqual, useSelector } from 'react-redux';
import { renderInputField, toNumber } from '../../../../helpers/form';
import { validateBiggerOrEqualZeroValue, getMaxValueValidator, validateRequiredValue } from '../../../../helpers/validators';
import { FORM_NAME, SECONDS_IN_HOUR } from '../../../../helpers/constants';

const validateMaxValue3600 = getMaxValueValidator(SECONDS_IN_HOUR);

const getInputFields = ({ validateRequiredValue, validateMaxValue3600 }) => [{
    name: 'cache_size',
    title: 'cache_size',
    description: 'cache_size_desc',
    placeholder: 'enter_cache_size',
    validate: validateRequiredValue,
},
{
    name: 'cache_ttl_min',
    title: 'cache_ttl_min_override',
    description: 'cache_ttl_min_override_desc',
    placeholder: 'enter_cache_ttl_min_override',
    max: SECONDS_IN_HOUR,
    validate: validateMaxValue3600,
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
        validateRequiredValue,
        validateMaxValue3600,
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
                            validate={[validateBiggerOrEqualZeroValue].concat(validate || [])}
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
