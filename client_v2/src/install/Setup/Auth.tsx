import React, { useMemo } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Input } from 'panel/common/controls/Input';
import intl from 'panel/common/intl';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { PRIVACY_POLICY_LINK, TERMS_LINK } from 'panel/helpers/constants';
import cn from 'clsx';
import { PasswordRequirements } from 'panel/install/Setup/blocks';
import { PasswordInput } from '../../common/controls/Input/PasswordInput';
import Controls from './Controls';
import { validatePasswordLength, validateRequiredValue } from '../../helpers/validators';
import { hasMinLength, hasLowercase, hasUppercase, hasAllowedAsciiOnly, hasNumberOrSpecial } from './helpers/helpers';
import styles from './styles.module.pcss';

type AuthFormValues = {
    username: string;
    password: string;
    confirm_password: string;
    privacy_consent: boolean;
};

type Props = {
    onAuthSubmit: (values: AuthFormValues) => void;
};

export const Auth = ({ onAuthSubmit }: Props) => {
    const {
        handleSubmit,
        watch,
        control,
        formState: { isDirty, isValid },
    } = useForm<AuthFormValues>({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
            confirm_password: '',
            privacy_consent: false,
        },
    });

    const password = watch('password') ?? '';
    const confirmPassword = watch('confirm_password') ?? '';

    const validateConfirmPassword = (value: string) => {
        if (value !== password) {
            return intl.getMessage('form_error_password');
        }
        return undefined;
    };

    const requirements = useMemo(() => {
        return {
            minLength: password.length > 0 && hasMinLength(password),
            allowedChars: password.length > 0 && hasAllowedAsciiOnly(password) && hasNumberOrSpecial(password),
            lowercase: password.length > 0 && hasLowercase(password),
            uppercase: password.length > 0 && hasUppercase(password),
            match:
                password.length > 0 &&
                confirmPassword.length > 0 &&
                password === confirmPassword,
        };
    }, [password, confirmPassword]);

    return (
        <div className={styles.configSetting}>
            <form className={styles.step} onSubmit={handleSubmit(onAuthSubmit)}>
                <div className={styles.info}>
                    <div className={styles.titleStep}>{intl.getMessage('setup_guide_auth_title')}</div>

                    <p className={styles.descStep}>{intl.getMessage('setup_guide_auth_subtitle')}</p>

                    <div className={styles.input}>
                        <Controller
                            name="username"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    id="install_username"
                                    label={intl.getMessage('install_auth_username')}
                                    placeholder={intl.getMessage('install_auth_username_enter')}
                                    autoComplete="username"
                                />
                            )}
                        />
                    </div>

                    <div className={styles.input}>
                        <Controller
                            name="password"
                            control={control}
                            rules={{
                                validate: {
                                    required: validateRequiredValue,
                                    passwordLength: validatePasswordLength,
                                },
                            }}
                            render={({ field }) => (
                                <PasswordInput
                                    id="install_password"
                                    label={intl.getMessage('install_auth_password')}
                                    placeholder={intl.getMessage('install_auth_password_enter')}
                                    autoComplete="new-password"
                                    value={field.value ?? ''}
                                    onChange={(value) => field.onChange(value)}
                                    name={field.name}
                                    onBlur={field.onBlur}
                                    ref={field.ref}
                                />
                            )}
                        />
                    </div>

                    <PasswordRequirements requirements={requirements} className={styles.authRequirementsMobile} />

                    <div className={styles.input}>
                        <Controller
                            name="confirm_password"
                            control={control}
                            rules={{
                                validate: {
                                    required: validateRequiredValue,
                                    confirmPassword: validateConfirmPassword,
                                },
                            }}
                            render={({ field }) => (
                                <PasswordInput
                                    id="install_confirm_password"
                                    label={intl.getMessage('install_auth_confirm')}
                                    placeholder={intl.getMessage('install_auth_confirm')}
                                    autoComplete="new-password"
                                    value={field.value ?? ''}
                                    onChange={(value) => field.onChange(value)}
                                    name={field.name}
                                    onBlur={field.onBlur}
                                    ref={field.ref}
                                />
                            )}
                        />
                    </div>

                    <div className={styles.consent}>
                        <Controller
                            name="privacy_consent"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field }) => (
                                <Checkbox
                                    checked={field.value}
                                    onChange={(e) => field.onChange(e.target.checked)}
                                    name={field.name}
                                    onBlur={field.onBlur}
                                    verticalAlign="start"
                                >
                                    <div className={styles.consentContent}>
                                        {intl.getMessage('setup_guide_auth_privacy', {
                                            a: (text: string) =>
                                                <a className={styles.link} href={PRIVACY_POLICY_LINK}>
                                                    {text}
                                                </a>,
                                            b: (text: string) =>
                                                <a className={styles.link} href={TERMS_LINK}>
                                                    {text}
                                                </a>,
                                        })}
                                    </div>
                                </Checkbox>
                            )}
                        />
                    </div>

                    <Controls isDirty={isDirty} isValid={isValid} />
                </div>

                <div className={styles.content}>
                    <PasswordRequirements
                        requirements={requirements}
                        className={cn(styles.banner, styles.authBanner)}
                    />
                </div>
            </form>
        </div>
    );
};
