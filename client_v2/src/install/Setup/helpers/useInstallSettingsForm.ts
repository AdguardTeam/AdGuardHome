import { createSignal, createEffect, createMemo, untrack, onMount } from 'solid-js';

import { ALL_INTERFACES_IP, STANDARD_DNS_PORT, STANDARD_WEB_PORT } from 'panel/helpers/constants';
import { validateInstallPort } from 'panel/helpers/validators';

import type { DnsConfig, SettingsFormValues, WebConfig, ConfigType } from '../types';

type HandleFix = (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => void;

export const createHandleAutofix =
    (getFields: () => SettingsFormValues, handleFix: HandleFix) => (type: 'web' | 'dns') => {
        const fields = getFields();
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

        if (type === 'web') {
            web.autofix = true;
        } else {
            dns.autofix = true;
        }

        handleFix(web, dns, false);
    };

export const useInstallSettingsForm = (
    config: ConfigType,
    validateForm: (data: SettingsFormValues) => void,
) => {
    const [webIp, setWebIp] = createSignal(config.web.ip || ALL_INTERFACES_IP);
    const [webPort, setWebPort] = createSignal(config.web.port || STANDARD_WEB_PORT);
    const [dnsIp, setDnsIp] = createSignal(config.dns.ip || ALL_INTERFACES_IP);
    const [dnsPort, setDnsPort] = createSignal(config.dns.port || STANDARD_DNS_PORT);
    const [isDirty, setIsDirty] = createSignal(false);

    const watchFields = createMemo<SettingsFormValues>(() => ({
        web: {
            ip: webIp(),
            port: webPort(),
        },
        dns: {
            ip: dnsIp(),
            port: dnsPort(),
        },
    }));

    // Track changes to mark form as dirty
    const handleWebIpChange = (value: string) => {
        setWebIp(value);
        setIsDirty(true);
    };

    const handleWebPortChange = (value: number) => {
        setWebPort(value);
        setIsDirty(true);
    };

    const handleDnsIpChange = (value: string) => {
        setDnsIp(value);
        setIsDirty(true);
    };

    const handleDnsPortChange = (value: number) => {
        setDnsPort(value);
        setIsDirty(true);
    };

    const runValidation = () => {
        const webPortVal = webPort();
        const dnsPortVal = dnsPort();
        const webIpVal = webIp();
        const dnsIpVal = dnsIp();

        const webPortError = validateInstallPort(webPortVal);
        const dnsPortError = validateInstallPort(dnsPortVal);

        if (webPortError || dnsPortError) {
            return;
        }

        validateForm({
            web: { ip: webIpVal, port: webPortVal },
            dns: { ip: dnsIpVal, port: dnsPortVal },
        });
    };

    onMount(runValidation);

    createEffect(() => {
        if (!isDirty()) {
            return;
        }
        runValidation();
    });

    const isValid = createMemo(() => {
        const webPortVal = webPort();
        const dnsPortVal = dnsPort();
        return !validateInstallPort(webPortVal) && !validateInstallPort(dnsPortVal);
    });

    const handleSubmit = (onSubmit: (data: SettingsFormValues) => void) => (e: Event) => {
        e.preventDefault();
        onSubmit(untrack(watchFields));
    };

    return {
        webIp,
        webPort,
        dnsIp,
        dnsPort,
        setWebIp: handleWebIpChange,
        setWebPort: handleWebPortChange,
        setDnsIp: handleDnsIpChange,
        setDnsPort: handleDnsPortChange,
        handleSubmit,
        isValid,
        isDirty,
        watchFields,
    };
};
