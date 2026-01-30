import { useEffect, useMemo } from 'react';
import { useForm } from 'react-hook-form';

import { ALL_INTERFACES_IP, STANDARD_DNS_PORT, STANDARD_WEB_PORT } from '../../../helpers/constants';
import { validateInstallPort } from '../../../helpers/validators';

import type { DnsConfig, SettingsFormValues, WebConfig, ConfigType } from '../types';

type HandleFix = (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => void;

export const createHandleAutofix = (watchFields: SettingsFormValues, handleFix: HandleFix) => (type: 'web' | 'dns') => {
    const web = {
        ip: watchFields.web?.ip,
        port: watchFields.web?.port,
        autofix: false,
    };
    const dns = {
        ip: watchFields.dns?.ip,
        port: watchFields.dns?.port,
        autofix: false,
    };

    if (type === 'web') {
        web.autofix = true;
    } else {
        dns.autofix = true;
    }

    handleFix(web, dns, false);
};

export const useInstallSettingsForm = (config: ConfigType, validateForm: (data: SettingsFormValues) => void) => {
    const defaultValues = useMemo(
        () => ({
            web: {
                ip: config.web.ip || ALL_INTERFACES_IP,
                port: config.web.port || STANDARD_WEB_PORT,
            },
            dns: {
                ip: config.dns.ip || ALL_INTERFACES_IP,
                port: config.dns.port || STANDARD_DNS_PORT,
            },
        }),
        [config.dns.ip, config.dns.port, config.web.ip, config.web.port],
    );

    const {
        control,
        watch,
        handleSubmit: reactHookFormSubmit,
        formState: { isValid, isDirty },
    } = useForm<SettingsFormValues>({
        defaultValues,
        mode: 'onBlur',
    });

    const watchFields = watch();

    const webIpVal = watch('web.ip');
    const webPortVal = watch('web.port');
    const dnsIpVal = watch('dns.ip');
    const dnsPortVal = watch('dns.port');

    useEffect(() => {
        if (!isDirty) {
            return;
        }

        const webPortError = validateInstallPort(webPortVal);
        const dnsPortError = validateInstallPort(dnsPortVal);

        if (webPortError || dnsPortError) {
            return;
        }

        validateForm({
            web: {
                ip: webIpVal,
                port: webPortVal,
            },
            dns: {
                ip: dnsIpVal,
                port: dnsPortVal,
            },
        });
    }, [dnsIpVal, dnsPortVal, isDirty, validateForm, webIpVal, webPortVal]);

    return {
        control,
        reactHookFormSubmit,
        isValid,
        isDirty,
        watchFields,
        webIpVal,
        webPortVal,
        dnsIpVal,
        dnsPortVal,
    };
};
