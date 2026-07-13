import { createSignal, createEffect, createMemo } from 'solid-js';
import type { Accessor } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { dhcpState } from 'panel/stores/dhcp';
import {
    validateIpv4,
    validateIpv4RangeEnd,
    validateNotInRange,
    validateGatewaySubnetMask,
    validateIpForGatewaySubnetMask,
    validateLeaseTime,
} from 'panel/helpers/validators';
import s from './DhcpV4Modal.module.pcss';

export type V4Config = {
    gateway_ip: string;
    subnet_mask: string;
    range_start: string;
    range_end: string;
    lease_duration: number;
};

type Props = {
    open: boolean;
    selectedInterface: Accessor<string>;
    onClose: () => void;
    onSave: (values: V4Config) => void;
};

export const DhcpV4Modal = (props: Props) => {
    const [gatewayIp, setGatewayIp] = createSignal('');
    const [subnetMask, setSubnetMask] = createSignal('');
    const [rangeStart, setRangeStart] = createSignal('');
    const [rangeEnd, setRangeEnd] = createSignal('');
    const [leaseDuration, setLeaseDuration] = createSignal('');
    const [gatewayIpError, setGatewayIpError] = createSignal('');
    const [subnetMaskError, setSubnetMaskError] = createSignal('');
    const [rangeStartError, setRangeStartError] = createSignal('');
    const [rangeEndError, setRangeEndError] = createSignal('');
    const [leaseDurationError, setLeaseDurationError] = createSignal('');

    createEffect(() => {
        if (props.open) {
            const v4 = dhcpState.v4;
            setGatewayIp(v4?.gateway_ip || '');
            setSubnetMask(v4?.subnet_mask || '');
            setRangeStart(v4?.range_start || '');
            setRangeEnd(v4?.range_end || '');
            setLeaseDuration(v4?.lease_duration ? String(v4.lease_duration) : '');
            setGatewayIpError('');
            setSubnetMaskError('');
            setRangeStartError('');
            setRangeEndError('');
            setLeaseDurationError('');
        }
    });

    const hasIpv4 = createMemo(
        () =>
            !!(
                dhcpState.interfaces &&
                dhcpState.interfaces[props.selectedInterface()]?.ipv4_addresses
            ),
    );

    const allValues = createMemo(() => ({
        v4: {
            gateway_ip: gatewayIp(),
            subnet_mask: subnetMask(),
            range_start: rangeStart(),
            range_end: rangeEnd(),
        },
    }));

    const isEmptyConfig = createMemo(
        () => !gatewayIp() && !subnetMask() && !rangeStart() && !rangeEnd() && !leaseDuration(),
    );

    const validateGatewayIp = () => {
        const err = validateIpv4(gatewayIp()) || validateNotInRange(gatewayIp(), allValues());
        setGatewayIpError(err || '');
    };
    const validateSubnetMask = () => {
        const err = validateGatewaySubnetMask(undefined, allValues());
        setSubnetMaskError(err || '');
    };
    const validateRangeStart = () => {
        const err =
            validateIpv4(rangeStart()) || validateIpForGatewaySubnetMask(rangeStart(), allValues());
        setRangeStartError(err || '');
    };
    const validateRangeEnd = () => {
        const err =
            validateIpv4(rangeEnd()) ||
            validateIpv4RangeEnd(undefined, allValues()) ||
            validateIpForGatewaySubnetMask(rangeEnd(), allValues());
        setRangeEndError(err || '');
    };

    const onGatewayBlur = () => {
        validateGatewayIp();
        validateRangeStart();
        validateRangeEnd();
    };
    const onRangeStartBlur = () => {
        validateRangeStart();
        validateGatewayIp();
        validateRangeEnd();
    };
    const onRangeEndBlur = () => {
        validateRangeEnd();
        validateGatewayIp();
    };
    const onSubnetBlur = () => {
        validateSubnetMask();
        validateRangeStart();
        validateRangeEnd();
    };

    const validateLeaseDuration = () => {
        const err = validateLeaseTime(leaseDuration());
        setLeaseDurationError(err || '');
    };

    const handleSave = () => {
        validateGatewayIp();
        validateSubnetMask();
        validateRangeStart();
        validateRangeEnd();
        validateLeaseDuration();
        if (
            gatewayIpError() ||
            subnetMaskError() ||
            rangeStartError() ||
            rangeEndError() ||
            leaseDurationError()
        ) {
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
        <ConfigDialog
            open={props.open}
            title={intl.getMessage('dhcp_ipv4_settings')}
            onClose={props.onClose}
            onSubmit={handleSave}
            processing={!!dhcpState.processingConfig}
            submitDisabled={!hasIpv4() || isEmptyConfig()}
        >
            <div class={theme.form.input}>
                <Input
                    value={gatewayIp()}
                    onChange={(e: Event) => setGatewayIp((e.target as HTMLInputElement).value)}
                    onBlur={onGatewayBlur}
                    id="v4_gateway_ip"
                    label={intl.getMessage('dhcp_form_gateway_address')}
                    placeholder="192.168.1.1"
                    disabled={!hasIpv4()}
                    errorMessage={gatewayIpError()}
                    size="large"
                />
            </div>
            <div class={s.formField}>
                <span class={cn(theme.text.t3, s.formFieldLabel)}>
                    {intl.getMessage('dhcp_form_range_title')}
                </span>
                <div class={s.rangeRow}>
                    <div>
                        <Input
                            value={rangeStart()}
                            onChange={(e: Event) =>
                                setRangeStart((e.target as HTMLInputElement).value)
                            }
                            onBlur={onRangeStartBlur}
                            id="v4_range_start"
                            placeholder="192.168.1.2"
                            disabled={!hasIpv4()}
                            errorMessage={rangeStartError()}
                            size="large"
                        />
                    </div>
                    <div>
                        <Input
                            value={rangeEnd()}
                            onChange={(e: Event) =>
                                setRangeEnd((e.target as HTMLInputElement).value)
                            }
                            onBlur={onRangeEndBlur}
                            id="v4_range_end"
                            placeholder="192.168.1.254"
                            disabled={!hasIpv4()}
                            errorMessage={rangeEndError()}
                            size="large"
                        />
                    </div>
                </div>
            </div>
            <div class={theme.form.input}>
                <Input
                    value={subnetMask()}
                    onChange={(e: Event) => setSubnetMask((e.target as HTMLInputElement).value)}
                    onBlur={onSubnetBlur}
                    id="v4_subnet_mask"
                    label={intl.getMessage('dhcp_form_subnet_input')}
                    placeholder="255.255.255.0"
                    disabled={!hasIpv4()}
                    errorMessage={subnetMaskError()}
                    size="large"
                />
            </div>
            <div class={theme.form.input}>
                <Input
                    value={leaseDuration()}
                    onChange={(e: Event) => setLeaseDuration((e.target as HTMLInputElement).value)}
                    onBlur={() => validateLeaseDuration()}
                    id="v4_lease_duration"
                    inputMode="numeric"
                    label={intl.getMessage('dhcp_form_lease_title')}
                    placeholder="86400"
                    disabled={!hasIpv4()}
                    size="large"
                    inputError={leaseDurationError()}
                />
            </div>
        </ConfigDialog>
    );
};
