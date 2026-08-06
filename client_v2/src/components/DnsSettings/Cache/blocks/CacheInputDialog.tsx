import { untrack, type Accessor } from 'solid-js';
import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { UINT32_RANGE } from 'panel/helpers/constants';
import { validateBetween, validateRequiredValue } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type CacheConfigKey = 'cache_size' | 'cache_ttl_min' | 'cache_ttl_max';

type Props = {
    configKey: CacheConfigKey;
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

const CACHE_DIALOG_CONFIG: Record<
    CacheConfigKey,
    {
        title: () => string;
        description: () => string;
        label: () => string;
        placeholder?: () => string;
        validate: (v: string) => string;
    }
> = {
    cache_size: {
        title: () => intl.getMessage('dns_cache_size_title'),
        description: () => intl.getMessage('dns_cache_size_desc'),
        label: () => intl.getMessage('dns_cache_size_label'),
        validate: (v) =>
            validateRequiredValue(v) ||
            validateBetween(Number(v), UINT32_RANGE.MIN, UINT32_RANGE.MAX) ||
            '',
    },
    cache_ttl_min: {
        title: () => intl.getMessage('dns_override_min_ttl_title'),
        description: () => intl.getMessage('dns_override_min_ttl_desc'),
        label: () => intl.getMessage('dns_override_min_ttl_label'),
        placeholder: () => intl.getMessage('dns_override_min_ttl_placeholder'),
        validate: (v) =>
            validateRequiredValue(v) ||
            validateBetween(Number(v), UINT32_RANGE.MIN, UINT32_RANGE.MAX) ||
            '',
    },
    cache_ttl_max: {
        title: () => intl.getMessage('dns_override_max_ttl_title'),
        description: () => intl.getMessage('dns_override_max_ttl_desc'),
        label: () => intl.getMessage('dns_override_max_ttl_label'),
        placeholder: () => intl.getMessage('dns_override_max_ttl_placeholder'),
        validate: (v) => {
            const requiredErr = validateRequiredValue(v);
            if (requiredErr) return requiredErr;
            const num = Number(v);
            const rangeErr = validateBetween(num, UINT32_RANGE.MIN, UINT32_RANGE.MAX);
            if (rangeErr) return rangeErr;
            // Cross-field TTL check: min > max
            const minVal = Number(dnsConfigState.cache_ttl_min);
            if (minVal > 0 && num > 0 && minVal > num) {
                return intl.getMessage('cache_config_ttl_validation');
            }
            return '';
        },
    },
};

export const CacheInputDialog = (props: Props) => {
    const config = () => CACHE_DIALOG_CONFIG[props.configKey];

    const field = useField<string>(
        () => props.open(),
        () => String(dnsConfigState[props.configKey] ?? 0),
        { validate: config().validate },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={config().title()}
            description={config().description()}
            onClose={props.onClose}
            onSubmit={() => {
                const configKey = props.configKey;
                field.submitIfValid((v) => {
                    setDnsConfig({ [configKey]: v === '' ? 0 : Number(v) });
                    untrack(() => props.onClose());
                });
            }}
            processing={props.processing}
        >
            <div class={theme.form.input}>
                <Input
                    type="number"
                    value={field.value()}
                    onChange={(e: Event) => field.setValue((e.target as HTMLInputElement).value)}
                    onBlur={() => field.validate()}
                    id={props.configKey}
                    label={config().label()}
                    placeholder={config().placeholder?.()}
                    min={UINT32_RANGE.MIN}
                    max={UINT32_RANGE.MAX}
                    errorMessage={field.error()}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
