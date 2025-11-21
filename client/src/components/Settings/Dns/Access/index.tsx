import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual } from 'react-redux';
import { useDispatch, useSelector } from '@/store/hooks';


import Card from '@/components/ui/Card';
import { setAccessList } from '@/actions/access';
import { RootState } from '@/initialState';
import Form from './Form';

const Access = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const { processingSet, ...values } = useSelector((state: RootState) => state.access, shallowEqual);

    const handleFormSubmit = (values: any) => {
        dispatch(setAccessList(values));
    };

    return (
        <Card title={t('access_title')} subtitle={t('access_desc')} bodyType="card-body box-body--settings">
            <Form initialValues={values} onSubmit={handleFormSubmit} processingSet={processingSet} />
        </Card>
    );
};

export default Access;
