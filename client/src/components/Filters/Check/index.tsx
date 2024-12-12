import React from 'react';
import { useTranslation } from 'react-i18next';
import { useSelector } from 'react-redux';
import { useForm } from 'react-hook-form';

import Card from '../../ui/Card';
import Info from './Info';

import { RootState } from '../../../initialState';

interface FormValues {
    name: string;
}

type Props = {
    onSubmit?: (data: FormValues) => void;
}

const Check = ({ onSubmit }: Props) => {
    const { t } = useTranslation();

    const processingCheck = useSelector((state: RootState) => state.filtering.processingCheck);
    const hostname = useSelector((state: RootState) => state.filtering.check.hostname);

    const {
        register,
        handleSubmit,
        formState: { isDirty, isValid },
    } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
            name: '',
        },
    });

    return (
        <Card title={t('check_title')} subtitle={t('check_desc')}>
            <form onSubmit={handleSubmit(onSubmit)}>
                <div className="row">
                    <div className="col-12 col-md-6">
                        <div className="input-group">
                            <input
                                id="name"
                                type="text"
                                className="form-control"
                                placeholder={t('form_enter_host') ?? ''}
                                {...register('name', { required: true })}
                            />
                            <span className="input-group-append">
                                <button
                                    className="btn btn-success btn-standard btn-large"
                                    type="submit"
                                    disabled={!isDirty || !isValid || processingCheck}
                                >
                                    {t('check')}
                                </button>
                            </span>
                        </div>

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
