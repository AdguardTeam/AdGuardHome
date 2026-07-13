import { createSignal, createMemo, Show } from 'solid-js';
import cn from 'clsx';

import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { getPathWithQueryString } from '../../helpers/helpers';
import { CLIENT_ID_LINK, MOBILE_CONFIG_LINKS, STANDARD_HTTPS_PORT } from '../../helpers/constants';
import { toNumber } from '../../helpers/form';
import {
    validateConfigClientId,
    validateServerName,
    validatePort,
    validateIsSafePort,
} from '../../helpers/validators';
import { Button } from '../../common/ui/Button';

import s from './MobileConfigForm.module.pcss';

type FormValues = {
    host: string;
    clientId: string;
    protocol: string;
    port?: number;
};

type Props = {
    initialValues?: FormValues;
};

const defaultFormValues: FormValues = {
    host: '',
    clientId: '',
    protocol: MOBILE_CONFIG_LINKS.DOT,
    port: undefined,
};

export const MobileConfigForm = (props: Props) => {
    const defaults = { ...defaultFormValues, ...props.initialValues };

    const [host, setHost] = createSignal(defaults.host);
    const [clientId, setClientId] = createSignal(defaults.clientId);
    const [protocol, setProtocol] = createSignal(defaults.protocol);
    const [port, setPort] = createSignal<number | undefined>(defaults.port);

    const [hostError, setHostError] = createSignal<string | undefined>();
    const [clientIdError, setClientIdError] = createSignal<string | undefined>();
    const [portError, setPortError] = createSignal<string | undefined>();

    const isValid = createMemo(() => {
        return !hostError() && !clientIdError() && !portError() && !!host();
    });

    const handleHostChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        setHost(value);
        const error = validateServerName(value);
        setHostError(error || undefined);
    };

    const handleClientIdChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        setClientId(value);
        const error = validateConfigClientId(value);
        setClientIdError(error || undefined);
    };

    const handlePortChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        const numValue = toNumber(value);
        setPort(numValue || undefined);
        const rangeError = validatePort(numValue);
        const safetyError = validateIsSafePort(numValue);
        setPortError(rangeError || safetyError || undefined);
    };

    const getHostName = () => {
        if (port() && port() !== STANDARD_HTTPS_PORT && protocol() === MOBILE_CONFIG_LINKS.DOH) {
            return `${host()}:${port()}`;
        }
        return host();
    };

    const getDownloadLink = () => {
        if (!host() || !isValid()) {
            return (
                <Button class={s.configLink} variant="primary" disabled>
                    {intl.getMessage('download_mobileconfig')}
                </Button>
            );
        }

        const linkParams: { host: string; client_id?: string } = { host: getHostName() };
        if (clientId()) {
            linkParams.client_id = clientId();
        }

        const handleDownload = () => {
            const link = document.createElement('a');
            link.href = getPathWithQueryString(protocol(), linkParams);
            link.download = '';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        };

        return (
            <Button variant="primary" onClick={handleDownload}>
                {intl.getMessage('download_mobileconfig')}
            </Button>
        );
    };

    return (
        <form onSubmit={(e) => e.preventDefault()}>
            <div class={s.form}>
                <div class={cn(s.formGroup, s.formGroupSettings)}>
                    <Input
                        type="text"
                        data-testid="mobile_config_host"
                        label={intl.getMessage('dhcp_table_hostname')}
                        placeholder={intl.getMessage('form_enter_hostname')}
                        value={host()}
                        onChange={handleHostChange}
                        error={!!hostError()}
                        errorMessage={hostError()}
                    />
                </div>
                <Show when={protocol() === MOBILE_CONFIG_LINKS.DOH}>
                    <div class={cn(s.formGroup, s.formGroupSettings)}>
                        <Input
                            type="number"
                            data-testid="mobile_config_port"
                            label={intl.getMessage('encryption_https')}
                            placeholder={intl.getMessage('encryption_https')}
                            value={port()?.toString() ?? ''}
                            onChange={handlePortChange}
                            error={!!portError()}
                            errorMessage={portError()}
                        />
                    </div>
                </Show>

                <div class={cn(s.formGroup, s.formGroupSettings)}>
                    <label for="clientId" class={cn(s.formLabel, s.formLabelWithDesc)}>
                        {intl.getMessage('client_id')}
                        <FaqTooltip
                            text={intl.getMessage('client_id_faq', {
                                a: (text: string) => (
                                    <a
                                        href={CLIENT_ID_LINK}
                                        target="_blank"
                                        rel="noreferrer"
                                        class={s.dnsLink}
                                    >
                                        {text}
                                    </a>
                                ),
                            })}
                        />
                    </label>

                    <Input
                        type="text"
                        data-testid="mobile_config_client_id"
                        placeholder={intl.getMessage('client_id_placeholder')}
                        value={clientId()}
                        onChange={handleClientIdChange}
                        error={!!clientIdError()}
                        errorMessage={clientIdError()}
                    />
                </div>

                <div class={cn(s.formGroup, s.formGroupSettings)}>
                    <label class={s.formLabel}>{intl.getMessage('protocol')}</label>
                    <Select
                        options={[
                            {
                                value: MOBILE_CONFIG_LINKS.DOT,
                                label: intl.getMessage('dns_over_tls'),
                            },
                            {
                                value: MOBILE_CONFIG_LINKS.DOH,
                                label: intl.getMessage('dns_over_https'),
                            },
                        ]}
                        value={{
                            value: protocol(),
                            label:
                                protocol() === MOBILE_CONFIG_LINKS.DOT
                                    ? intl.getMessage('dns_over_tls')
                                    : intl.getMessage('dns_over_https'),
                        }}
                        onChange={(option) => setProtocol(option?.value)}
                        isSearchable={false}
                        size="responsive"
                        height="big"
                    />
                </div>
            </div>

            {getDownloadLink()}
        </form>
    );
};
