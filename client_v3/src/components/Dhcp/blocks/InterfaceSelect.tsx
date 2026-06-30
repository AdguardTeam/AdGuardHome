import { Show, createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Select } from 'panel/common/controls/Select';
import { type DhcpInterfaces } from 'panel/initialState';

import s from '../Dhcp.module.pcss';

type InterfaceOption = {
    value: string;
    label: string;
};

type Props = {
    interfaces?: DhcpInterfaces;
    selectedInterface: string;
    enabled: boolean;
    onChange: (name: string) => void;
};

export const InterfaceSelect = (props: Props) => {
    const options = createMemo<InterfaceOption[]>(() => {
        if (!props.interfaces) return [];
        return Object.keys(props.interfaces).map((key) => {
            const iface = props.interfaces![key];
            const name = iface.name || key;
            const ipv4 = iface.ipv4_addresses?.[0] || '';
            const ipv6 = iface.ipv6_addresses?.[0] || '';
            const parts = [name, ipv4, ipv6].filter(Boolean);
            return {
                value: name,
                label: parts.join(' — '),
            };
        });
    });

    const selected = createMemo(
        () => options().find((opt) => opt.value === props.selectedInterface) || null,
    );

    return (
        <Show when={props.interfaces}>
            <div>
                <span class={cn(theme.text.t3, s.fieldLabel)}>
                    {intl.getMessage('dhcp_interface_select')}
                </span>
                <Select
                    id="dhcp_interface"
                    options={options()}
                    value={selected()}
                    onChange={(option: any) => props.onChange(option.value)}
                    isDisabled={props.enabled}
                    placeholder={intl.getMessage('dhcp_interface_select')}
                    size="responsive"
                    height="big"
                />
            </div>
        </Show>
    );
};
