import { createMemo } from 'solid-js';
import { createForm, setValue, getValue } from '@modular-forms/solid';
import { Input } from 'panel/common/controls/Input';
import intl from 'panel/common/intl';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { PRIVACY_POLICY_LINK, TERMS_LINK } from 'panel/helpers/constants';
import cn from 'clsx';
import { PasswordRequirements } from 'panel/install/Setup/blocks';
import { PasswordInput } from '../../common/controls/Input/PasswordInput';
import { Controls } from './Controls';
import { validatePasswordLength, validateRequiredValue } from '../../helpers/validators';
import {
    hasMinLength,
    hasLowercase,
    hasUppercase,
    hasAllowedAsciiOnly,
    hasNumberOrSpecial,
} from './helpers/helpers';
import styles from './styles.module.pcss';

type AuthFormValues = {
    username: string;
    password: string;
    confirm_password: string;
    privacy_consent: boolean;
};

type AuthInitialValues = {
    username?: string;
    password?: string;
    privacy_consent?: boolean;
};

type Props = {
    onAuthSubmit: (values: AuthFormValues) => void;
    initialValues?: AuthInitialValues;
};

export const Auth = (props: Props) => {
    const [authForm, { Form, Field }] = createForm<AuthFormValues>({
        initialValues: {
            username: props.initialValues?.username ?? '',
            password: props.initialValues?.password ?? '',
            confirm_password: props.initialValues?.password ?? '',
            privacy_consent: props.initialValues?.privacy_consent ?? false,
        },
        validateOn: 'change',
    });

    const password = () => (getValue(authForm, 'password') as string) ?? '';
    const confirmPassword = () => (getValue(authForm, 'confirm_password') as string) ?? '';

    const validateConfirmPassword = (value: string) => {
        if (value !== password()) {
            return intl.getMessage('form_error_password');
        }
        return undefined;
    };

    const validatePasswordLowercase = (value: string) => {
        if (value && !hasLowercase(value)) {
            return intl.getMessage('password_requirements_lowercase');
        }
        return undefined;
    };

    const validatePasswordUppercase = (value: string) => {
        if (value && !hasUppercase(value)) {
            return intl.getMessage('password_requirements_uppercase');
        }
        return undefined;
    };

    const validatePasswordSpecial = (value: string) => {
        if (value && !(hasAllowedAsciiOnly(value) && hasNumberOrSpecial(value))) {
            return intl.getMessage('password_requirements_special');
        }
        return undefined;
    };

    const requirements = createMemo(() => {
        const pwd = password();
        const confirmPwd = confirmPassword();
        return {
            minLength: pwd.length > 0 && hasMinLength(pwd),
            allowedChars: pwd.length > 0 && hasAllowedAsciiOnly(pwd) && hasNumberOrSpecial(pwd),
            lowercase: pwd.length > 0 && hasLowercase(pwd),
            uppercase: pwd.length > 0 && hasUppercase(pwd),
            match: pwd.length > 0 && confirmPwd.length > 0 && pwd === confirmPwd,
        };
    });

    const isPrivacyValid = createMemo(() =>
        Boolean(getValue(authForm, 'privacy_consent') as boolean | undefined),
    );

    return (
        <div class={styles.configSetting}>
            <Form class={styles.step} onSubmit={props.onAuthSubmit}>
                <div class={styles.info}>
                    <div class={styles.titleStep}>{intl.getMessage('setup_guide_auth_title')}</div>

                    <p class={styles.descStep}>{intl.getMessage('setup_guide_auth_subtitle')}</p>

                    <div class={styles.input}>
                        <Field name="username" validate={validateRequiredValue}>
                            {(field, props) => (
                                <Input
                                    {...props}
                                    type="text"
                                    id="install_username"
                                    label={intl.getMessage('install_auth_username')}
                                    placeholder={intl.getMessage('install_auth_username_enter')}
                                    autocomplete="username"
                                    value={(field.value as string) ?? ''}
                                    errorMessage={field.error as string}
                                    size="large"
                                />
                            )}
                        </Field>
                    </div>

                    <div class={styles.input}>
                        <Field
                            name="password"
                            validate={
                                [
                                    validateRequiredValue,
                                    validatePasswordLength,
                                    validatePasswordLowercase,
                                    validatePasswordUppercase,
                                    validatePasswordSpecial,
                                ] as any
                            }
                        >
                            {(field, props) => (
                                <PasswordInput
                                    {...props}
                                    id="install_password"
                                    label={intl.getMessage('install_auth_password')}
                                    placeholder={intl.getMessage('install_auth_password_enter')}
                                    autocomplete="new-password"
                                    value={(field.value as string) ?? ''}
                                    onChange={(value) =>
                                        setValue(authForm, 'password', value, {
                                            shouldValidate: true,
                                        })
                                    }
                                    errorMessage={field.error as string}
                                    size="large"
                                />
                            )}
                        </Field>
                    </div>

                    <PasswordRequirements
                        requirements={requirements()}
                        class={styles.authRequirementsMobile}
                    />

                    <div class={styles.input}>
                        <Field
                            name="confirm_password"
                            validate={[validateRequiredValue, validateConfirmPassword] as any}
                        >
                            {(field, props) => (
                                <PasswordInput
                                    {...props}
                                    id="install_confirm_password"
                                    label={intl.getMessage('install_auth_confirm')}
                                    placeholder={intl.getMessage('install_auth_confirm')}
                                    autocomplete="new-password"
                                    value={(field.value as string) ?? ''}
                                    onChange={(value) =>
                                        setValue(authForm, 'confirm_password', value, {
                                            shouldValidate: true,
                                        })
                                    }
                                    errorMessage={field.error as string}
                                    size="large"
                                />
                            )}
                        </Field>
                    </div>

                    <div class={styles.consent}>
                        <Field
                            name="privacy_consent"
                            type="boolean"
                            validate={validateRequiredValue as any}
                        >
                            {(field, props) => (
                                <Checkbox
                                    {...props}
                                    checked={(field.value as boolean) ?? false}
                                    onChange={(e: Event) =>
                                        setValue(
                                            authForm,
                                            'privacy_consent',
                                            (e.target as HTMLInputElement).checked,
                                            { shouldValidate: true },
                                        )
                                    }
                                    verticalAlign="start"
                                >
                                    <div class={styles.consentContent}>
                                        {intl.getMessage('setup_guide_auth_privacy', {
                                            a: (text: string) => (
                                                <a class={styles.link} href={PRIVACY_POLICY_LINK}>
                                                    {text}
                                                </a>
                                            ),
                                            b: (text: string) => (
                                                <a class={styles.link} href={TERMS_LINK}>
                                                    {text}
                                                </a>
                                            ),
                                        })}
                                    </div>
                                </Checkbox>
                            )}
                        </Field>
                    </div>

                    <Controls isDirty={authForm.dirty} isValid={isPrivacyValid()} />
                </div>

                <div class={styles.content}>
                    <PasswordRequirements
                        requirements={requirements()}
                        class={cn(styles.banner, styles.authBanner)}
                    />
                </div>
            </Form>
        </div>
    );
};
