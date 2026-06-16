import { createSignal, createMemo, For, Show } from 'solid-js';

import intl from 'panel/common/intl';
import { dnsConfigState, clearDnsCache } from 'panel/stores/dnsConfig';
import { CACHE_CONFIG_FIELDS, UINT32_RANGE } from 'panel/helpers/constants';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import theme from 'panel/lib/theme';
import { formatNumber } from 'panel/helpers/helpers';

const CACHE_TTL_MAX_VALUE = 4_294_967_295;

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

export const Form = (props: CacheFormProps) => {
    const [cacheEnabled, setCacheEnabled] = createSignal(props.initialValues?.cache_enabled || false);
    const [cacheSize, setCacheSize] = createSignal(props.initialValues?.cache_size || 0);
    const [cacheTtlMin, setCacheTtlMin] = createSignal(props.initialValues?.cache_ttl_min || 0);
    const [cacheTtlMax, setCacheTtlMax] = createSignal(props.initialValues?.cache_ttl_max || 0);
    const [cacheOptimistic, setCacheOptimistic] = createSignal(props.initialValues?.cache_optimistic || false);
    const [openConfirmClear, setOpenConfirmClear] = createSignal(false);

    const minExceedsMax = createMemo(() =>
        cacheTtlMin() > 0 && cacheTtlMax() > 0 && cacheTtlMin() > cacheTtlMax(),
    );

    const cacheSizeZeroWhenEnabled = createMemo(() =>
        cacheEnabled() && cacheSize() === 0,
    );

    const getFieldValue = (name: string): number => {
        switch (name) {
            case CACHE_CONFIG_FIELDS.cache_size: return cacheSize();
            case CACHE_CONFIG_FIELDS.cache_ttl_min: return cacheTtlMin();
            case CACHE_CONFIG_FIELDS.cache_ttl_max: return cacheTtlMax();
            default: return 0;
        }
    };

    const setFieldValue = (name: string, value: number) => {
        switch (name) {
            case CACHE_CONFIG_FIELDS.cache_size: setCacheSize(value); break;
            case CACHE_CONFIG_FIELDS.cache_ttl_min: setCacheTtlMin(value); break;
            case CACHE_CONFIG_FIELDS.cache_ttl_max: setCacheTtlMax(value); break;
        }
    };

    const handleCacheInputsValueError = (name: string) => {
        const value = getFieldValue(name);
        switch (name) {
            case CACHE_CONFIG_FIELDS.cache_size: {
                if (cacheSizeZeroWhenEnabled()) {
                    return (
                        <div class={theme.form.error}>
                            {intl.getMessage('cache_config_size_validation')}
                        </div>
                    );
                }
                if (value > CACHE_TTL_MAX_VALUE || value < 0) {
                    return (
                        <div class={theme.form.error}>
                            {intl.getMessage('form_value_length_common_error', {
                                max_length: formatNumber(CACHE_TTL_MAX_VALUE),
                            })}
                        </div>
                    );
                }
                break;
            }
            case CACHE_CONFIG_FIELDS.cache_ttl_min: {
                if (value > CACHE_TTL_MAX_VALUE || value < 0) {
                    return (
                        <div class={theme.form.error}>
                            {intl.getMessage('form_value_length_common_error', {
                                max_length: formatNumber(CACHE_TTL_MAX_VALUE),
                            })}
                        </div>
                    );
                }
                break;
            }
            case CACHE_CONFIG_FIELDS.cache_ttl_max:
                if (value > CACHE_TTL_MAX_VALUE || value < 0) {
                    return (
                        <div class={theme.form.error}>
                            {intl.getMessage('form_value_length_common_error', {
                                max_length: formatNumber(CACHE_TTL_MAX_VALUE),
                            })}
                        </div>
                    );
                }
                break;
        }
        return null;
    };

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        props.onSubmit({
            cache_enabled: cacheEnabled(),
            cache_size: cacheSize(),
            cache_ttl_min: cacheTtlMin(),
            cache_ttl_max: cacheTtlMax(),
            cache_optimistic: cacheOptimistic(),
        });
    };

    return (
        <form onSubmit={handleSubmit} class={theme.form.form}>
            <div class={theme.form.group}>
                <div class={theme.form.input}>
                    <Checkbox
                        name="cache_enabled"
                        checked={cacheEnabled()}
                        onChange={(e: Event) => setCacheEnabled((e.target as HTMLInputElement).checked)}
                        data-testid="dns_cache_enabled"
                        verticalAlign="start"
                    >
                        <div>
                            <div class={theme.text.t2}>
                                {intl.getMessage('cache_enabled')}
                            </div>
                            <div class={theme.text.t4}>
                                {intl.getMessage('cache_enabled_desc')}
                            </div>
                        </div>
                    </Checkbox>
                </div>

                <For each={INPUTS_FIELDS}>
                    {({ name, title, faq, placeholder }) => (
                        <div class={theme.form.input}>
                            <Input
                                type="number"
                                value={String(getFieldValue(name))}
                                onChange={(e: Event) => {
                                    const v = (e.target as HTMLInputElement).value;
                                    setFieldValue(name, v === '' ? 0 : Number(v));
                                }}
                                id={name}
                                label={
                                    <>
                                        {title}
                                        <FaqTooltip text={faq} menuSize="large" />
                                    </>
                                }
                                placeholder={placeholder}
                                min={0}
                                max={UINT32_RANGE.MAX}
                            />
                            {handleCacheInputsValueError(name)}
                        </div>
                    )}
                </For>

                <Show when={minExceedsMax()}>
                    <div class={theme.form.error}>
                        {intl.getMessage('cache_config_ttl_validation')}
                    </div>
                </Show>

                <div class={theme.form.input}>
                    <Checkbox
                        name="cache_optimistic"
                        checked={cacheOptimistic()}
                        onChange={(e: Event) => setCacheOptimistic((e.target as HTMLInputElement).checked)}
                        data-testid="dns_cache_optimistic"
                        verticalAlign="start"
                    >
                        <div>
                            <div class={theme.text.t2}>
                                {intl.getMessage('cache_config_optimistic')}
                            </div>
                            <div class={theme.text.t4}>
                                {intl.getMessage('cache_config_optimistic_desc')}
                            </div>
                        </div>
                    </Checkbox>
                </div>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="dns_save"
                    variant="primary"
                    size="small"
                    disabled={
                        dnsConfigState.processingSetConfig ||
                        minExceedsMax() ||
                        cacheSizeZeroWhenEnabled()
                    }
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    id="dns_clear"
                    variant="secondary-danger"
                    size="small"
                    onClick={() => setOpenConfirmClear(true)}
                    class={theme.form.button}
                >
                    {intl.getMessage('cache_config_clear')}
                </Button>
            </div>

            <Show when={openConfirmClear()}>
                <ConfirmDialog
                    onClose={() => setOpenConfirmClear(false)}
                    onConfirm={() => {
                        clearDnsCache();
                        setOpenConfirmClear(false);
                    }}
                    buttonText={intl.getMessage('confirm')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('cache_confirm_clear_title')}
                    text={intl.getMessage('cache_confirm_clear_desc')}
                    buttonVariant="danger"
                />
            </Show>
        </form>
    );
};
