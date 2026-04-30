import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Select } from 'panel/common/controls/Select';

import s from '../Dhcp.module.pcss';

type InterfaceOption = {
    value: string;
    label: string;
};

type Props = {
    interfaces?: Record<string, any>;
    selectedInterface: string;
    enabled: boolean;
    onChange: (name: string) => void;
};

export const InterfaceSelect = ({ interfaces, selectedInterface, enabled, onChange }: Props) => {
    if (!interfaces) {
        return null;
    }

    const options: InterfaceOption[] = Object.keys(interfaces).map((key) => {
        const iface = interfaces[key];
        const name = iface.name || key;
        const ipv4 = iface.ipv4_addresses?.[0] || '';
        const ipv6 = iface.ipv6_addresses?.[0] || '';
        const parts = [name, ipv4, ipv6].filter(Boolean);
        return {
            value: name,
            label: parts.join(' — '),
        };
    });

    const selected = options.find((opt) => opt.value === selectedInterface) || null;

    return (
        <div>
            <span className={cn(theme.text.t3, s.fieldLabel)}>
                {intl.getMessage('dhcp_interface_select_v2')}
            </span>
            <Select
                id="dhcp_interface"
                options={options}
                value={selected}
                onChange={(option: any) => onChange(option.value)}
                isDisabled={enabled}
                placeholder={intl.getMessage('dhcp_interface_select_v2')}
                size="responsive"
                height="medium"
            />
        </div>
    );
};
