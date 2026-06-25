import { createSignal, createMemo, For, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import {
    validateBetween,
    validateIp,
    validateIpPerLine,
    validateIpv4,
    validateIpv6,
    validateMaxValue,
    validateRequiredValue,
} from 'panel/helpers/validators';
import {
    BLOCKING_MODES,
    IPV4_SUBNET_PREFIX,
    IPV6_SUBNET_PREFIX,
    RATE_LIMIT,
    UINT32_RANGE,
} from 'panel/helpers/constants';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Input } from 'panel/common/controls/Input';
import { Textarea } from 'panel/common/controls/Textarea';
import { Radio } from 'panel/common/controls/Radio';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import theme from 'panel/lib/theme';

import s from './Form.module.pcss';

const blockingModesDescriptions = [
    intl.getMessage('server_config_blocking_mode_default_desc'),
    intl.getMessage('server_config_blocking_mode_refused_desc'),
    intl.getMessage('server_config_blocking_mode_nxdomain_desc'),
    intl.getMessage('server_config_blocking_mode_null_ip_desc'),
    intl.getMessage('server_config_blocking_mode_custom_ip_desc'),
];

const checkboxes: {
    name: 'dnssec_enabled' | 'disable_ipv6';
    placeholder: string;
    subtitle: string;
}[] = [
    {
        name: 'dnssec_enabled',
        placeholder: intl.getMessage('server_config_dnssec_enable'),
        subtitle: intl.getMessage('server_config_dnssec_enable_desc'),
    },
    {
        name: 'disable_ipv6',
        placeholder: intl.getMessage('server_config_disable_ipv6'),
        subtitle: intl.getMessage('server_config_disable_ipv6_desc'),
    },
];

const customIps: {
    name: 'blocking_ipv4' | 'blocking_ipv6';
    label: string;
    placeholder: string;
    faq: string;
    validateIp: (value: string) => string;
}[] = [
    {
        name: 'blocking_ipv4',
        label: intl.getMessage('server_config_blocking_mode_ipv4'),
        placeholder: intl.getMessage('server_config_blocking_mode_ipv4_placeholder'),
        faq: intl.getMessage('server_config_blocking_mode_ipv4_faq'),
        validateIp: validateIpv4,
    },
    {
        name: 'blocking_ipv6',
        label: intl.getMessage('server_config_blocking_mode_ipv6'),
        placeholder: intl.getMessage('server_config_blocking_mode_ipv6_placeholder'),
        faq: intl.getMessage('server_config_blocking_mode_ipv6_faq'),
        validateIp: validateIpv6,
    },
];

const blockingModeOptions = [
    { value: BLOCKING_MODES.default, text: intl.getMessage('server_config_default') },
    { value: BLOCKING_MODES.refused, text: intl.getMessage('server_config_refused') },
    { value: BLOCKING_MODES.nxdomain, text: intl.getMessage('server_config_nxdomain') },
    { value: BLOCKING_MODES.null_ip, text: intl.getMessage('server_config_null_ip') },
    { value: BLOCKING_MODES.custom_ip, text: intl.getMessage('server_config_custom_ip') },
];

type FormData = {
    ratelimit: number;
    ratelimit_subnet_len_ipv4: number;
    ratelimit_subnet_len_ipv6: number;
    ratelimit_whitelist: string;
    edns_cs_enabled: boolean;
    edns_cs_use_custom: boolean;
    edns_cs_custom_ip?: string;
    dnssec_enabled: boolean;
    disable_ipv6: boolean;
    blocking_mode: string;
    blocking_ipv4?: string;
    blocking_ipv6?: string;
    blocked_response_ttl: number;
};

type Props = {
    processing?: boolean;
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

export const Form = (props: Props) => {
    const [ratelimit, setRatelimit] = createSignal(props.initialValues?.ratelimit ?? 0);
    const [ratelimitSubnetIpv4, setRatelimitSubnetIpv4] = createSignal(
        props.initialValues?.ratelimit_subnet_len_ipv4 ?? 0,
    );
    const [ratelimitSubnetIpv6, setRatelimitSubnetIpv6] = createSignal(
        props.initialValues?.ratelimit_subnet_len_ipv6 ?? 0,
    );
    const [ratelimitWhitelist, setRatelimitWhitelist] = createSignal(
        props.initialValues?.ratelimit_whitelist ?? '',
    );
    const [ednsCsEnabled, setEdnsCsEnabled] = createSignal(
        props.initialValues?.edns_cs_enabled ?? false,
    );
    const [ednsCsUseCustom, setEdnsCsUseCustom] = createSignal(
        props.initialValues?.edns_cs_use_custom ?? false,
    );
    const [ednsCsCustomIp, setEdnsCsCustomIp] = createSignal(
        props.initialValues?.edns_cs_custom_ip ?? '',
    );
    const [dnssecEnabled, setDnssecEnabled] = createSignal(
        props.initialValues?.dnssec_enabled ?? false,
    );
    const [disableIpv6, setDisableIpv6] = createSignal(props.initialValues?.disable_ipv6 ?? false);
    const [blockingMode, setBlockingMode] = createSignal(
        props.initialValues?.blocking_mode ?? BLOCKING_MODES.default,
    );
    const [blockingIpv4, setBlockingIpv4] = createSignal(props.initialValues?.blocking_ipv4 ?? '');
    const [blockingIpv6, setBlockingIpv6] = createSignal(props.initialValues?.blocking_ipv6 ?? '');
    const [blockedResponseTtl, setBlockedResponseTtl] = createSignal(
        props.initialValues?.blocked_response_ttl ?? 0,
    );

    // Error signals
    const [ratelimitError, setRatelimitError] = createSignal('');
    const [subnetIpv4Error, setSubnetIpv4Error] = createSignal('');
    const [subnetIpv6Error, setSubnetIpv6Error] = createSignal('');
    const [whitelistError, setWhitelistError] = createSignal('');
    const [customIpError, setCustomIpError] = createSignal('');
    const [blockingIpv4Error, setBlockingIpv4Error] = createSignal('');
    const [blockingIpv6Error, setBlockingIpv6Error] = createSignal('');
    const [ttlError, setTtlError] = createSignal('');

    const validateAll = () => {
        setRatelimitError(
            validateRequiredValue(String(ratelimit())) ||
                validateMaxValue(ratelimit(), RATE_LIMIT.MAX) ||
                '',
        );
        setSubnetIpv4Error(
            validateRequiredValue(String(ratelimitSubnetIpv4())) ||
                validateBetween(
                    ratelimitSubnetIpv4(),
                    IPV4_SUBNET_PREFIX.MIN,
                    IPV4_SUBNET_PREFIX.MAX,
                ) ||
                '',
        );
        setSubnetIpv6Error(
            validateRequiredValue(String(ratelimitSubnetIpv6())) ||
                validateBetween(
                    ratelimitSubnetIpv6(),
                    IPV6_SUBNET_PREFIX.MIN,
                    IPV6_SUBNET_PREFIX.MAX,
                ) ||
                '',
        );
        setWhitelistError(
            ratelimitWhitelist() ? validateIpPerLine(ratelimitWhitelist()) || '' : '',
        );
        setTtlError(validateRequiredValue(String(blockedResponseTtl())) || validateBetween(blockedResponseTtl(), UINT32_RANGE.MIN, UINT32_RANGE.MAX) || '');

        if (ednsCsUseCustom()) {
            const err =
                validateRequiredValue(ednsCsCustomIp()) || validateIp(ednsCsCustomIp()) || '';
            setCustomIpError(err);
        }

        if (blockingMode() === BLOCKING_MODES.custom_ip) {
            setBlockingIpv4Error(
                validateRequiredValue(blockingIpv4()) || validateIpv4(blockingIpv4()) || '',
            );
            setBlockingIpv6Error(
                validateRequiredValue(blockingIpv6()) || validateIpv6(blockingIpv6()) || '',
            );
        }
    };

    const hasErrors = createMemo(
        () =>
            ratelimitError() ||
            subnetIpv4Error() ||
            subnetIpv6Error() ||
            whitelistError() ||
            customIpError() ||
            blockingIpv4Error() ||
            blockingIpv6Error() ||
            ttlError(),
    );

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        validateAll();
        if (hasErrors()) return;

        props.onSubmit({
            ratelimit: ratelimit(),
            ratelimit_subnet_len_ipv4: ratelimitSubnetIpv4(),
            ratelimit_subnet_len_ipv6: ratelimitSubnetIpv6(),
            ratelimit_whitelist: ratelimitWhitelist(),
            edns_cs_enabled: ednsCsEnabled(),
            edns_cs_use_custom: ednsCsUseCustom(),
            edns_cs_custom_ip: ednsCsCustomIp(),
            dnssec_enabled: dnssecEnabled(),
            disable_ipv6: disableIpv6(),
            blocking_mode: blockingMode(),
            blocking_ipv4: blockingIpv4(),
            blocking_ipv6: blockingIpv6(),
            blocked_response_ttl: blockedResponseTtl(),
        });
    };

    return (
        <form onSubmit={handleSubmit} class={theme.form.form}>
            <div class={theme.form.group}>
                <div class={theme.form.input}>
                    <Input
                        value={String(ratelimit())}
                        onChange={(e: Event) =>
                            setRatelimit(Number((e.target as HTMLInputElement).value))
                        }
                        onBlur={() =>
                            setRatelimitError(
                                validateRequiredValue(String(ratelimit())) ||
                                    validateMaxValue(ratelimit(), RATE_LIMIT.MAX) ||
                                    '',
                            )
                        }
                        data-testid="dns_config_ratelimit"
                        type="number"
                        label={
                            <>
                                {intl.getMessage('server_config_rate_limit')}
                                <FaqTooltip
                                    text={intl.getMessage('server_config_rate_limit_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('server_config_rate_limit_placeholder')}
                        errorMessage={ratelimitError()}
                        min={UINT32_RANGE.MIN}
                        max={UINT32_RANGE.MAX}
                    />
                </div>

                <div class={theme.form.input}>
                    <Input
                        value={String(ratelimitSubnetIpv4())}
                        onChange={(e: Event) =>
                            setRatelimitSubnetIpv4(Number((e.target as HTMLInputElement).value))
                        }
                        onBlur={() =>
                            setSubnetIpv4Error(
                                validateRequiredValue(String(ratelimitSubnetIpv4())) ||
                                    validateBetween(
                                        ratelimitSubnetIpv4(),
                                        IPV4_SUBNET_PREFIX.MIN,
                                        IPV4_SUBNET_PREFIX.MAX,
                                    ) ||
                                    '',
                            )
                        }
                        data-testid="dns_config_subnet_ipv4"
                        type="number"
                        label={
                            <>
                                {intl.getMessage('server_config_subnet_len_ipv4')}
                                <FaqTooltip
                                    text={intl.getMessage('server_config_subnet_len_ipv4_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('server_config_subnet_len_placeholder')}
                        errorMessage={subnetIpv4Error()}
                        min={0}
                        max={32}
                    />
                </div>

                <div class={theme.form.input}>
                    <Input
                        value={String(ratelimitSubnetIpv6())}
                        onChange={(e: Event) =>
                            setRatelimitSubnetIpv6(Number((e.target as HTMLInputElement).value))
                        }
                        onBlur={() =>
                            setSubnetIpv6Error(
                                validateRequiredValue(String(ratelimitSubnetIpv6())) ||
                                    validateBetween(
                                        ratelimitSubnetIpv6(),
                                        IPV6_SUBNET_PREFIX.MIN,
                                        IPV6_SUBNET_PREFIX.MAX,
                                    ) ||
                                    '',
                            )
                        }
                        data-testid="dns_config_subnet_ipv6"
                        type="number"
                        label={
                            <>
                                {intl.getMessage('server_config_subnet_len_ipv6')}
                                <FaqTooltip
                                    text={intl.getMessage('server_config_subnet_len_ipv6_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('server_config_subnet_len_placeholder')}
                        errorMessage={subnetIpv6Error()}
                        min={0}
                        max={128}
                    />
                </div>

                <div class={theme.form.input}>
                    <Textarea
                        value={ratelimitWhitelist()}
                        onChange={(e: Event) =>
                            setRatelimitWhitelist((e.target as HTMLTextAreaElement).value)
                        }
                        onBlur={() =>
                            setWhitelistError(
                                ratelimitWhitelist()
                                    ? validateIpPerLine(ratelimitWhitelist()) || ''
                                    : '',
                            )
                        }
                        data-testid="dns_config_ratelimit_whitelist"
                        label={
                            <>
                                {intl.getMessage('server_config_rate_limit_whitelist')}
                                <FaqTooltip
                                    text={intl.getMessage('server_config_rate_limit_whitelist_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('ip_addresses_placeholder')}
                        errorMessage={whitelistError()}
                        size="medium"
                    />
                </div>

                <div class={theme.form.input}>
                    <Checkbox
                        name="edns_cs_enabled"
                        checked={ednsCsEnabled()}
                        onChange={(e: Event) =>
                            setEdnsCsEnabled((e.target as HTMLInputElement).checked)
                        }
                        data-testid="dns_config_edns_cs_enabled"
                        verticalAlign="start"
                    >
                        <div>
                            <div class={theme.text.t2}>
                                {intl.getMessage('server_config_edns_enable')}
                            </div>
                            <div class={theme.text.t4}>
                                {intl.getMessage('server_config_edns_cs_desc')}
                            </div>
                        </div>
                    </Checkbox>
                </div>

                <div class={theme.form.inner}>
                    <div class={theme.form.input}>
                        <Checkbox
                            name="edns_cs_use_custom"
                            checked={ednsCsUseCustom()}
                            onChange={(e: Event) =>
                                setEdnsCsUseCustom((e.target as HTMLInputElement).checked)
                            }
                            data-testid="dns_config_edns_use_custom_ip"
                            disabled={!ednsCsEnabled()}
                            verticalAlign="start"
                        >
                            <div>
                                <div class={theme.text.t2}>
                                    {intl.getMessage('server_config_edns_use_custom_ip')}
                                </div>
                                <div class={theme.text.t4}>
                                    {intl.getMessage('server_config_edns_use_custom_ip_desc')}
                                </div>
                            </div>
                        </Checkbox>
                    </div>

                    <Show when={ednsCsUseCustom()}>
                        <div class={theme.form.input}>
                            <Input
                                value={ednsCsCustomIp()}
                                onChange={(e: Event) =>
                                    setEdnsCsCustomIp((e.target as HTMLInputElement).value)
                                }
                                onBlur={() => {
                                    if (ednsCsUseCustom()) {
                                        setCustomIpError(
                                            validateRequiredValue(ednsCsCustomIp()) ||
                                                validateIp(ednsCsCustomIp()) ||
                                                '',
                                        );
                                    }
                                }}
                                data-testid="dns_config_edns_cs_custom_ip"
                                placeholder={intl.getMessage('enter_ip_address_placeholder')}
                                errorMessage={customIpError()}
                                disabled={!ednsCsEnabled()}
                            />
                        </div>
                    </Show>
                </div>

                <For each={checkboxes}>
                    {({ name, placeholder, subtitle }) => (
                        <div class={theme.form.input}>
                            <Checkbox
                                name={name}
                                checked={
                                    name === 'dnssec_enabled' ? dnssecEnabled() : disableIpv6()
                                }
                                onChange={(e: Event) => {
                                    const v = (e.target as HTMLInputElement).checked;
                                    if (name === 'dnssec_enabled') setDnssecEnabled(v);
                                    else setDisableIpv6(v);
                                }}
                                id={`dns_config_${name}`}
                                verticalAlign="start"
                            >
                                <div>
                                    <div class={theme.text.t2}>{placeholder}</div>
                                    <div class={theme.text.t4}>{subtitle}</div>
                                </div>
                            </Checkbox>
                        </div>
                    )}
                </For>

                <div class={theme.form.input}>
                    <div class={cn(s.subtitle, theme.title.h6)}>
                        {intl.getMessage('server_config_blocking_mode')}
                    </div>
                    <div class={s.descriptions}>
                        <For each={blockingModesDescriptions}>
                            {(description) => <div class={theme.text.t2}>{description}</div>}
                        </For>
                    </div>

                    <Radio
                        value={blockingMode()}
                        handleChange={(v: string) => setBlockingMode(v)}
                        name="blocking_mode"
                        options={blockingModeOptions}
                    />
                </div>

                <Show when={blockingMode() === BLOCKING_MODES.custom_ip}>
                    <For each={customIps}>
                        {({ label, name, placeholder, faq, validateIp: validateIpFn }) => (
                            <div class={theme.form.input}>
                                <Input
                                    value={
                                        name === 'blocking_ipv4' ? blockingIpv4() : blockingIpv6()
                                    }
                                    onChange={(e: Event) => {
                                        const v = (e.target as HTMLInputElement).value;
                                        if (name === 'blocking_ipv4') setBlockingIpv4(v);
                                        else setBlockingIpv6(v);
                                    }}
                                    onBlur={() => {
                                        const val =
                                            name === 'blocking_ipv4'
                                                ? blockingIpv4()
                                                : blockingIpv6();
                                        const err =
                                            validateRequiredValue(val) || validateIpFn(val) || '';
                                        if (name === 'blocking_ipv4') setBlockingIpv4Error(err);
                                        else setBlockingIpv6Error(err);
                                    }}
                                    data-testid={`dns_config_${name}`}
                                    type="text"
                                    label={
                                        <>
                                            {label}
                                            <FaqTooltip text={faq} menuSize="large" />
                                        </>
                                    }
                                    placeholder={placeholder}
                                    errorMessage={
                                        name === 'blocking_ipv4'
                                            ? blockingIpv4Error()
                                            : blockingIpv6Error()
                                    }
                                />
                            </div>
                        )}
                    </For>
                </Show>

                <div class={theme.form.input}>
                    <Input
                        value={String(blockedResponseTtl())}
                        onChange={(e: Event) =>
                            setBlockedResponseTtl(Number((e.target as HTMLInputElement).value))
                        }
                        onBlur={() =>
                            setTtlError(validateRequiredValue(String(blockedResponseTtl())) || '')
                        }
                        data-testid="dns_config_blocked_response_ttl"
                        type="number"
                        label={
                            <>
                                {intl.getMessage('server_config_blocking_mode_ttl')}
                                <FaqTooltip
                                    text={intl.getMessage('server_config_blocking_mode_ttl_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('form_enter_blocked_response_ttl')}
                        errorMessage={ttlError()}
                        min={UINT32_RANGE.MIN}
                        max={UINT32_RANGE.MAX}
                    />
                </div>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="dns_config_save"
                    variant="primary"
                    disabled={props.processing}
                    class={theme.form.button}
                    size="small"
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
