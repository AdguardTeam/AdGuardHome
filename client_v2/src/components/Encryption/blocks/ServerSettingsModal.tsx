import { createEffect, createSignal, on } from 'solid-js';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import intl from 'panel/common/intl';
import { encryptionState, setTlsConfig } from 'panel/stores/encryption';
import { toNumber } from 'panel/helpers/form';
import { validateServerName, validatePort, validateIsSafePort } from 'panel/helpers/validators';
import s from '../styles.module.pcss';
import theme from 'panel/lib/theme';

type Props = {
    open: boolean;
    onClose: () => void;
};

export const ServerSettingsModal = (props: Props) => {
    const [serverName, setServerName] = createSignal('');
    const [portHttps, setPortHttps] = createSignal(0);
    const [portDot, setPortDot] = createSignal(0);
    const [portDoq, setPortDoq] = createSignal(0);
    const [errors, setErrors] = createSignal<Record<string, string>>({});

    createEffect(
        on(
            () => props.open,
            (open) => {
                if (open) {
                    setServerName(encryptionState.server_name || '');
                    setPortHttps(encryptionState.port_https || 0);
                    setPortDot(encryptionState.port_dns_over_tls || 0);
                    setPortDoq(encryptionState.port_dns_over_quic || 0);
                    setErrors({});
                }
            },
        ),
    );

    const clearError = (field: string) => {
        setErrors((prev) => {
            const next = { ...prev };
            delete next[field];
            return next;
        });
    };

    const hasErrors = () => Object.values(errors()).some(Boolean);

    const handleServerNameChange = (e: Event) => {
        setServerName((e.target as HTMLInputElement).value);
        clearError('server_name');
    };

    const handlePortHttpsChange = (e: Event) => {
        setPortHttps(toNumber((e.target as HTMLInputElement).value));
        clearError('port_https');
    };

    const handlePortDotChange = (e: Event) => {
        setPortDot(toNumber((e.target as HTMLInputElement).value));
        clearError('port_dns_over_tls');
    };

    const handlePortDoqChange = (e: Event) => {
        setPortDoq(toNumber((e.target as HTMLInputElement).value));
        clearError('port_dns_over_quic');
    };

    const handleServerNameBlur = () => {
        const err = validateServerName(serverName());
        setErrors((prev) => {
            const next = { ...prev };
            if (err) {
                next.server_name = err;
            } else {
                delete next.server_name;
            }
            return next;
        });
    };

    const handlePortHttpsBlur = () => {
        const err = validatePort(portHttps()) || validateIsSafePort(portHttps());
        setErrors((prev) => {
            const next = { ...prev };
            if (err) {
                next.port_https = err as string;
            } else {
                delete next.port_https;
            }
            return next;
        });
    };

    const handlePortDotBlur = () => {
        const err = validatePort(portDot());
        setErrors((prev) => {
            const next = { ...prev };
            if (err) {
                next.port_dns_over_tls = err as string;
            } else {
                delete next.port_dns_over_tls;
            }
            return next;
        });
    };

    const handlePortDoqBlur = () => {
        const err = validatePort(portDoq());
        setErrors((prev) => {
            const next = { ...prev };
            if (err) {
                next.port_dns_over_quic = err as string;
            } else {
                delete next.port_dns_over_quic;
            }
            return next;
        });
    };

    const save = () => {
        setTlsConfig({
            server_name: serverName(),
            port_https: portHttps(),
            port_dns_over_tls: portDot(),
            port_dns_over_quic: portDoq(),
        });
        props.onClose();
    };

    const processing = () => encryptionState.processingConfig || encryptionState.processingValidate;

    return (
        <ConfigDialog
            open={props.open}
            title={intl.getMessage('encrypted_dns_settings')}
            onClose={props.onClose}
            onSubmit={save}
            processing={processing()}
            submitDisabled={processing() || hasErrors()}
        >
            <div class={theme.form.input}>
                <Input
                    id="server_name"
                    name="server_name"
                    value={serverName()}
                    onChange={handleServerNameChange}
                    onBlur={handleServerNameBlur}
                    label={
                        <>
                            {intl.getMessage('encryption_server')}
                            <FaqTooltip
                                menuSize="large"
                                text={
                                    <>
                                        <div class={s.tooltipText}>
                                            {intl.getMessage('encryption_server_tooltip_1')}
                                        </div>
                                        <div class={s.tooltipText}>
                                            {intl.getMessage('encryption_server_tooltip_2')}
                                        </div>
                                    </>
                                }
                            />
                        </>
                    }
                    placeholder={intl.getMessage('encryption_server_enter')}
                    errorMessage={errors().server_name}
                    size="large"
                />
            </div>
            <div class={theme.form.input}>
                <Input
                    id="port_https"
                    name="port_https"
                    type="number"
                    value={portHttps()}
                    onChange={handlePortHttpsChange}
                    onBlur={handlePortHttpsBlur}
                    label={
                        <>
                            {intl.getMessage('encryption_https')}
                            <FaqTooltip
                                menuSize="large"
                                text={intl.getMessage('encryption_https_tooltip')}
                            />
                        </>
                    }
                    errorMessage={errors().port_https}
                    size="large"
                />
            </div>
            <div class={theme.form.input}>
                <Input
                    id="port_dns_over_tls"
                    name="port_dns_over_tls"
                    type="number"
                    value={portDot()}
                    onChange={handlePortDotChange}
                    onBlur={handlePortDotBlur}
                    label={
                        <>
                            {intl.getMessage('encryption_dot')}
                            <FaqTooltip
                                menuSize="large"
                                text={intl.getMessage('encryption_dot_tooltip')}
                            />
                        </>
                    }
                    errorMessage={errors().port_dns_over_tls}
                    size="large"
                />
            </div>
            <div class={theme.form.input}>
                <Input
                    id="port_dns_over_quic"
                    name="port_dns_over_quic"
                    type="number"
                    value={portDoq()}
                    onChange={handlePortDoqChange}
                    onBlur={handlePortDoqBlur}
                    label={
                        <>
                            {intl.getMessage('encryption_doq')}
                            <FaqTooltip
                                menuSize="large"
                                text={intl.getMessage('encryption_doq_tooltip')}
                            />
                        </>
                    }
                    errorMessage={errors().port_dns_over_quic}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
