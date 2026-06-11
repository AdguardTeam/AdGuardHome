import React, { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { DhcpInterfaces } from 'panel/initialState';
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

type FormValues = {
    gateway_ip: string;
    subnet_mask: string;
    range_start: string;
    range_end: string;
    lease_duration: string;
};

type Props = {
    v4?: V4Config;
    interfaces?: DhcpInterfaces;
    selectedInterface: string;
    processingConfig: boolean;
    onSave: (values: V4Config) => void;
};

export const Ipv4Settings = ({
    v4,
    interfaces,
    selectedInterface,
    processingConfig,
    onSave,
}: Props) => {
    const hasIpv4 = !!(interfaces && interfaces[selectedInterface]?.ipv4_addresses);

    const { handleSubmit, control, reset, trigger, watch } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: {
            gateway_ip: v4?.gateway_ip || '',
            subnet_mask: v4?.subnet_mask || '',
            range_start: v4?.range_start || '',
            range_end: v4?.range_end || '',
            lease_duration: v4?.lease_duration ? String(v4.lease_duration) : '',
        },
    });

    const formValues = watch();
    const isEmptyConfig =
        !formValues.gateway_ip &&
        !formValues.subnet_mask &&
        !formValues.range_start &&
        !formValues.range_end &&
        !formValues.lease_duration;

    useEffect(() => {
        reset({
            gateway_ip: v4?.gateway_ip || '',
            subnet_mask: v4?.subnet_mask || '',
            range_start: v4?.range_start || '',
            range_end: v4?.range_end || '',
            lease_duration: v4?.lease_duration ? String(v4.lease_duration) : '',
        });
    }, [v4, reset]);

    const onFormSubmit = (values: FormValues) => {
        onSave({
            gateway_ip: values.gateway_ip.trim(),
            subnet_mask: values.subnet_mask.trim(),
            range_start: values.range_start.trim(),
            range_end: values.range_end.trim(),
            lease_duration: values.lease_duration ? Number(values.lease_duration.trim()) : 0,
        });
    };

    return (
        <form onSubmit={handleSubmit(onFormSubmit)} className={s.form}>
            <div className={cn(theme.form.group, s.formGroup)}>
                <div className={s.formField}>
                    <div>
                        <Controller
                            name="gateway_ip"
                            control={control}
                            rules={{
                                validate: {
                                    validateIpv4: (value: string) => validateIpv4(value),
                                    validateNotInRange: (value: string, allValues: FormValues) =>
                                        validateNotInRange(value, { v4: allValues }),
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    id="v4_gateway_ip"
                                    label={intl.getMessage('dhcp_form_gateway_address')}
                                    placeholder="192.168.1.1"
                                    disabled={!hasIpv4}
                                    errorMessage={fieldState.error?.message}
                                    onBlur={(_e) => {
                                        field.onBlur();
                                        trigger('range_start');
                                        trigger('range_end');
                                    }}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className={s.formField}>
                    <span className={cn(theme.text.t3, s.formFieldLabel)}>
                        {intl.getMessage('dhcp_form_range_title')}
                    </span>
                    <div className={s.rangeRow}>
                        <div>
                            <Controller
                                name="range_start"
                                control={control}
                                rules={{
                                    validate: {
                                        validateIpv4: (value: string) => validateIpv4(value),
                                        validateIpForGatewaySubnetMask: (
                                            value: string,
                                            allValues: FormValues,
                                        ) =>
                                            validateIpForGatewaySubnetMask(value, {
                                                v4: allValues,
                                            }),
                                    },
                                }}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        id="v4_range_start"
                                        placeholder="192.168.1.2"
                                        disabled={!hasIpv4}
                                        errorMessage={fieldState.error?.message}
                                        onBlur={(_e) => {
                                            field.onBlur();
                                            trigger('gateway_ip');
                                            trigger('range_end');
                                        }}
                                    />
                                )}
                            />
                        </div>
                        <div>
                            <Controller
                                name="range_end"
                                control={control}
                                rules={{
                                    validate: {
                                        validateIpv4: (value: string) => validateIpv4(value),
                                        validateIpv4RangeEnd: (_: string, allValues: FormValues) =>
                                            validateIpv4RangeEnd(undefined, { v4: allValues }),
                                        validateIpForGatewaySubnetMask: (
                                            value: string,
                                            allValues: FormValues,
                                        ) =>
                                            validateIpForGatewaySubnetMask(value, {
                                                v4: allValues,
                                            }),
                                    },
                                }}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        id="v4_range_end"
                                        placeholder="192.168.1.254"
                                        disabled={!hasIpv4}
                                        errorMessage={fieldState.error?.message}
                                        onBlur={(_e) => {
                                            field.onBlur();
                                            trigger('gateway_ip');
                                        }}
                                    />
                                )}
                            />
                        </div>
                    </div>
                </div>

                <div className={s.formField}>
                    <div>
                        <Controller
                            name="subnet_mask"
                            control={control}
                            rules={{
                                validate: {
                                    validateGatewaySubnetMask: (_: string, allValues: FormValues) =>
                                        validateGatewaySubnetMask(undefined, { v4: allValues }),
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    id="v4_subnet_mask"
                                    label={intl.getMessage('dhcp_form_subnet_input')}
                                    placeholder="255.255.255.0"
                                    disabled={!hasIpv4}
                                    errorMessage={fieldState.error?.message}
                                    onBlur={(_e) => {
                                        field.onBlur();
                                        trigger('range_start');
                                        trigger('range_end');
                                    }}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className={s.formField}>
                    <div>
                        <Controller
                            name="lease_duration"
                            control={control}
                            render={({ field }) => (
                                <Input
                                    {...field}
                                    id="v4_lease_duration"
                                    inputMode="numeric"
                                    label={intl.getMessage('dhcp_form_lease_title')}
                                    placeholder="86400"
                                    disabled={!hasIpv4}
                                />
                            )}
                        />
                    </div>
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
