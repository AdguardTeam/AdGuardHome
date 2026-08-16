import { Show, For, createMemo } from 'solid-js';

import { CopiedText } from 'panel/common/ui/CopiedText/CopiedText';
import { getInterfaceIp } from 'panel/helpers/helpers';
import { ALL_INTERFACES_IP } from 'panel/helpers/constants';
import { InstallInterface } from 'panel/initialState';
import styles from '../styles.module.pcss';
import { stripZoneId, getDnsAddressWithPort } from '../helpers/helpers';

const getWebAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `http://[${normalizedIp}]:${port}`;
    }

    return `http://${normalizedIp}:${port}`;
};

type RenderItemProps = {
    ip: string;
    port: number;
    isDns: boolean;
    interfaceName?: string;
};

const RenderItem = (props: RenderItemProps) => {
    const webAddress = createMemo(() => getWebAddressWithPort(props.ip, props.port));
    const dnsAddress = createMemo(() => getDnsAddressWithPort(props.ip, props.port));

    return (
        <li class={styles.addressListItem}>
            <Show when={props.isDns} fallback={<CopiedText text={webAddress()} />}>
                <CopiedText text={dnsAddress()} />
            </Show>
        </li>
    );
};

type AddressListProps = {
    interfaces: InstallInterface[];
    address: string;
    port: number;
    isDns?: boolean;
};

export const AddressList = (props: AddressListProps) => (
    <ul class={styles.addressList}>
        <Show
            when={props.address === ALL_INTERFACES_IP}
            fallback={
                <RenderItem ip={props.address} port={props.port} isDns={props.isDns ?? false} />
            }
        >
            <For
                each={Object.values(props.interfaces)
                    .filter((iface: InstallInterface) => iface?.ip_addresses?.length > 0)
                    .sort((a: InstallInterface, b: InstallInterface) => {
                        const aIp = stripZoneId(getInterfaceIp(a));
                        const bIp = stripZoneId(getInterfaceIp(b));

                        const ipCompare = aIp.localeCompare(bIp);
                        if (ipCompare !== 0) {
                            return ipCompare;
                        }

                        return (a.name ?? '').localeCompare(b.name ?? '');
                    })}
            >
                {(iface: InstallInterface) => {
                    const ip = getInterfaceIp(iface);
                    return (
                        <RenderItem
                            ip={ip}
                            port={props.port}
                            isDns={props.isDns ?? false}
                            interfaceName={iface.name}
                        />
                    );
                }}
            </For>
        </Show>
    </ul>
);
