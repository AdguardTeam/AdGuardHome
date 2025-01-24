import React from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';

import { clearDnsCache } from '../../../../actions/dnsConfig';
import { CACHE_CONFIG_FIELDS, UINT32_RANGE } from '../../../../helpers/constants';
import { replaceZeroWithEmptyString } from '../../../../helpers/helpers';
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

type FormData = {
    cache_size: number;
    cache_ttl_min: number;
    cache_ttl_max: number;
    cache_optimistic: boolean;
};

type CacheFormProps = {
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

const Form = ({ initialValues, onSubmit }: CacheFormProps) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const { processingSetConfig } = useSelector((state: RootState) => state.dnsConfig);

    const {
        register,
        handleSubmit,
        watch,
        formState: { isSubmitting, isDirty },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            cache_size: initialValues?.cache_size || 0,
            cache_ttl_min: initialValues?.cache_ttl_min || 0,
            cache_ttl_max: initialValues?.cache_ttl_max || 0,
            cache_optimistic: initialValues?.cache_optimistic || false,
        },
    });

    const cache_ttl_min = watch('cache_ttl_min');
    const cache_ttl_max = watch('cache_ttl_max');

    const minExceedsMax = cache_ttl_min > 0 && cache_ttl_max > 0 && cache_ttl_min > cache_ttl_max;

    const handleClearCache = () => {
        if (window.confirm(t('confirm_dns_cache_clear'))) {
            dispatch(clearDnsCache());
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="row">
                {INPUTS_FIELDS.map(({ name, title, description, placeholder }) => (
                    <div className="col-12" key={name}>
                        <div className="col-12 col-md-7 p-0">
                            <div className="form__group form__group--settings">
                                <label htmlFor={name} className="form__label form__label--with-desc">
                                    {t(title)}
                                </label>

                                <div className="form__desc form__desc--top">{t(description)}</div>

                                <input
                                    type="number"
                                    className="form-control"
                                    placeholder={t(placeholder)}
                                    disabled={processingSetConfig}
                                    min={0}
                                    max={UINT32_RANGE.MAX}
                                    {...register(name as keyof FormData, {
                                        valueAsNumber: true,
                                        setValueAs: (value) => replaceZeroWithEmptyString(value),
                                    })}
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
                        <label className="checkbox">
                            <input
                                type="checkbox"
                                className="checkbox__input"
                                disabled={processingSetConfig}
                                {...register('cache_optimistic')}
                            />
                            <span className="checkbox__label">
                                <span className="checkbox__label-text checkbox__label-text--long">
                                    <span className="checkbox__label-title">{t('cache_optimistic')}</span>
                                    <span className="checkbox__label-subtitle">{t('cache_optimistic_desc')}</span>
                                </span>
                            </span>
                        </label>
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standard btn-large"
                disabled={isSubmitting || !isDirty || processingSetConfig || minExceedsMax}>
                {t('save_btn')}
            </button>

            <button
                type="button"
                className="btn btn-outline-secondary btn-standard form__button"
                onClick={handleClearCache}>
                {t('clear_cache')}
            </button>
        </form>
    );
};

export default Form;
