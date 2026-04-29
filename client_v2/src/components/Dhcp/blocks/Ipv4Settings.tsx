import React, { useState, useEffect } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';

import s from '../Dhcp.module.pcss';

type V4Config = {
    gateway_ip: string;
    subnet_mask: string;
    range_start: string;
    range_end: string;
    lease_duration: number;
};

type Props = {
    v4: V4Config | undefined;
    interfaces: Record<string, any> | undefined;
    selectedInterface: string;
    processingConfig: boolean;
    onSave: (values: V4Config) => void;
};

export const Ipv4Settings = ({ v4, interfaces, selectedInterface, processingConfig, onSave }: Props) => {
    const [gatewayIp, setGatewayIp] = useState(v4?.gateway_ip || '');
    const [subnetMask, setSubnetMask] = useState(v4?.subnet_mask || '');
    const [rangeStart, setRangeStart] = useState(v4?.range_start || '');
    const [rangeEnd, setRangeEnd] = useState(v4?.range_end || '');
    const [leaseDuration, setLeaseDuration] = useState<string>(
        v4?.lease_duration ? String(v4.lease_duration) : '',
    );

    useEffect(() => {
        setGatewayIp(v4?.gateway_ip || '');
        setSubnetMask(v4?.subnet_mask || '');
        setRangeStart(v4?.range_start || '');
        setRangeEnd(v4?.range_end || '');
        setLeaseDuration(v4?.lease_duration ? String(v4.lease_duration) : '');
    }, [v4]);

    const hasIpv4 = !!(interfaces && interfaces[selectedInterface]?.ipv4_addresses);
    const isEmptyConfig = !gatewayIp && !subnetMask && !rangeStart && !rangeEnd && !leaseDuration;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSave({
            gateway_ip: gatewayIp,
            subnet_mask: subnetMask,
            range_start: rangeStart,
            range_end: rangeEnd,
            lease_duration: leaseDuration ? Number(leaseDuration) : 0,
        });
    };

    return (
        <form onSubmit={handleSubmit} className={s.form}>
            <div className={cn(theme.form.group, s.formGroup)}>
                <div className={s.formField}>
                    <Input
                        id="v4_gateway_ip"
                        label={intl.getMessage('dhcp_form_gateway_address')}
                        placeholder="192.168.1.1"
                        value={gatewayIp}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setGatewayIp(e.target.value)}
                        disabled={!hasIpv4}
                    />
                </div>

                <div className={s.formField}>
                    <span className={cn(theme.text.t3, s.formFieldLabel)}>
                        {intl.getMessage('dhcp_form_range_title_v2')}
                    </span>
                    <div className={s.rangeRow}>
                        <Input
                            id="v4_range_start"
                            placeholder="192.168.1.2"
                            value={rangeStart}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRangeStart(e.target.value)}
                            disabled={!hasIpv4}
                        />
                        <Input
                            id="v4_range_end"
                            placeholder="192.168.1.254"
                            value={rangeEnd}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRangeEnd(e.target.value)}
                            disabled={!hasIpv4}
                        />
                    </div>
                </div>

                <div className={s.formField}>
                    <Input
                        id="v4_subnet_mask"
                        label={intl.getMessage('dhcp_form_subnet_input_v2')}
                        placeholder="255.255.255.0"
                        value={subnetMask}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSubnetMask(e.target.value)}
                        disabled={!hasIpv4}
                    />
                </div>

                <div className={s.formField}>
                    <Input
                        id="v4_lease_duration"
                        type="number"
                        label={intl.getMessage('dhcp_form_lease_title_v2')}
                        placeholder="86400"
                        value={leaseDuration}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLeaseDuration(e.target.value)}
                        disabled={!hasIpv4}
                    />
                </div>
            </div>

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={processingConfig || !hasIpv4 || isEmptyConfig}
                    className={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
