import React, { useEffect, useMemo } from 'react';
import { Controller, useForm } from 'react-hook-form';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import {
    validateRequiredValue,
    validateIp,
    validateMac,
    validateHostname,
    validateIpNotDuplicate,
    validateMacNotDuplicate,
    validateIpv4InCidr,
    validateIpGateway,
} from 'panel/helpers/validators';
import { parseSubnetMask } from 'panel/helpers/helpers';

type LeaseData = {
    mac: string;
    ip: string;
    hostname: string;
};

type Props = {
    isOpen: boolean;
    isEdit: boolean;
    isMakeStatic?: boolean;
    initialData?: LeaseData;
    processingAdding: boolean;
    processingUpdating: boolean;
    staticLeases: LeaseData[];
    dhcpConfig?: { gatewayIp: string; subnetMask: string };
    onSubmit: (data: LeaseData) => void;
    onClose: () => void;
};

export const StaticLeaseModal = ({
    isOpen,
    isEdit,
    isMakeStatic,
    initialData,
    processingAdding,
    processingUpdating,
    staticLeases,
    dhcpConfig,
    onSubmit,
    onClose,
}: Props) => {
    const {
        handleSubmit,
        control,
        reset,
        formState: { isValid },
    } = useForm<LeaseData>({
        mode: 'onChange',
        defaultValues: {
            mac: initialData?.mac || '',
            ip: initialData?.ip || '',
            hostname: initialData?.hostname || '',
        },
    });

    useEffect(() => {
        reset({
            mac: initialData?.mac || '',
            ip: initialData?.ip || '',
            hostname: initialData?.hostname || '',
        });
    }, [initialData, reset]);

    const isProcessing = processingAdding || processingUpdating;

    const cidr = useMemo(() => {
        if (!dhcpConfig?.gatewayIp || !dhcpConfig?.subnetMask) {
            return undefined;
        }
        const prefix = parseSubnetMask(dhcpConfig.subnetMask);
        if (prefix === null) {
            return undefined;
        }
        return `${dhcpConfig.gatewayIp}/${prefix}`;
    }, [dhcpConfig]);

    const getTitle = () => {
        if (isMakeStatic) {
            return intl.getMessage('make_static');
        }
        if (isEdit) {
            return intl.getMessage('dhcp_edit_static_lease');
        }
        return intl.getMessage('dhcp_new_static_lease');
    };

    const submitLabel = isMakeStatic ? intl.getMessage('make_static') : intl.getMessage('save');

    return (
        <Dialog
            visible={isOpen}
            title={getTitle()}
            onClose={onClose}
            wrapClassName="rc-dialog-update"
        >
            <form onSubmit={handleSubmit(onSubmit)}>
                {isMakeStatic && (
                    <div className={theme.dialog.body}>{intl.getMessage('make_static_desc')}</div>
                )}
                <div className={theme.form.group}>
                    <div className={theme.form.input}>
                        <Controller
                            name="mac"
                            control={control}
                            rules={{
                                validate: {
                                    validateRequiredValue,
                                    validateMac,
                                    validateMacNotDuplicate: validateMacNotDuplicate(
                                        staticLeases,
                                        isEdit ? initialData?.mac : undefined,
                                    ),
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    id="static_lease_mac"
                                    label={intl.getMessage('dhcp_table_mac_address')}
                                    placeholder={intl.getMessage('form_enter_mac')}
                                    errorMessage={fieldState.error?.message}
                                    disabled={isEdit || isMakeStatic}
                                />
                            )}
                        />
                    </div>

                    <div className={theme.form.input}>
                        <Controller
                            name="hostname"
                            control={control}
                            rules={{
                                validate: {
                                    validateHostname,
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    id="static_lease_hostname"
                                    label={intl.getMessage('dhcp_table_hostname')}
                                    placeholder={intl.getMessage('form_enter_hostname')}
                                    errorMessage={fieldState.error?.message}
                                    disabled={isMakeStatic}
                                />
                            )}
                        />
                    </div>

                    <div className={theme.form.input}>
                        <Controller
                            name="ip"
                            control={control}
                            rules={{
                                validate: {
                                    validateRequiredValue,
                                    validateIp,
                                    validateIpNotDuplicate: validateIpNotDuplicate(
                                        staticLeases,
                                        isEdit ? initialData?.ip : undefined,
                                    ),
                                    validateIpv4InCidr: (value: string) => {
                                        if (!cidr || !value) {
                                            return undefined;
                                        }
                                        return validateIpv4InCidr(value, { cidr });
                                    },
                                    validateIpGateway: (value: string) => {
                                        if (!dhcpConfig?.gatewayIp || !value) {
                                            return undefined;
                                        }
                                        return validateIpGateway(value, {
                                            gatewayIp: dhcpConfig.gatewayIp,
                                        });
                                    },
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    id="static_lease_ip"
                                    label={intl.getMessage('dhcp_table_ip_address')}
                                    placeholder={intl.getMessage('form_enter_ip')}
                                    errorMessage={fieldState.error?.message}
                                />
                            )}
                        />
                    </div>
                </div>
                <div className={theme.dialog.footer}>
                    <Button
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={isProcessing || !isValid}
                        className={theme.dialog.button}
                    >
                        {submitLabel}
                    </Button>
                    <Button
                        variant="secondary"
                        size="small"
                        onClick={onClose}
                        disabled={isProcessing}
                        className={theme.dialog.button}
                    >
                        {intl.getMessage('cancel')}
                    </Button>
                </div>
            </form>
        </Dialog>
    );
};
