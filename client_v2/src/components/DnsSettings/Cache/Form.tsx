import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useDispatch, useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import { clearDnsCache } from 'panel/actions/dnsConfig';
import { CACHE_CONFIG_FIELDS, UINT32_RANGE } from 'panel/helpers/constants';
import { RootState } from 'panel/initialState';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import theme from 'panel/lib/theme';

const INPUTS_FIELDS = [
    {
        name: CACHE_CONFIG_FIELDS.cache_size,
        title: intl.getMessage('cache_config_size'),
        faq: intl.getMessage('cache_config_size_faq'),
        placeholder: intl.getMessage('enter_cache_size'),
    },
    {
        name: CACHE_CONFIG_FIELDS.cache_ttl_min,
        title: intl.getMessage('cache_config_min_ttl'),
        faq: intl.getMessage('cache_config_min_ttl_faq'),
        placeholder: intl.getMessage('cache_config_min_ttl_placeholder'),
    },
    {
        name: CACHE_CONFIG_FIELDS.cache_ttl_max,
        title: intl.getMessage('cache_config_max_ttl'),
        faq: intl.getMessage('cache_config_max_ttl_faq'),
        placeholder: intl.getMessage('cache_config_max_ttl_placeholder'),
    },
];

type FormData = {
    cache_enabled: boolean;
    cache_size: number;
    cache_ttl_min: number;
    cache_ttl_max: number;
    cache_optimistic: boolean;
};

type CacheFormProps = {
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

export const Form = ({ initialValues, onSubmit }: CacheFormProps) => {
    const dispatch = useDispatch();

    const { processingSetConfig } = useSelector((state: RootState) => state.dnsConfig);

    const {
        handleSubmit,
        watch,
        control,
        formState: { isSubmitting },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            cache_enabled: initialValues?.cache_enabled || false,
            cache_size: initialValues?.cache_size || 0,
            cache_ttl_min: initialValues?.cache_ttl_min || 0,
            cache_ttl_max: initialValues?.cache_ttl_max || 0,
            cache_optimistic: initialValues?.cache_optimistic || false,
        },
    });

    const cache_enabled = watch('cache_enabled');
    const cache_size = watch('cache_size');
    const cache_ttl_min = watch('cache_ttl_min');
    const cache_ttl_max = watch('cache_ttl_max');

    const minExceedsMax = cache_ttl_min > 0 && cache_ttl_max > 0 && cache_ttl_min > cache_ttl_max;
    const cacheSizeZeroWhenEnabled = cache_enabled && cache_size === 0;

    const handleClearCache = () => {
        if (window.confirm(intl.getMessage('confirm_dns_cache_clear'))) {
            dispatch(clearDnsCache());
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className={theme.form.form}>
            <div className={theme.form.group}>
                <div className={theme.form.input}>
                    <Controller
                        name="cache_enabled"
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                name={field.name}
                                checked={field.value}
                                onChange={field.onChange}
                                onBlur={field.onBlur}
                                data-testid="dns_cache_enabled"
                                disabled={processingSetConfig}
                                verticalAlign="start">
                                <div>
                                    <div className={theme.text.t2}>{intl.getMessage('cache_enabled')}</div>
                                    <div className={theme.text.t4}>{intl.getMessage('cache_enabled_desc')}</div>
                                </div>
                            </Checkbox>
                        )}
                    />
                </div>

                {INPUTS_FIELDS.map(({ name, title, faq, placeholder }) => (
                    <div key={name} className={theme.form.input}>
                        <Controller
                            name={name as keyof FormData}
                            control={control}
                            render={({ field }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    id={name}
                                    label={
                                        <>
                                            {title}
                                            <FaqTooltip text={faq} menuSize="large" />
                                        </>
                                    }
                                    placeholder={placeholder}
                                    disabled={processingSetConfig}
                                    min={0}
                                    max={UINT32_RANGE.MAX}
                                    onChange={(e) => {
                                        const value = e.target.value === '' ? 0 : Number(e.target.value);
                                        field.onChange(value);
                                    }}
                                    value={field.value === 0 ? '' : String(field.value)}
                                />
                            )}
                        />

                        {name === CACHE_CONFIG_FIELDS.cache_size && cacheSizeZeroWhenEnabled && (
                            <span className={theme.form.error}>{intl.getMessage('cache_size_validation')}</span>
                        )}
                    </div>
                ))}

                {minExceedsMax && <div className={theme.form.error}>{intl.getMessage('ttl_cache_validation')}</div>}

                <div className={theme.form.input}>
                    <Controller
                        name="cache_optimistic"
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                name={field.name}
                                checked={field.value}
                                onChange={field.onChange}
                                onBlur={field.onBlur}
                                data-testid="dns_cache_optimistic"
                                disabled={processingSetConfig}
                                verticalAlign="start">
                                <div>
                                    <div className={theme.text.t2}>{intl.getMessage('cache_config_optimistic')}</div>
                                    <div className={theme.text.t4}>
                                        {intl.getMessage('cache_config_optimistic_desc')}
                                    </div>
                                </div>
                            </Checkbox>
                        )}
                    />
                </div>
            </div>

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="dns_save"
                    variant="primary"
                    size="small"
                    disabled={isSubmitting || processingSetConfig || minExceedsMax || cacheSizeZeroWhenEnabled}
                    className={theme.form.button}>
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    id="dns_clear"
                    variant="secondary-danger"
                    size="small"
                    onClick={handleClearCache}
                    className={theme.form.button}>
                    {intl.getMessage('cache_config_clear')}
                </Button>
            </div>
        </form>
    );
};
