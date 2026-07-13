import { createSignal, createEffect, type Accessor, Show } from 'solid-js';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { BLOCKING_MODES, UINT32_RANGE } from 'panel/helpers/constants';
import { getBlockingModeOptions } from '../../helpers';
import {
    validateRequiredValue,
    validateBetween,
    validateIpv4,
    validateIpv6,
} from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const BlockingModeDialog = (props: Props) => {
    const blockingModeOptions = getBlockingModeOptions();

    const [blockingMode, setBlockingMode] = createSignal(dnsConfigState.blocking_mode);
    createEffect(() => {
        if (props.open()) {
            setBlockingMode(dnsConfigState.blocking_mode);
        }
    });

    const blockingIpv4 = useField<string>(
        () => props.open(),
        () => dnsConfigState.blocking_ipv4,
        {
            validate: (v) => validateRequiredValue(v) || validateIpv4(v) || '',
        },
    );
    const blockingIpv6 = useField<string>(
        () => props.open(),
        () => dnsConfigState.blocking_ipv6,
        {
            validate: (v) => validateRequiredValue(v) || validateIpv6(v) || '',
        },
    );
    const ttl = useField<number>(
        () => props.open(),
        () => dnsConfigState.blocked_response_ttl,
        {
            validate: (v) => {
                if (Number.isNaN(v)) {
                    return intl.getMessage('form_error_required');
                }
                return validateBetween(v, UINT32_RANGE.MIN, UINT32_RANGE.MAX) || '';
            },
        },
    );

    const handleSubmit = () => {
        const isCustom = blockingMode() === BLOCKING_MODES.custom_ip;
        if (isCustom && (blockingIpv4.validate() || blockingIpv6.validate())) return;
        if (ttl.validate()) return;

        const payload: Record<string, unknown> = {
            blocking_mode: blockingMode(),
            blocked_response_ttl: ttl.value(),
        };
        if (blockingMode() === BLOCKING_MODES.custom_ip) {
            payload.blocking_ipv4 = blockingIpv4.value();
            payload.blocking_ipv6 = blockingIpv6.value();
        }
        setDnsConfig(payload);
        props.onClose();
    };

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_blocking_mode_title')}
            description={intl.getMessage('dns_blocking_mode_desc')}
            onClose={props.onClose}
            onSubmit={handleSubmit}
            processing={props.processing}
        >
            <Radio
                name="blocking_mode"
                options={blockingModeOptions}
                value={blockingMode()}
                handleChange={(v: string) => setBlockingMode(v)}
                inModal
            />
            <Show when={blockingMode() === BLOCKING_MODES.custom_ip}>
                <>
                    <div class={theme.form.input}>
                        <Input
                            id="blocking_ipv4"
                            label={
                                <>
                                    {intl.getMessage('dns_blocking_mode_ipv4_label')}
                                    <FaqTooltip
                                        text={intl.getMessage(
                                            'server_config_blocking_mode_ipv4_faq',
                                        )}
                                        menuSize="large"
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('dns_blocking_mode_ipv4_placeholder')}
                            value={blockingIpv4.value()}
                            onChange={(e) => blockingIpv4.setValue(e.target.value)}
                            onBlur={() => blockingIpv4.validate()}
                            errorMessage={blockingIpv4.error()}
                            disabled={blockingMode() !== BLOCKING_MODES.custom_ip}
                            size="large"
                        />
                    </div>
                    <div class={theme.form.input}>
                        <Input
                            id="blocking_ipv6"
                            label={
                                <>
                                    {intl.getMessage('dns_blocking_mode_ipv6_label')}
                                    <FaqTooltip
                                        text={intl.getMessage(
                                            'server_config_blocking_mode_ipv6_faq',
                                        )}
                                        menuSize="large"
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('dns_blocking_mode_ipv6_placeholder')}
                            value={blockingIpv6.value()}
                            onChange={(e) => blockingIpv6.setValue(e.target.value)}
                            onBlur={() => blockingIpv6.validate()}
                            errorMessage={blockingIpv6.error()}
                            disabled={blockingMode() !== BLOCKING_MODES.custom_ip}
                            size="large"
                        />
                    </div>
                </>
            </Show>
            <div class={theme.form.input}>
                <Input
                    type="number"
                    id="blocked_response_ttl"
                    label={
                        <>
                            {intl.getMessage('dns_blocking_mode_ttl_label')}
                            <FaqTooltip
                                text={intl.getMessage('server_config_blocking_mode_ttl_faq')}
                                menuSize="large"
                            />
                        </>
                    }
                    placeholder={intl.getMessage('dns_blocking_mode_ttl_placeholder')}
                    value={ttl.value()}
                    onChange={(e) =>
                        ttl.setValue(e.target.value === '' ? NaN : Number(e.target.value))
                    }
                    onBlur={() => ttl.validate()}
                    min={UINT32_RANGE.MIN}
                    max={UINT32_RANGE.MAX}
                    errorMessage={ttl.error()}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
