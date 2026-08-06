import React, { useMemo } from 'react';
import { Controller, useFormContext } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import { UINT32_RANGE } from '../../../helpers/constants';
import { validateIpv6, validateRequiredValue } from '../../../helpers/validators';
import { DhcpFormValues } from '.';
import { Input } from '../../ui/Controls/Input';
import { toNumber } from '../../../helpers/form';

type FormDHCPv6Props = {
    processingConfig?: boolean;
    ipv6placeholders?: {
        range_start: string;
        range_end: string;
        lease_duration: string;
    };
    interfaces: any;
    onSubmit?: (data: DhcpFormValues) => Promise<void> | void;
};

const FormDHCPv6 = ({ processingConfig, ipv6placeholders, interfaces, onSubmit }: FormDHCPv6Props) => {
    const { t } = useTranslation();
    const {
        handleSubmit,
        formState: { isSubmitting, isValid },
        control,
        watch,
    } = useFormContext<DhcpFormValues>();

    const interfaceName = watch('interface_name');
    const isInterfaceIncludesIpv6 = interfaces?.[interfaceName]?.ipv6_addresses;

    const formValues = watch('v6');
    const isEmptyConfig = !Object.values(formValues || {}).some(Boolean);

    const isDisabled = useMemo(() => {
        return isSubmitting || !isValid || processingConfig || !isInterfaceIncludesIpv6 || isEmptyConfig;
    }, [isSubmitting, isValid, processingConfig, isInterfaceIncludesIpv6, isEmptyConfig]);

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group mb-0">
                        <div className="row">
                            <div className="col-12">
                                <label>{t('dhcp_form_range_title')}</label>
                            </div>

                            <div className="col">
                                <Controller
                                    name="v6.range_start"
                                    control={control}
                                    rules={{
                                        validate: isInterfaceIncludesIpv6
                                            ? {
                                                  ipv6: validateIpv6,
                                                  required: validateRequiredValue,
                                              }
                                            : undefined,
                                    }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="v6_range_start"
                                            placeholder={t(ipv6placeholders.range_start)}
                                            error={fieldState.error?.message}
                                            disabled={!isInterfaceIncludesIpv6}
                                        />
                                    )}
                                />
                            </div>

                            <div className="col">
                                <Controller
                                    name="v6.range_end"
                                    control={control}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="v6_range_end"
                                            placeholder={t(ipv6placeholders.range_end)}
                                            error={fieldState.error?.message}
                                            disabled
                                        />
                                    )}
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row">
                <div className="col-lg-6 form__group form__group--settings">
                    <Controller
                        name="v6.lease_duration"
                        control={control}
                        rules={{
                            validate: isInterfaceIncludesIpv6
                                ? {
                                      required: validateRequiredValue,
                                  }
                                : undefined,
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="number"
                                data-testid="v6_lease_duration"
                                label={t('dhcp_form_lease_title')}
                                placeholder={t(ipv6placeholders.lease_duration)}
                                error={fieldState.error?.message}
                                disabled={!isInterfaceIncludesIpv6}
                                min={1}
                                max={UINT32_RANGE.MAX}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>
            </div>

            <div className="btn-list">
                <button
                    data-testid="v6_save"
                    type="submit"
                    className="btn btn-success btn-standard"
                    disabled={isDisabled}>
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default FormDHCPv6;
