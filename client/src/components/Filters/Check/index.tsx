import React from 'react';
import { useTranslation } from 'react-i18next';
import { useSelector } from 'react-redux';
import { Controller, useForm } from 'react-hook-form';

import Card from '../../ui/Card';
import Info from './Info';

import { RootState } from '../../../initialState';
import { validateRequiredValue } from '../../../helpers/validators';
import { Input } from '../../ui/Controls/Input';

interface FormValues {
    name: string;
}

type Props = {
    onSubmit?: (data: FormValues) => void;
};

const Check = ({ onSubmit }: Props) => {
    const { t } = useTranslation();

    const processingCheck = useSelector((state: RootState) => state.filtering.processingCheck);
    const hostname = useSelector((state: RootState) => state.filtering.check.hostname);

    const {
        control,
        handleSubmit,
        formState: { isDirty, isValid },
    } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: {
            name: '',
        },
    });

    return (
        <Card title={t('check_title')} subtitle={t('check_desc')}>
            <form onSubmit={handleSubmit(onSubmit)}>
                <div className="row">
                    <div className="col-12 col-md-6">
                        <Controller
                            name="name"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    data-testid="check_domain_name"
                                    placeholder={t('form_enter_host')}
                                    error={fieldState.error?.message}
                                    rightAddon={
                                        <span className="input-group-append">
                                            <button
                                                className="btn btn-success btn-standard btn-large"
                                                type="submit"
                                                data-testid="check_domain_submit"
                                                disabled={!isDirty || !isValid || processingCheck}>
                                                {t('check')}
                                            </button>
                                        </span>
                                    }
                                />
                            )}
                        />

                        {hostname && (
                            <>
                                <hr />
                                <Info />
                            </>
                        )}
                    </div>
                </div>
            </form>
        </Card>
    );
};

export default Check;
