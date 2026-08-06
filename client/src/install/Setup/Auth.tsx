import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import Controls from './Controls';
import { validatePasswordLength, validateRequiredValue } from '../../helpers/validators';
import { Input } from '../../components/ui/Controls/Input';

type AuthFormValues = {
    username: string;
    password: string;
    confirm_password: string;
};

type Props = {
    onAuthSubmit: (values: AuthFormValues) => void;
};

export const Auth = ({ onAuthSubmit }: Props) => {
    const { t } = useTranslation();
    const {
        handleSubmit,
        watch,
        control,
        formState: { isDirty, isValid },
    } = useForm<AuthFormValues>({
        mode: 'onBlur',
        defaultValues: {
            username: '',
            password: '',
            confirm_password: '',
        },
    });

    const password = watch('password');

    const validateConfirmPassword = (value: string) => {
        if (value !== password) {
            return t('form_error_password');
        }
        return undefined;
    };

    return (
        <form className="setup__step" onSubmit={handleSubmit(onAuthSubmit)}>
            <div className="setup__group">
                <div className="setup__subtitle">
                    <Trans>install_auth_title</Trans>
                </div>

                <p className="setup__desc">
                    <Trans>install_auth_desc</Trans>
                </p>

                <div className="form-group">
                    <Controller
                        name="username"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="text"
                                data-testid="install_username"
                                label={t('install_auth_username')}
                                placeholder={t('install_auth_username_enter')}
                                error={fieldState.error?.message}
                                autoComplete="username"
                            />
                        )}
                    />
                </div>

                <div className="form-group">
                    <Controller
                        name="password"
                        control={control}
                        rules={{
                            validate: {
                                required: validateRequiredValue,
                                passwordLength: validatePasswordLength,
                            },
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="password"
                                data-testid="install_password"
                                label={t('install_auth_password')}
                                placeholder={t('install_auth_password_enter')}
                                error={fieldState.error?.message}
                                autoComplete="new-password"
                            />
                        )}
                    />
                </div>

                <div className="form-group">
                    <Controller
                        name="confirm_password"
                        control={control}
                        rules={{
                            validate: {
                                required: validateRequiredValue,
                                confirmPassword: validateConfirmPassword,
                            },
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="password"
                                data-testid="install_confirm_password"
                                label={t('install_auth_confirm')}
                                placeholder={t('install_auth_confirm')}
                                error={fieldState.error?.message}
                                autoComplete="new-password"
                            />
                        )}
                    />
                </div>
            </div>

            <Controls isDirty={isDirty} isValid={isValid} />
        </form>
    );
};
