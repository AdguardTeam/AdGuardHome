import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { installGetAddresses, installConfigure, installCheckConfig } from 'panel/api/generated';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import type { InstallInterface } from '../initialState';
import type { NetInterface } from 'panel/api/model/netInterface';
import type { InitialConfiguration } from 'panel/api/model/initialConfiguration';
import type { CheckConfigRequest } from 'panel/api/model/checkConfigRequest';
import {
    ALL_INTERFACES_IP,
    INSTALL_FIRST_STEP,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
} from 'panel/helpers/constants';

type InstallState = {
    step: number;
    processingDefault: boolean;
    processingSubmit: boolean;
    processingCheck: boolean;
    submitted: boolean;
    auth: {
        username: string;
        password: string;
        privacy_consent: boolean;
    };
    web: {
        ip: string;
        port: number;
        status: string;
        can_autofix: boolean;
    };
    dns: {
        ip: string;
        port: number;
        status: string;
        can_autofix: boolean;
    };
    staticIp: {
        static: string;
        ip: string;
        error: string;
    };
    interfaces: InstallInterface[];
    dnsVersion: string;
};

const initialState: InstallState = {
    step: INSTALL_FIRST_STEP,
    processingDefault: true,
    processingSubmit: false,
    processingCheck: false,
    submitted: false,
    auth: { username: '', password: '', privacy_consent: false },
    web: { ip: ALL_INTERFACES_IP, port: STANDARD_WEB_PORT, status: '', can_autofix: false },
    dns: { ip: ALL_INTERFACES_IP, port: STANDARD_DNS_PORT, status: '', can_autofix: false },
    staticIp: { static: '', ip: '', error: '' },
    interfaces: [],
    dnsVersion: '',
};

const [state, setState] = createStore<InstallState>(initialState);

export const getDefaultAddresses = async () => {
    setState('processingDefault', true);
    try {
        const data = await installGetAddresses();
        const normalizedInterfaces = Array.isArray(data.interfaces)
            ? data.interfaces
            : Object.entries(data.interfaces || {}).map(
                  ([name, iface]: [string, NetInterface]) => ({
                      flags: iface.flags,
                      hardware_address: iface.hardware_address,
                      ip_addresses: [...iface.ipv4_addresses, ...iface.ipv6_addresses],
                      mtu: 0,
                      name: iface.name || name,
                  }),
              );
        setState({
            web: { ...state.web, port: data.web_port },
            dns: { ...state.dns, port: data.dns_port },
            interfaces: normalizedInterfaces,
            processingDefault: false,
            dnsVersion: data.version,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingDefault', false);
    }
};

export const nextStep = () => {
    setState('step', (prev) => prev + 1);
};

export const prevStep = () => {
    setState('step', (prev) => prev - 1);
};

export const setAuthData = (auth: Partial<InstallState['auth']>) => {
    setState('auth', (prev) => ({ ...prev, ...auth }));
};

export const setAllSettings = async (
    config: InitialConfiguration & { confirm_password: string },
) => {
    setState({ processingSubmit: true, submitted: false });
    try {
        const { confirm_password, ...rest } = config;
        void confirm_password;
        await installConfigure(rest);
        setState({ processingSubmit: false, submitted: true });
        addSuccessToast(intl.getMessage('install_saved'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingSubmit', false);
    }
};

export const checkConfig = async (values: CheckConfigRequest) => {
    setState('processingCheck', true);
    try {
        const data = await installCheckConfig(values);
        setState({
            web: {
                ip: values.web?.ip ?? '',
                port: values.web?.port ?? 0,
                ...data.web,
            },
            dns: {
                ip: values.dns?.ip ?? '',
                port: values.dns?.port ?? 0,
                ...data.dns,
            },
            staticIp: { ...untrack(() => state.staticIp), ...data.static_ip },
            processingCheck: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingCheck', false);
    }
};

export const installState = untrack(() => state);
