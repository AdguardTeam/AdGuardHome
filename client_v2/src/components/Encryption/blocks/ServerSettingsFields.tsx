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
    /** Current settings, e.g. a `createStore` object. */
    values: ServerSettingsValues;
    /** Raw input value — the host decides how to store it (e.g. `toNumber`). */
    onFieldChange: (field: ServerSettingsField, value: string) => void;
    onFieldBlur: (field: ServerSettingsField) => void;
    /** Error to render under a field, composed by the host. */
    errorFor?: (field: ServerSettingsField) => string | undefined;
    /** Non-blocking warning to render under a field, composed by the host. */
    warningFor?: (field: ServerSettingsField) => string | undefined;
    /** DOM id prefix — the wizard prefixes its ids with `tls_setup_`. */
    idPrefix?: string;
    clearablePorts?: boolean;
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
};

/**
 * Non-blocking note under a field, e.g. a server name missing from the
 * certificate.  Mirrors the wizard's yellow step message so both render the
 * same way.
 */
const FieldWarning = (props: { text?: string }) => (
    <Show when={props.text}>
        <div class={cn(theme.text.t3, theme.status.statusYellow, s.fieldWarning)}>{props.text}</div>
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
        />
        <FieldWarning text={props.warning} />
    </div>
);

/**
 * Server name and the three encrypted DNS ports — the settings shared by the
 * TLS setup wizard's config step and the "Encrypted DNS server settings"
 * dialog.  Presentational: the host owns the values, their validation and the
 * error precedence.
 */
export const ServerSettingsFields = (props: Props) => {
    const id = (field: ServerSettingsField) => `${props.idPrefix ?? ''}${field}`;
    const error = (field: ServerSettingsField) => props.errorFor?.(field);
    const warning = (field: ServerSettingsField) => props.warningFor?.(field);

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
                />
                <FieldWarning text={warning('server_name')} />
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
                clearable={props.clearablePorts}
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
                clearable={props.clearablePorts}
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
                clearable={props.clearablePorts}
            />
        </>
    );
};
