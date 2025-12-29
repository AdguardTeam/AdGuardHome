import React, { useMemo } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { Input } from 'panel/common/controls/Input';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Checkbox } from 'panel/common/controls/Checkbox';
import Controls from './Controls';
import { validatePasswordLength, validateRequiredValue } from '../../helpers/validators';


type AuthFormValues = {
    username: string;
    password: string;
    confirm_password: string;
    privacy_consent: boolean;
};

type Props = {
    onAuthSubmit: (values: AuthFormValues) => void;
};

const hasMinLength = (v: string) => v.length >= 8;
const hasLowercase = (v: string) => /[a-z]/.test(v);
const hasUppercase = (v: string) => /[A-Z]/.test(v);
const hasAllowedAsciiOnly = (v: string) => /^[\x20-\x7E]*$/.test(v);
const hasNumberOrSpecial = (v: string) => /[\d\W_]/.test(v);
type RequirementIconProps = {
    ok: boolean;
};

const RequirementIcon = ({ ok }: RequirementIconProps) => {
    const iconName = ok ? 'check' : 'cross';
    const iconClass = ok ? 'icon-green' : 'icon-red'

    return <Icon icon={iconName} className={iconClass}/>;
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
            privacy_consent: false,
        },
    });

    const password = watch('password') ?? '';
    const confirmPassword = watch('confirm_password') ?? '';

    const validateConfirmPassword = (value: string) => {
        if (value !== password) {
            return t('form_error_password');
        }
        return undefined;
    };

    const requirements = useMemo(() => {
        const pwd = password;

        return {
            minLength: pwd.length > 0 && hasMinLength(pwd),
            allowedChars: pwd.length > 0 && hasAllowedAsciiOnly(pwd) && hasNumberOrSpecial(pwd),
            lowercase: pwd.length > 0 && hasLowercase(pwd),
            uppercase: pwd.length > 0 && hasUppercase(pwd),
            match:
                pwd.length > 0 &&
                confirmPassword.length > 0 &&
                pwd === confirmPassword,
        };
    }, [password, confirmPassword]);

    return (
        <div className="setup__config-setting">
            <form className="setup__step" onSubmit={handleSubmit(onAuthSubmit)}>
                <div className="setup__left-side">
                    <div className="setup__title">{intl.getMessage('setup_guide_auth_title')}</div>

                    <p className="setup__desc">{intl.getMessage('setup_guide_auth_subtitle')}</p>

                    <div className="form-group">
                        <Controller
                            name="username"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    id="install_username"
                                    label={intl.getMessage('install_auth_username')}
                                    placeholder={intl.getMessage('install_auth_username_enter')}
                                    errorMessage={fieldState.error?.message}
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
                                    id="install_password"
                                    label={intl.getMessage('install_auth_password')}
                                    placeholder={intl.getMessage('install_auth_password_enter')}
                                    errorMessage={fieldState.error?.message}
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
                                    id="install_confirm_password"
                                    label={intl.getMessage('install_auth_confirm')}
                                    placeholder={intl.getMessage('install_auth_confirm')}
                                    errorMessage={fieldState.error?.message}
                                    autoComplete="new-password"
                                />
                            )}
                        />
                    </div>

                    <div className="setup__consent">
                        <Controller
                            name="privacy_consent"
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    checked={field.value}
                                    onChange={(e) => field.onChange(e.target.checked)}
                                    name={field.name}
                                    onBlur={field.onBlur}
                                >
                                    <div>
                                        {intl.getMessage('setup_guide_auth_privacy', {
                                            a: (text: string) => <a className="setup__link" href="https://adguard.com/privacy-policy">{text}</a>,
                                            b: (text: string) => <a className="setup__link" href="https://adguard.com/privacy-policy">{text}</a>,
                                        })}
                                    </div>
                                </Checkbox>
                            )}
                        />
                    </div>

                    <Controls isDirty={isDirty} isValid={isValid} />
                </div>

                <div className="setup__right-side">
                    <div className="setup__banner">
                        <h3 className="setup__banner-title">{intl.getMessage('password_requirements')}</h3>

                        <ul className="setup__banner-list">
                            <li className="setup__banner-item">
                                <RequirementIcon ok={requirements.minLength} />
                                {intl.getMessage('password_requirements_characters')}
                            </li>

                            <li className="setup__banner-item">
                                <RequirementIcon ok={requirements.allowedChars} />
                                {intl.getMessage('password_requirements_special')}
                            </li>

                            <li className="setup__banner-item">
                                <RequirementIcon ok={requirements.lowercase} />
                                {intl.getMessage('password_requirements_lowercase')}
                            </li>

                            <li className="setup__banner-item">
                                <RequirementIcon ok={requirements.uppercase} />
                                {intl.getMessage('password_requirements_uppercase')}
                            </li>

                            <li className="setup__banner-item">
                                <RequirementIcon ok={requirements.match} />
                                {intl.getMessage('password_requirements_match')}
                            </li>
                        </ul>
                    </div>
                </div>
            </form>
        </div>
    );
};
