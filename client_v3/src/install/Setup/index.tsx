import { createMemo, onMount, onCleanup, Show, Switch, Match } from 'solid-js';

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
import { INSTALL_TOTAL_STEPS, ALL_INTERFACES_IP, DEBOUNCE_TIMEOUT } from '../../helpers/constants';

import Greeting from './Greeting';
import type { ConfigType, DnsConfig, WebConfig } from './types';
import { InterfaceSettings } from './InterfaceSettings';
import { DnsSettings } from './DnsSettings';
import { Controls } from './Controls';
import { Submit } from './Submit';
import { Progress } from './blocks/Progress';
import { Auth } from './Auth';
import Toasts from '../../components/Toasts';

import styles from './styles.module.pcss';
import { getDnsAddressWithPort } from './helpers/helpers';

type InstallInterface = {
    name?: string;
    ip_addresses?: string[];
};

const getInstallDnsAddresses = (
    dns: { ip: string; port: number },
    interfaces: InstallInterface[],
) => {
    if (!dns?.ip || !dns?.port) {
        return [];
    }

    if (dns.ip === ALL_INTERFACES_IP) {
        return (interfaces || [])
            .filter((iface: InstallInterface) => iface?.ip_addresses?.length > 0)
            .map((iface: InstallInterface) => getInterfaceIp(iface))
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

    const handleNextStep = () => {
        if (step() <= INSTALL_TOTAL_STEPS) {
            nextStep();
        }
    };

    const handleAuthSubmit = (values: any) => {
        setAuthData(values);
        handleNextStep();
    };

    const handleFinalSubmit = () => {
        const config: any = {
            web: web(),
            dns: dns(),
            ...auth(),
        };

        if (web().port && dns().port) {
            setAllSettings(config);
        }
    };

    // Debounced checkConfig
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;
    const debouncedCheckConfig = (values: any) => {
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
        let address = getWebAddress(ip, port);
        if (ip === ALL_INTERFACES_IP) {
            address = getWebAddress(window.location.hostname, port);
        }
        window.location.replace(address);
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
