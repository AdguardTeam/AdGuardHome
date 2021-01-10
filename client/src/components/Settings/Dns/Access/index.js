import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Form from './Form';
import Card from '../../../ui/Card';
import { setAccessList } from '../../../../actions/access';

const Access = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        processing,
        processingSet,
        ...values
    } = useSelector((state) => state.access, shallowEqual);

    const handleFormSubmit = (values) => {
        dispatch(setAccessList(values));
    };

    return (
        <Card
            title={t('access_title')}
            subtitle={t('access_desc')}
            bodyType="card-body box-body--settings"
        >
            <Form
                initialValues={values}
                onSubmit={handleFormSubmit}
                processingSet={processingSet}
            />
        </Card>
    );
};

export default Access;
