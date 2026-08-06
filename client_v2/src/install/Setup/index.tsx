import { createMemo, createEffect, onMount, onCleanup, Show, Switch, Match } from 'solid-js';

import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { SetupGuide } from 'panel/components/SetupGuide/SetupGuide';
import {
    installState,
    getDefaultAddresses,
    nextStep,
    setAuthData,
    setAllSettings,
    checkConfig,
} from 'panel/stores/install';
import { dashboardState } from 'panel/stores/dashboard';

import { getInterfaceIp, getWebAddress } from '../../helpers/helpers';
import {
    INSTALL_TOTAL_STEPS,
    ALL_INTERFACES_IP,
    DEBOUNCE_TIMEOUT,
    LANGUAGE_QUERY_PARAM,
} from '../../helpers/constants';

import { Greeting } from './Greeting';
import type { ConfigType, DnsConfig, SettingsFormValues, WebConfig } from './types';
import type { InitialConfiguration } from 'panel/api/model/initialConfiguration';
import type { AuthFormValues } from './Auth';
import { InterfaceSettings } from './InterfaceSettings';
import { DnsSettings } from './DnsSettings';
import { Controls } from './Controls';
import { Submit } from './Submit';
import { Progress } from './blocks/Progress';
import { Auth } from './Auth';
import { Toasts } from 'panel/components/Toasts';

import styles from './styles.module.pcss';
import { getDnsAddressWithPort } from './helpers/helpers';

const getInstallDnsAddresses = (
    dns: { ip: string; port: number },
    interfaces: { ip_addresses?: string[] }[],
) => {
    if (!dns?.ip || !dns?.port) {
        return [];
    }

    if (dns.ip === ALL_INTERFACES_IP) {
        return (interfaces || [])
            .filter((iface) => iface?.ip_addresses?.length > 0)
            .map((iface) => getInterfaceIp(iface))
            .filter(Boolean)
            .map((ip: string) => getDnsAddressWithPort(ip, dns.port));
    }

    return [getDnsAddressWithPort(dns.ip, dns.port)];
};

export const Setup = () => {
    const step = () => installState.step;
    const web = () => installState.web;
    const dns = () => installState.dns;
    const staticIp = () => installState.staticIp;
    const interfaces = () => installState.interfaces;
    const auth = () => installState.auth;
    const processingDefault = () => installState.processingDefault;

    const installDnsAddresses = createMemo(() => getInstallDnsAddresses(dns(), interfaces()));

    const resolvedDnsAddresses = createMemo(() => {
        const dnsAddresses = dashboardState.dnsAddresses || [];
        return dnsAddresses.length > 0 ? dnsAddresses : installDnsAddresses();
    });

    onMount(() => {
        getDefaultAddresses();
    });

    createEffect(() => {
        step();
        window.scrollTo({ top: 0, behavior: 'instant' });
    });

    const handleNextStep = () => {
        if (step() <= INSTALL_TOTAL_STEPS) {
            nextStep();
        }
    };

    const handleAuthSubmit = (values: AuthFormValues) => {
        setAuthData(values);
        handleNextStep();
    };

    const handleFinalSubmit = () => {
        const config: InitialConfiguration & { confirm_password: string } = {
            web: web(),
            dns: dns(),
            language: installState.language,
            username: auth().username,
            password: auth().password,
            confirm_password: auth().password,
        };

        if (web().port && dns().port) {
            setAllSettings(config);
        }
    };

    // Debounced checkConfig
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;
    const debouncedCheckConfig = (values: SettingsFormValues) => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }
        debounceTimer = setTimeout(() => {
            const { web, dns } = values;
            if (values && web.port && dns.port) {
                checkConfig({ web, dns, set_static_ip: false });
            }
        }, DEBOUNCE_TIMEOUT);
    };

    onCleanup(() => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }
    });

    const handleFix = (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => {
        checkConfig({ web, dns, set_static_ip });
    };

    const openDashboard = (ip: string, port: number) => {
        const host = ip === ALL_INTERFACES_IP ? window.location.hostname : ip;
        const url = new URL(getWebAddress(host, port));
        url.searchParams.set(LANGUAGE_QUERY_PARAM, installState.language);
        window.location.replace(url.toString());
    };

    const config = createMemo<ConfigType>(() => ({
        web: web(),
        dns: dns(),
        staticIp: staticIp(),
    }));

    return (
        <Show when={!processingDefault()}>
            <div class={styles.setup}>
                <PublicHeader
                    dropdownClass={styles.dropdown}
                    dropdownPosition="bottomRight"
                    center={<Progress step={step()} />}
                    useLocalLanguage={true}
                    hideLanguageDropdown={true}
                />

                <div class={styles.container}>
                    <Switch>
                        <Match when={step() === 1}>
                            <Greeting />
                        </Match>
                        <Match when={step() === 2}>
                            <Auth onAuthSubmit={handleAuthSubmit} initialValues={auth()} />
                        </Match>
                        <Match when={step() === 3}>
                            <InterfaceSettings
                                config={config()}
                                initialValues={config()}
                                interfaces={interfaces()}
                                handleSubmit={handleNextStep}
                                validateForm={debouncedCheckConfig}
                                handleFix={handleFix}
                            />
                        </Match>
                        <Match when={step() === 4}>
                            <DnsSettings
                                config={config()}
                                initialValues={config()}
                                interfaces={interfaces()}
                                handleSubmit={handleNextStep}
                                validateForm={debouncedCheckConfig}
                                handleFix={handleFix}
                            />
                        </Match>
                        <Match when={step() === 5}>
                            <SetupGuide
                                dnsAddresses={resolvedDnsAddresses()}
                                isStep
                                footer={<Controls />}
                            />
                        </Match>
                        <Match when={step() === 6}>
                            <Submit
                                openDashboard={openDashboard}
                                webConfig={web()}
                                onSubmit={handleFinalSubmit}
                            />
                        </Match>
                    </Switch>
                </div>
            </div>

            <Toasts />
        </Show>
    );
};
