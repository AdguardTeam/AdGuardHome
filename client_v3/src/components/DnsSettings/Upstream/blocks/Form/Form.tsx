import { createSignal, createMemo } from 'solid-js';
import cn from 'clsx';

import { settingsState, testUpstreamWithFormValues } from 'panel/stores/settings';
import { dnsConfigState } from 'panel/stores/dnsConfig';
import { Textarea } from 'panel/common/controls/Textarea';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import intl from 'panel/common/intl';
import { DNS_REQUEST_OPTIONS, UINT32_RANGE } from 'panel/helpers/constants';
import { validateUpstreams } from 'panel/helpers/validators';
import theme from 'panel/lib/theme';

import { Examples } from '../Examples';

import s from './Form.module.pcss';

type FormData = {
    upstream_dns: string;
    upstream_mode: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    use_private_ptr_resolvers: boolean;
    resolve_clients: boolean;
    upstream_timeout: number;
};

type FormProps = {
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

export const Form = (props: FormProps) => {
    const [upstreamDns, setUpstreamDns] = createSignal(props.initialValues?.upstream_dns || '');
    const [upstreamMode, setUpstreamMode] = createSignal(
        props.initialValues?.upstream_mode || DNS_REQUEST_OPTIONS.LOAD_BALANCING,
    );
    const [fallbackDns, setFallbackDns] = createSignal(props.initialValues?.fallback_dns || '');
    const [bootstrapDns, setBootstrapDns] = createSignal(props.initialValues?.bootstrap_dns || '');
    const [localPtrUpstreams, setLocalPtrUpstreams] = createSignal(
        props.initialValues?.local_ptr_upstreams || '',
    );
    const [usePrivatePtrResolvers, setUsePrivatePtrResolvers] = createSignal(
        props.initialValues?.use_private_ptr_resolvers || false,
    );
    const [resolveClients, setResolveClients] = createSignal(
        props.initialValues?.resolve_clients || false,
    );
    const [upstreamTimeout, setUpstreamTimeout] = createSignal(
        props.initialValues?.upstream_timeout || 0,
    );

    const [upstreamDnsError, setUpstreamDnsError] = createSignal('');
    const [fallbackDnsError, setFallbackDnsError] = createSignal('');
    const [bootstrapDnsError, setBootstrapDnsError] = createSignal('');
    const [localPtrError, setLocalPtrError] = createSignal('');

    const upstreamModeOptions = createMemo(() => [
        {
            text: intl.getMessage('upstream_dns_load_balancing'),
            value: DNS_REQUEST_OPTIONS.LOAD_BALANCING,
            description: intl.getMessage('upstream_dns_load_balancing_desc'),
        },
        {
            text: intl.getMessage('upstream_dns_parallel_requests'),
            value: DNS_REQUEST_OPTIONS.PARALLEL,
            description: intl.getMessage('upstream_dns_parallel_requests_desc'),
        },
        {
            text: intl.getMessage('upstream_dns_fastest_addr'),
            value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
            description: (
                <>
                    {intl.getMessage('upstream_dns_fastest_addr_desc')}
                    <div class={cn(theme.text.t2, s.warning)}>
                        {intl.getMessage('upstream_dns_fastest_addr_warning')}
                    </div>
                </>
            ),
        },
    ]);

    const handleUpstreamTest = () => {
        testUpstreamWithFormValues(
            {
                bootstrap_dns: bootstrapDns(),
                upstream_dns: upstreamDns(),
                local_ptr_upstreams: localPtrUpstreams(),
                fallback_dns: fallbackDns(),
            },
            dnsConfigState.upstream_dns_file,
        );
    };

    const isSavingDisabled = createMemo(
        () => settingsState.processingTestUpstream || dnsConfigState.processingSetConfig,
    );

    const isTestDisabled = createMemo(() => !upstreamDns() || settingsState.processingTestUpstream);

    const validateUpstreamDns = () => {
        const err = upstreamDns() ? validateUpstreams(upstreamDns()) : undefined;
        setUpstreamDnsError(err || '');
    };

    const validateFallbackDns = () => {
        const err = fallbackDns() ? validateUpstreams(fallbackDns()) : undefined;
        setFallbackDnsError(err || '');
    };

    const validateBootstrapDns = () => {
        const err = bootstrapDns() ? validateUpstreams(bootstrapDns()) : undefined;
        setBootstrapDnsError(err || '');
    };

    const validateLocalPtr = () => {
        const err = localPtrUpstreams() ? validateUpstreams(localPtrUpstreams()) : undefined;
        setLocalPtrError(err || '');
    };

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        validateUpstreamDns();
        validateFallbackDns();
        validateBootstrapDns();
        validateLocalPtr();

        if (upstreamDnsError() || fallbackDnsError() || bootstrapDnsError() || localPtrError()) {
            return;
        }

        props.onSubmit({
            upstream_dns: upstreamDns(),
            upstream_mode: upstreamMode(),
            fallback_dns: fallbackDns(),
            bootstrap_dns: bootstrapDns(),
            local_ptr_upstreams: localPtrUpstreams(),
            use_private_ptr_resolvers: usePrivatePtrResolvers(),
            resolve_clients: resolveClients(),
            upstream_timeout: upstreamTimeout(),
        });
    };

    return (
        <form onSubmit={handleSubmit}>
            <div class={theme.form.group}>
                <div class={theme.form.input}>
                    <Textarea
                        value={upstreamDns()}
                        onChange={(e: Event) =>
                            setUpstreamDns((e.target as HTMLTextAreaElement).value)
                        }
                        onBlur={validateUpstreamDns}
                        id="upstream_dns"
                        label={
                            <>
                                {intl.getMessage('upstream_dns_addresses')}
                                <FaqTooltip
                                    text={intl.getMessage('upstream_dns_addresses_faq', {
                                        a: (text: string) => (
                                            <a
                                                href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams"
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                class={theme.link.link}
                                            >
                                                {text}
                                            </a>
                                        ),
                                        b: (text: string) => (
                                            <a
                                                href="https://link.adtidy.org/forward.html?action=dns_kb_providers&from=ui&app=home"
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                class={theme.link.link}
                                            >
                                                {text}
                                            </a>
                                        ),
                                    })}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('upstream_dns_placeholder')}
                        disabled={
                            !!dnsConfigState.upstream_dns_file ||
                            settingsState.processingTestUpstream
                        }
                        size="medium"
                        errorMessage={upstreamDnsError()}
                    />
                </div>

                <div class={theme.form.input}>
                    <Radio
                        name="upstream_mode"
                        value={upstreamMode()}
                        handleChange={(v: string) => setUpstreamMode(v)}
                        options={upstreamModeOptions()}
                        disabled={settingsState.processingTestUpstream}
                        verticalAlign="start"
                        textClass={s.radioText}
                    />
                </div>
            </div>

            <Examples />

            <div class={theme.form.group}>
                <div class={theme.form.input}>
                    <Textarea
                        value={fallbackDns()}
                        onChange={(e: Event) =>
                            setFallbackDns((e.target as HTMLTextAreaElement).value)
                        }
                        onBlur={validateFallbackDns}
                        id="fallback_dns"
                        label={
                            <>
                                {intl.getMessage('upstream_fallback_title')}
                                <FaqTooltip
                                    text={intl.getMessage('upstream_fallback_title_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('ip_addresses_placeholder')}
                        size="medium"
                        errorMessage={fallbackDnsError()}
                    />
                </div>

                <div class={theme.form.input}>
                    <Textarea
                        value={bootstrapDns()}
                        onChange={(e: Event) =>
                            setBootstrapDns((e.target as HTMLTextAreaElement).value)
                        }
                        onBlur={validateBootstrapDns}
                        id="bootstrap_dns"
                        data-testid="bootstrap_dns"
                        label={
                            <>
                                {intl.getMessage('upstream_bootstrap_dns_title')}
                                <FaqTooltip
                                    text={intl.getMessage('upstream_bootstrap_dns_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('ip_addresses_placeholder')}
                        size="medium"
                        errorMessage={bootstrapDnsError()}
                    />
                </div>

                <div class={theme.form.input}>
                    <Textarea
                        value={localPtrUpstreams()}
                        onChange={(e: Event) =>
                            setLocalPtrUpstreams((e.target as HTMLTextAreaElement).value)
                        }
                        onBlur={validateLocalPtr}
                        id="local_ptr_upstreams"
                        data-testid="local_ptr_upstreams"
                        label={
                            <>
                                {intl.getMessage('upstream_ptr')}
                                <FaqTooltip
                                    text={
                                        <>
                                            <div>
                                                {intl.getMessage('upstream_ptr_faq_1', {
                                                    value: '192.168.1.1/24',
                                                })}
                                            </div>
                                            <div>{intl.getMessage('upstream_ptr_faq_2')}</div>
                                            {dnsConfigState.default_local_ptr_upstreams?.length >
                                                0 && (
                                                <div>
                                                    {intl.getMessage('upstream_ptr_faq_3', {
                                                        value_1:
                                                            dnsConfigState
                                                                .default_local_ptr_upstreams[0],
                                                        value_2:
                                                            dnsConfigState
                                                                .default_local_ptr_upstreams[1],
                                                    })}
                                                </div>
                                            )}
                                        </>
                                    }
                                    menuSize="large"
                                    spacing
                                />
                            </>
                        }
                        placeholder={intl.getMessage('ip_addresses_placeholder')}
                        size="medium"
                        errorMessage={localPtrError()}
                    />
                </div>

                <div class={theme.form.input}>
                    <Checkbox
                        id="dns_use_private_ptr_resolvers"
                        name="use_private_ptr_resolvers"
                        checked={usePrivatePtrResolvers()}
                        onChange={(e: Event) =>
                            setUsePrivatePtrResolvers((e.target as HTMLInputElement).checked)
                        }
                        verticalAlign="start"
                    >
                        <div>
                            <div class={theme.text.t2}>
                                {intl.getMessage('upstream_private_ptr_resolvers_title')}
                            </div>
                            <div class={theme.text.t4}>
                                {intl.getMessage('upstream_private_ptr_resolvers_desc')}
                            </div>
                        </div>
                    </Checkbox>
                </div>

                <div class={theme.form.input}>
                    <Checkbox
                        id="dns_resolve_clients"
                        name="resolve_clients"
                        checked={resolveClients()}
                        onChange={(e: Event) =>
                            setResolveClients((e.target as HTMLInputElement).checked)
                        }
                        verticalAlign="start"
                    >
                        <div>
                            <div class={theme.text.t2}>
                                {intl.getMessage('upstream_enable_reverse_lookup_title')}
                            </div>
                            <div class={theme.text.t4}>
                                {intl.getMessage('upstream_enable_reverse_lookup_desc')}
                            </div>
                        </div>
                    </Checkbox>
                </div>

                <div class={theme.form.input}>
                    <Input
                        value={String(upstreamTimeout())}
                        onChange={(e: Event) =>
                            setUpstreamTimeout(Number((e.target as HTMLInputElement).value))
                        }
                        type="number"
                        id="upstream_timeout"
                        label={
                            <>
                                {intl.getMessage('upstream_timeout')}
                                <FaqTooltip
                                    text={intl.getMessage('upstream_timeout_faq')}
                                    menuSize="large"
                                />
                            </>
                        }
                        placeholder={intl.getMessage('upstream_timeout_placeholder')}
                        min={1}
                        max={UINT32_RANGE.MAX}
                    />
                </div>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    id="dns_upstream_save"
                    disabled={isSavingDisabled()}
                    class={theme.form.button}
                >
                    {intl.getMessage('apply')}
                </Button>

                <Button
                    type="button"
                    variant="secondary"
                    size="small"
                    id="dns_upstream_test"
                    onClick={handleUpstreamTest}
                    disabled={isTestDisabled()}
                    class={theme.form.button}
                >
                    {intl.getMessage('test_upstreams')}
                </Button>
            </div>
        </form>
    );
};
