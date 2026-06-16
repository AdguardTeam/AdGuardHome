import { createSignal, createEffect, createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import type { DhcpInterfaces } from 'panel/initialState';
import {
    validateIpv4,
    validateIpv4RangeEnd,
    validateNotInRange,
    validateGatewaySubnetMask,
    validateIpForGatewaySubnetMask,
} from 'panel/helpers/validators';

import s from '../Dhcp.module.pcss';

type V4Config = {
    gateway_ip: string;
    subnet_mask: string;
    range_start: string;
    range_end: string;
    lease_duration: number;
};

type Props = {
    v4?: V4Config;
    interfaces?: DhcpInterfaces;
    selectedInterface: string;
    processingConfig: boolean;
    onSave: (values: V4Config) => void;
};

export const Ipv4Settings = (props: Props) => {
    const [gatewayIp, setGatewayIp] = createSignal('');
    const [subnetMask, setSubnetMask] = createSignal('');
    const [rangeStart, setRangeStart] = createSignal('');
    const [rangeEnd, setRangeEnd] = createSignal('');
    const [leaseDuration, setLeaseDuration] = createSignal('');

    const [gatewayIpError, setGatewayIpError] = createSignal('');
    const [subnetMaskError, setSubnetMaskError] = createSignal('');
    const [rangeStartError, setRangeStartError] = createSignal('');
    const [rangeEndError, setRangeEndError] = createSignal('');

    // Reset form when v4 changes
    createEffect(() => {
        const v4 = props.v4;
        setGatewayIp(v4?.gateway_ip || '');
        setSubnetMask(v4?.subnet_mask || '');
        setRangeStart(v4?.range_start || '');
        setRangeEnd(v4?.range_end || '');
        setLeaseDuration(v4?.lease_duration ? String(v4.lease_duration) : '');
    });

    const hasIpv4 = createMemo(() =>
        !!(props.interfaces && props.interfaces[props.selectedInterface]?.ipv4_addresses),
    );

    const allValues = createMemo(() => ({
        v4: {
            gateway_ip: gatewayIp(),
            subnet_mask: subnetMask(),
            range_start: rangeStart(),
            range_end: rangeEnd(),
            lease_duration: leaseDuration(),
        },
    }));

    const isEmptyConfig = createMemo(() =>
        !gatewayIp() && !subnetMask() && !rangeStart() && !rangeEnd() && !leaseDuration(),
    );

    const validateGatewayIp = () => {
        const err = validateIpv4(gatewayIp()) || validateNotInRange(gatewayIp(), allValues());
        setGatewayIpError(err || '');
    };

    const validateRangeStart = () => {
        const err = validateIpv4(rangeStart()) || validateIpForGatewaySubnetMask(rangeStart(), allValues());
        setRangeStartError(err || '');
    };

    const validateRangeEnd = () => {
        const err =
            validateIpv4(rangeEnd()) ||
            validateIpv4RangeEnd(undefined, allValues()) ||
            validateIpForGatewaySubnetMask(rangeEnd(), allValues());
        setRangeEndError(err || '');
    };

    const validateSubnetMask = () => {
        const err = validateGatewaySubnetMask(undefined, allValues());
        setSubnetMaskError(err || '');
    };

    const onFormSubmit = (e: Event) => {
        e.preventDefault();
        validateGatewayIp();
        validateSubnetMask();
        validateRangeStart();
        validateRangeEnd();

        if (gatewayIpError() || subnetMaskError() || rangeStartError() || rangeEndError()) {
            return;
        }

        props.onSave({
            gateway_ip: gatewayIp().trim(),
            subnet_mask: subnetMask().trim(),
            range_start: rangeStart().trim(),
            range_end: rangeEnd().trim(),
            lease_duration: leaseDuration() ? Number(leaseDuration().trim()) : 0,
        });
    };

    return (
        <form onSubmit={onFormSubmit} class={s.form}>
            <div class={cn(theme.form.group, s.formGroup)}>
                <div class={s.formField}>
                    <div>
                        <Input
                            value={gatewayIp()}
                            onChange={(e: Event) => setGatewayIp((e.target as HTMLInputElement).value)}
                            onBlur={() => {
                                validateGatewayIp();
                                validateRangeStart();
                                validateRangeEnd();
                            }}
                            id="v4_gateway_ip"
                            label={intl.getMessage('dhcp_form_gateway_address')}
                            placeholder="192.168.1.1"
                            disabled={!hasIpv4()}
                            errorMessage={gatewayIpError()}
                        />
                    </div>
                </div>

                <div class={s.formField}>
                    <span class={cn(theme.text.t3, s.formFieldLabel)}>
                        {intl.getMessage('dhcp_form_range_title')}
                    </span>
                    <div class={s.rangeRow}>
                        <div>
                            <Input
                                value={rangeStart()}
                                onChange={(e: Event) => setRangeStart((e.target as HTMLInputElement).value)}
                                onBlur={() => {
                                    validateRangeStart();
                                    validateGatewayIp();
                                    validateRangeEnd();
                                }}
                                id="v4_range_start"
                                placeholder="192.168.1.2"
                                disabled={!hasIpv4()}
                                errorMessage={rangeStartError()}
                            />
                        </div>
                        <div>
                            <Input
                                value={rangeEnd()}
                                onChange={(e: Event) => setRangeEnd((e.target as HTMLInputElement).value)}
                                onBlur={() => {
                                    validateRangeEnd();
                                    validateGatewayIp();
                                }}
                                id="v4_range_end"
                                placeholder="192.168.1.254"
                                disabled={!hasIpv4()}
                                errorMessage={rangeEndError()}
                            />
                        </div>
                    </div>
                </div>

                <div class={s.formField}>
                    <div>
                        <Input
                            value={subnetMask()}
                            onChange={(e: Event) => setSubnetMask((e.target as HTMLInputElement).value)}
                            onBlur={() => {
                                validateSubnetMask();
                                validateRangeStart();
                                validateRangeEnd();
                            }}
                            id="v4_subnet_mask"
                            label={intl.getMessage('dhcp_form_subnet_input')}
                            placeholder="255.255.255.0"
                            disabled={!hasIpv4()}
                            errorMessage={subnetMaskError()}
                        />
                    </div>
                </div>

                <div class={s.formField}>
                    <div>
                        <Input
                            value={leaseDuration()}
                            onChange={(e: Event) => setLeaseDuration((e.target as HTMLInputElement).value)}
                            id="v4_lease_duration"
                            inputMode="numeric"
                            label={intl.getMessage('dhcp_form_lease_title')}
                            placeholder="86400"
                            disabled={!hasIpv4()}
                        />
                    </div>
                </div>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={props.processingConfig || !hasIpv4() || isEmptyConfig()}
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
