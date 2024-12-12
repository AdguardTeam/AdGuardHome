import React from 'react';
import { useForm } from 'react-hook-form';
import { withTranslation, Trans } from 'react-i18next';
import flow from 'lodash/flow';
import cn from 'classnames';
import i18n from '../../i18n';
import Controls from './Controls';
import { validatePasswordLength } from '../../helpers/validators';

type Props = {
    onAuthSubmit: (...args: unknown[]) => string;
    pristine: boolean;
    invalid: boolean;
    t: (...args: unknown[]) => string;
}

const Auth = (props: Props) => {
    const { t, onAuthSubmit } = props;
    const {
        register,
        handleSubmit,
        watch,
        formState: { errors, isDirty, isValid },
    } = useForm({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
            confirm_password: '',
        },
    });

    const password = watch('password');

    const validateConfirmPassword = (value: string) => {
        if (value !== password) {
            return i18n.t('form_error_password');
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
                    <label>
                        <Trans>install_auth_username</Trans>
                    </label>
                    <input
                        {...register('username', { required: t('form_error_required') })}
                        type="text"
                        className={cn('form-control', { 'is-invalid': errors.username })}
                        placeholder={t('install_auth_username_enter')}
                        autoComplete="username"
                    />
                    {errors.username && (
                        <div className="form__message form__message--error">
                            {errors.username.message}
                        </div>
                    )}
                </div>

                <div className="form-group">
                    <label>
                        <Trans>install_auth_password</Trans>
                    </label>
                    <input
                        {...register('password', {
                            required: t('form_error_required'),
                            validate: validatePasswordLength,
                        })}
                        type="password"
                        className={cn('form-control', { 'is-invalid': errors.password })}
                        placeholder={t('install_auth_password_enter')}
                        autoComplete="new-password"
                    />
                    {errors.password && (
                        <div className="form__message form__message--error">
                            {errors.password.message}
                        </div>
                    )}
                </div>

                <div className="form-group">
                    <label>
                        <Trans>install_auth_confirm</Trans>
                    </label>
                    <input
                        {...register('confirm_password', {
                            required: t('form_error_required'),
                            validate: validateConfirmPassword,
                        })}
                        type="password"
                        className={cn('form-control', { 'is-invalid': errors.confirm_password })}
                        placeholder={t('install_auth_confirm')}
                        autoComplete="new-password"
                    />
                    {errors.confirm_password && (
                        <div className="invalid-feedback">
                            {errors.confirm_password.message}
                        </div>
                    )}
                </div>
            </div>

            <Controls
                isDirty={isDirty}
                isValid={isValid}
            />
        </form>
    );
};

export default flow([
    withTranslation(),
])(Auth);
