import React from 'react';
import { useForm, Controller } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch, useSelector, shallowEqual } from 'react-redux';

import { normalizeMac } from '../../../../helpers/form';
import {
    validateIpv4,
    validateMac,
    validateRequiredValue,
    validateIpv4InCidr,
    validateIpGateway,
} from '../../../../helpers/validators';

import { toggleLeaseModal } from '../../../../actions';
import { RootState } from '../../../../initialState';
import { Input } from '../../../ui/Controls/Input';

type Props = {
    initialValues?: {
        mac?: string;
        ip?: string;
        hostname?: string;
        cidr?: string;
        gatewayIp?: string;
    };
    processingAdding?: boolean;
    cidr?: string;
    isEdit?: boolean;
    onSubmit: (data: any) => void;
};

export const Form = ({ initialValues, processingAdding, cidr, isEdit, onSubmit }: Props) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const dynamicLease = useSelector((store: RootState) => store.dhcp.leaseModalConfig, shallowEqual);

    const {
        handleSubmit,
        control,
        reset,
        formState: { isSubmitting, isDirty },
    } = useForm({
        defaultValues: initialValues,
        mode: 'onBlur',
    });

    const onClick = () => {
        reset();
        dispatch(toggleLeaseModal());
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="modal-body">
                <div className="form__group">
                    <Controller
                        name="mac"
                        control={control}
                        rules={{ validate: { required: validateRequiredValue, mac: validateMac } }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="text"
                                data-testid="static_lease_mac"
                                placeholder={t('form_enter_mac')}
                                disabled={isEdit}
                                error={fieldState.error?.message}
                                onChange={(e) => field.onChange(normalizeMac(e.target.value))}
                            />
                        )}
                    />
                </div>

                <div className="form__group">
                    <Controller
                        name="ip"
                        control={control}
                        rules={{
                            validate: {
                                required: validateRequiredValue,
                                ipv4: validateIpv4,
                                inCidr: validateIpv4InCidr,
                                gateway: validateIpGateway,
                            },
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="text"
                                data-testid="static_lease_ip"
                                error={fieldState.error?.message}
                                placeholder={t('form_enter_subnet_ip', { cidr })}
                            />
                        )}
                    />
                </div>

                <div className="form__group">
                    <Controller
                        name="hostname"
                        control={control}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="text"
                                data-testid="static_lease_hostname"
                                error={fieldState.error?.message}
                                placeholder={t('form_enter_hostname')}
                            />
                        )}
                    />
                </div>
            </div>

            <div className="modal-footer">
                <div className="btn-list">
                    <button
                        type="button"
                        data-testid="static_lease_cancel"
                        className="btn btn-secondary btn-standard"
                        disabled={isSubmitting}
                        onClick={onClick}>
                        <Trans>cancel_btn</Trans>
                    </button>

                    <button
                        type="submit"
                        data-testid="static_lease_save"
                        className="btn btn-success btn-standard"
                        disabled={isSubmitting || processingAdding || (!isDirty && !dynamicLease)}>
                        <Trans>save_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};
