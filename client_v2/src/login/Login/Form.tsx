import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { validateRequiredValue } from 'panel/helpers/validators';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';

export type LoginFormValues = {
    username: string;
    password: string;
};

type LoginFormProps = {
    onSubmit: (data: LoginFormValues) => void;
    processing: boolean;
};

const Form = ({ onSubmit, processing }: LoginFormProps) => {
    const { t } = useTranslation();
    const {
        handleSubmit,
        control,
        formState: { isValid },
    } = useForm<LoginFormValues>({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
        },
    });

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="card">
            <div className="card-body p-6">
                <div className="form__group form__group--settings">
                    <Controller
                        name="username"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                id="username"
                                data-testid="username"
                                type="text"
                                label={t('username_label')}
                                placeholder={t('username_placeholder')}
                                errorMessage={fieldState.error?.message}
                                autoComplete="username"
                                autoCapitalize="none"
                            />
                        )}
                    />
                </div>

                <div className="form__group form__group--settings">
                    <Controller
                        name="password"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                id="password"
                                data-testid="password"
                                type="password"
                                label={t('password_label')}
                                placeholder={t('password_placeholder')}
                                errorMessage={fieldState.error?.message}
                                autoComplete="current-password"
                            />
                        )}
                    />
                </div>

                <div className="form-footer">
                    <Button
                        id="sign_in"
                        data-testid="sign_in"
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={processing || !isValid}>
                        {t('sign_in')}
                    </Button>
                </div>
            </div>
        </form>
    );
};

export default Form;
