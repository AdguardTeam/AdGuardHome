import { Show, type JSX } from 'solid-js';
import cn from 'clsx';

import { Input } from 'panel/common/controls/Input';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    STANDARD_HTTPS_PORT,
} from 'panel/helpers/constants';
import type { PortField, ServerSettingsField, ServerSettingsValues } from '../validate';
import s from '../styles.module.pcss';

type Props = {
    values: ServerSettingsValues;
    onFieldChange: (field: ServerSettingsField, value: string) => void;
    onFieldBlur: (field: ServerSettingsField) => void;
    errorFor?: (field: ServerSettingsField) => string | undefined;
    warningFor?: (field: ServerSettingsField) => string | undefined;
    idPrefix?: string;
    testIdPrefix?: string;
    clearable?: boolean;
};

type PortInputProps = {
    id: string;
    name: PortField;
    label: JSX.Element;
    value?: number | string;
    onChange: (e: Event) => void;
    onBlur: () => void;
    errorMessage?: string;
    warning?: string;
    clearable?: boolean;
    testId?: string;
};

const FieldWarning = (props: { text?: string; testId?: string }) => (
    <Show when={props.text}>
        <div
            class={cn(theme.text.t3, theme.status.statusYellow, s.fieldWarning)}
            data-testid={props.testId}
        >
            {props.text}
        </div>
    </Show>
);

const PortInput = (props: PortInputProps) => (
    <div class={theme.form.input}>
        <Input
            id={props.id}
            name={props.name}
            type="number"
            value={props.value ?? ''}
            onChange={props.onChange}
            onBlur={props.onBlur}
            isClearable={props.clearable}
            label={props.label}
            errorMessage={props.errorMessage}
            size="large"
            data-testid={props.testId}
        />
        <FieldWarning
            text={props.warning}
            testId={props.testId ? `${props.testId}-warning` : undefined}
        />
    </div>
);

/**
 * Server name and the three encrypted DNS ports — the settings shared by the
 * TLS setup wizard's config step and the "Encrypted DNS server settings"
 * dialog.
 */
export const ServerSettingsFields = (props: Props) => {
    const id = (field: ServerSettingsField) => `${props.idPrefix ?? ''}${field}`;
    const error = (field: ServerSettingsField) => props.errorFor?.(field);
    const warning = (field: ServerSettingsField) => props.warningFor?.(field);
    const testId = (field: string) =>
        props.testIdPrefix ? `${props.testIdPrefix}-${field}` : undefined;

    return (
        <>
            <div class={theme.form.input}>
                <Input
                    id={id('server_name')}
                    name="server_name"
                    value={props.values.server_name ?? ''}
                    onChange={(e) =>
                        props.onFieldChange('server_name', (e.target as HTMLInputElement).value)
                    }
                    onBlur={() => props.onFieldBlur('server_name')}
                    isClearable={props.clearable}
                    label={
                        <>
                            {intl.getMessage('encryption_server')}
                            <FaqTooltip
                                menuSize="large"
                                text={
                                    <div class={s.tooltipText}>
                                        {intl.getMessage('encryption_server_tooltip')}
                                    </div>
                                }
                            />
                        </>
                    }
                    placeholder={intl.getMessage('encryption_server_enter')}
                    errorMessage={error('server_name')}
                    size="large"
                    data-testid={testId('server-name')}
                />
                <FieldWarning
                    text={warning('server_name')}
                    testId={
                        props.testIdPrefix ? `${props.testIdPrefix}-server-name-warning` : undefined
                    }
                />
            </div>

            <PortInput
                id={id('port_https')}
                name="port_https"
                label={
                    <>
                        {intl.getMessage('encryption_https')}
                        <FaqTooltip
                            menuSize="large"
                            text={intl.getMessage('encryption_https_tooltip', {
                                port: STANDARD_HTTPS_PORT,
                            })}
                        />
                    </>
                }
                value={props.values.port_https}
                onChange={(e) =>
                    props.onFieldChange('port_https', (e.target as HTMLInputElement).value)
                }
                onBlur={() => props.onFieldBlur('port_https')}
                errorMessage={error('port_https')}
                warning={warning('port_https')}
                clearable={props.clearable}
                testId={testId('port-https')}
            />

            <PortInput
                id={id('port_dns_over_tls')}
                name="port_dns_over_tls"
                label={
                    <>
                        {intl.getMessage('encryption_dot')}
                        <FaqTooltip
                            menuSize="large"
                            text={intl.getMessage('encryption_dot_tooltip', {
                                port: DNS_OVER_TLS_PORT,
                            })}
                        />
                    </>
                }
                value={props.values.port_dns_over_tls}
                onChange={(e) =>
                    props.onFieldChange('port_dns_over_tls', (e.target as HTMLInputElement).value)
                }
                onBlur={() => props.onFieldBlur('port_dns_over_tls')}
                errorMessage={error('port_dns_over_tls')}
                warning={warning('port_dns_over_tls')}
                clearable={props.clearable}
                testId={testId('port-dns-over-tls')}
            />

            <PortInput
                id={id('port_dns_over_quic')}
                name="port_dns_over_quic"
                label={
                    <>
                        {intl.getMessage('encryption_doq')}
                        <FaqTooltip
                            menuSize="large"
                            text={intl.getMessage('encryption_doq_tooltip', {
                                port: DNS_OVER_QUIC_PORT,
                            })}
                        />
                    </>
                }
                value={props.values.port_dns_over_quic}
                onChange={(e) =>
                    props.onFieldChange('port_dns_over_quic', (e.target as HTMLInputElement).value)
                }
                onBlur={() => props.onFieldBlur('port_dns_over_quic')}
                errorMessage={error('port_dns_over_quic')}
                warning={warning('port_dns_over_quic')}
                clearable={props.clearable}
                testId={testId('port-dns-over-quic')}
            />
        </>
    );
};
