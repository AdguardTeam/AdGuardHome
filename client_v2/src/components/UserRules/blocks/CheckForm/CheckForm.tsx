import React from 'react';
import { Controller, UseFormHandleSubmit, Control } from 'react-hook-form';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import { Button } from 'panel/common/ui/Button';
import theme from 'panel/lib/theme';
import { CheckFormValues, DNS_RECORD_TYPE_OPTIONS } from '../../types';

import s from './CheckForm.module.pcss';

type Props = {
    control: Control<CheckFormValues>;
    handleSubmit: UseFormHandleSubmit<CheckFormValues>;
    onSubmit: (data: CheckFormValues) => void;
    isValid: boolean;
    processingCheck: boolean;
};

export const CheckForm = ({ control, handleSubmit, onSubmit, isValid, processingCheck }: Props) => (
    <form onSubmit={handleSubmit(onSubmit)}>
        <div className={s.formFields}>
            <Controller
                name="hostname"
                control={control}
                rules={{
                    required: intl.getMessage('form_error_required'),
                }}
                render={({ field, fieldState }) => (
                    <div className={s.formGroup}>
                        <Input
                            {...field}
                            id="user-rules-hostname"
                            data-testid="user-rules-check-hostname"
                            type="text"
                            size="medium"
                            label={intl.getMessage('user_rules_check_hostname_label')}
                            placeholder={intl.getMessage('user_rules_check_hostname_placeholder')}
                            errorMessage={fieldState.error?.message}
                            isClearable
                            onClear={() => field.onChange('')}
                        />
                    </div>
                )}
            />

            <Controller
                name="client"
                control={control}
                render={({ field, fieldState }) => (
                    <div className={s.formGroup}>
                        <Input
                            {...field}
                            id="user-rules-client"
                            data-testid="user-rules-check-client"
                            type="text"
                            size="medium"
                            label={intl.getMessage('user_rules_check_client_label')}
                            placeholder={intl.getMessage('user_rules_check_client_placeholder')}
                            errorMessage={fieldState.error?.message}
                            isClearable
                            onClear={() => field.onChange('')}
                        />
                    </div>
                )}
            />

            <Controller
                name="qtype"
                control={control}
                rules={{
                    required: intl.getMessage('form_error_required'),
                }}
                render={({ field, fieldState }) => (
                    <div className={s.formGroup}>
                        <div className={s.selectField} data-testid="user-rules-check-qtype">
                            <label
                                className={cn(s.selectLabel, theme.text.t3)}
                                htmlFor="user-rules-qtype-input"
                            >
                                {intl.getMessage('user_rules_check_dns_record_type_label')}
                            </label>

                            <Select
                                id="user-rules-qtype"
                                inputId="user-rules-qtype-input"
                                size="responsive"
                                height="medium"
                                menuSize="large"
                                placeholder={intl.getMessage(
                                    'user_rules_dns_record_type_placeholder',
                                )}
                                options={DNS_RECORD_TYPE_OPTIONS}
                                value={DNS_RECORD_TYPE_OPTIONS.find(
                                    (option) => option.value === field.value,
                                )}
                                onChange={(option) => field.onChange(option?.value || '')}
                                onBlur={field.onBlur}
                            />

                            {fieldState.error && (
                                <div className={theme.form.error}>{fieldState.error.message}</div>
                            )}
                        </div>
                    </div>
                )}
            />
        </div>

        <div className={s.checkActions}>
            <Button
                type="submit"
                variant="primary"
                size="small"
                disabled={!isValid || processingCheck}
                className={s.checkSubmitButton}
                data-testid="user-rules-check-submit"
            >
                {intl.getMessage('user_rules_check_button')}
            </Button>
        </div>
    </form>
);
