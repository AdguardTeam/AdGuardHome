import React from 'react';

import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import { renderInputField, toNumber, CheckboxField } from '../../../../helpers/form';
import { CACHE_CONFIG_FIELDS, FORM_NAME, UINT32_RANGE } from '../../../../helpers/constants';

import { replaceZeroWithEmptyString } from '../../../../helpers/helpers';
import { clearDnsCache } from '../../../../actions/dnsConfig';
import { RootState } from '../../../../initialState';

const INPUTS_FIELDS = [
    {
        name: CACHE_CONFIG_FIELDS.cache_size,
        title: 'cache_size',
        description: 'cache_size_desc',
        placeholder: 'enter_cache_size',
    },
    {
        name: CACHE_CONFIG_FIELDS.cache_ttl_min,
        title: 'cache_ttl_min_override',
        description: 'cache_ttl_min_override_desc',
        placeholder: 'enter_cache_ttl_min_override',
    },
    {
        name: CACHE_CONFIG_FIELDS.cache_ttl_max,
        title: 'cache_ttl_max_override',
        description: 'cache_ttl_max_override_desc',
        placeholder: 'enter_cache_ttl_max_override',
    },
];

interface CacheFormProps {
    handleSubmit: (...args: unknown[]) => string;
    submitting: boolean;
    invalid: boolean;
}

const Form = ({ handleSubmit, submitting, invalid }: CacheFormProps) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const { processingSetConfig } = useSelector((state: RootState) => state.dnsConfig, shallowEqual);
    const { cache_ttl_max, cache_ttl_min } = useSelector(
        (state: RootState) => state.form[FORM_NAME.CACHE].values,
        shallowEqual,
    );

    const minExceedsMax = cache_ttl_min > 0 && cache_ttl_max > 0 && cache_ttl_min > cache_ttl_max;

    const handleClearCache = () => {
        if (window.confirm(t('confirm_dns_cache_clear'))) {
            dispatch(clearDnsCache());
        }
    };

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                {INPUTS_FIELDS.map(({ name, title, description, placeholder }) => (
                    <div className="col-12" key={name}>
                        <div className="col-12 col-md-7 p-0">
                            <div className="form__group form__group--settings">
                                <label htmlFor={name} className="form__label form__label--with-desc">
                                    {t(title)}
                                </label>

                                <div className="form__desc form__desc--top">{t(description)}</div>

                                <Field
                                    name={name}
                                    type="number"
                                    component={renderInputField}
                                    placeholder={t(placeholder)}
                                    disabled={processingSetConfig}
                                    className="form-control"
                                    normalizeOnBlur={replaceZeroWithEmptyString}
                                    normalize={toNumber}
                                    min={0}
                                    max={UINT32_RANGE.MAX}
                                />
                            </div>
                        </div>
                    </div>
                ))}
                {minExceedsMax && <span className="text-danger pl-3 pb-3">{t('ttl_cache_validation')}</span>}
            </div>

            <div className="row">
                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <Field
                            name="cache_optimistic"
                            type="checkbox"
                            component={CheckboxField}
                            placeholder={t('cache_optimistic')}
                            disabled={processingSetConfig}
                            subtitle={t('cache_optimistic_desc')}
                        />
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standard btn-large"
                disabled={submitting || invalid || processingSetConfig || minExceedsMax}>
                <Trans>save_btn</Trans>
            </button>

            <button
                type="button"
                className="btn btn-outline-secondary btn-standard form__button"
                onClick={handleClearCache}>
                <Trans>clear_cache</Trans>
            </button>
        </form>
    );
};

export default reduxForm({ form: FORM_NAME.CACHE })(Form);
