import { createMemo, untrack } from 'solid-js';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import styles from 'panel/install/Setup/styles.module.pcss';
import cn from 'clsx';
import { Controls } from './Controls';
import { WebBanner } from './blocks/Banner';
import { AddressList } from './blocks';
import { buildInterfaceOptions } from './helpers/InterfaceOptions';
import { createHandleAutofix, useInstallSettingsForm } from './helpers/useInstallSettingsForm';

import { ALL_INTERFACES_IP, STATUS_RESPONSE, STANDARD_WEB_PORT } from '../../helpers/constants';

import { InstallInterface } from '../../initialState';

import type { ConfigType, DnsConfig, SettingsFormValues, WebConfig } from './types';

type Props = {
    handleSubmit: (data: SettingsFormValues) => void;
    handleChange?: (data: SettingsFormValues) => void;
    handleFix: (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => void;
    validateForm: (data: SettingsFormValues) => void;
    config: ConfigType;
    interfaces: InstallInterface[];
    initialValues?: object;
};

export const InterfaceSettings = (props: Props) => {
    const form = useInstallSettingsForm(
        untrack(() => props.config),
        untrack(() => props.validateForm),
    );

    const webStatus = () => props.config.web.status;
    const isWebFixAvailable = () => props.config.web.can_autofix;
    const staticIp = () => props.config.staticIp;

    const webIpOptions = createMemo(() => buildInterfaceOptions(props.interfaces));

    const handleAutofix = createHandleAutofix(
        form.watchFields,
        untrack(() => props.handleFix),
    );

    const handleStaticIp = (ip: string) => {
        const fields = form.watchFields();
        const web = {
            ip: fields.web?.ip,
            port: fields.web?.port,
            autofix: false,
        };
        const dns = {
            ip: fields.dns?.ip,
            port: fields.dns?.port,
            autofix: false,
        };
        const set_static_ip = true;

        if (window.confirm(intl.getMessage('confirm_static_ip', { ip }))) {
            props.handleFix(web, dns, set_static_ip);
        }
    };

    const getStaticIpMessage = createMemo(() => {
        const staticIpVal = staticIp();
        const { static: status, ip } = staticIpVal;

        switch (status) {
            case STATUS_RESPONSE.NO:
                return (
                    <>
                        <div class={styles.spacerBottom}>
                            {intl.getMessage('install_static_configure', { ip })}
                        </div>

                        <Button
                            type="button"
                            size="small"
                            variant="secondary"
                            class={styles.button}
                            onClick={() => handleStaticIp(ip)}
                        >
                            {intl.getMessage('set_static_ip')}
                        </Button>
                    </>
                );
            case STATUS_RESPONSE.ERROR:
                return (
                    <div class={styles.errorText}>{intl.getMessage('install_static_error')}</div>
                );
            case STATUS_RESPONSE.YES:
                return <div class={styles.successText}>{intl.getMessage('install_static_ok')}</div>;
            default:
                return null;
        }
    });

    const onSubmit = (data: SettingsFormValues) => {
        props.validateForm(data);
        props.handleSubmit(data);
    };

    return (
        <div class={styles.configSetting}>
            <form class={styles.step} onSubmit={(e) => form.handleSubmit(onSubmit)(e)}>
                <div class={styles.info}>
                    <div>
                        <div class={styles.titleStep}>{intl.getMessage('setup_ui_title')}</div>

                        <p class={styles.descAdresses}>{intl.getMessage('setup_ui_desc')}</p>

                        <WebBanner
                            class={cn(styles.banner, styles.bannerMobile)}
                            webIp={form.webIp}
                            webPort={form.webPort}
                            setWebIp={form.setWebIp}
                            setWebPort={form.setWebPort}
                            webIpOptions={webIpOptions()}
                            webStatus={webStatus()}
                            isWebFixAvailable={isWebFixAvailable()}
                            onAutofix={() => handleAutofix('web')}
                        />
                    </div>

                    <AddressList
                        interfaces={props.interfaces}
                        address={form.webIp() || ALL_INTERFACES_IP}
                        port={form.webPort() || STANDARD_WEB_PORT}
                    />

                    <div class={styles.group}>{getStaticIpMessage()}</div>

                    <Controls invalid={!form.isValid()} />
                </div>

                <div class={styles.content}>
                    <WebBanner
                        class={styles.banner}
                        webIp={form.webIp}
                        webPort={form.webPort}
                        setWebIp={form.setWebIp}
                        setWebPort={form.setWebPort}
                        webIpOptions={webIpOptions()}
                        webStatus={webStatus()}
                        isWebFixAvailable={isWebFixAvailable()}
                        onAutofix={() => handleAutofix('web')}
                    />
                </div>
            </form>
        </div>
    );
};
