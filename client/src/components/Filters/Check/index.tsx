import React from 'react';
import { useTranslation } from 'react-i18next';
import { useSelector } from 'react-redux';
import { Controller, useForm } from 'react-hook-form';

import Card from '../../ui/Card';
import Info from './Info';

import { RootState } from '../../../initialState';
import { validateRequiredValue } from '../../../helpers/validators';
import { Input } from '../../ui/Controls/Input';
import { DNS_RECORD_TYPES } from '../../../helpers/constants';
import { Select } from '../../ui/Controls/Select';

export type FilteringCheckFormValues = {
    name: string;
    client?: string;
    qtype?: string;
}

type Props = {
    onSubmit?: (data: FilteringCheckFormValues) => void;
};

const Check = ({ onSubmit }: Props) => {
    const { t } = useTranslation();

    const processingCheck = useSelector((state: RootState) => state.filtering.processingCheck);
    const hostname = useSelector((state: RootState) => state.filtering.check.hostname);

    const {
        control,
        handleSubmit,
        formState: { isValid },
    } = useForm<FilteringCheckFormValues>({
        mode: 'onBlur',
        defaultValues: {
            name: '',
            client: '',
            qtype: DNS_RECORD_TYPES[0],
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
                                    label={t('check_hostname')}
                                    data-testid="check_domain_name"
                                    placeholder="example.com"
                                    error={fieldState.error?.message}
                                />
                            )}
                        />

                        <Controller
                            name="client"
                            control={control}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    data-testid="check_client_id"
                                    label={t('check_client_id')}
                                    placeholder={t('check_enter_client_id')}
                                    error={fieldState.error?.message}
                                />
                            )}
                        />

                        <Controller
                            name="qtype"
                            control={control}
                            render={({ field }) => (
                                <Select
                                    {...field}
                                    label={t('check_dns_record')}
                                    data-testid="check_dns_record_type"
                                >
                                    {DNS_RECORD_TYPES.map((type) => (
                                        <option key={type} value={type}>
                                            {type}
                                        </option>
                                    ))}
                                </Select>
                            )}
                        />

                        <button
                            className="btn btn-success btn-standard btn-large"
                            type="submit"
                            data-testid="check_domain_submit"
                            disabled={!isValid || processingCheck}
                        >
                            {t('check')}
                        </button>

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
