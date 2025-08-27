import React from 'react';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';

import { setAccessList } from 'panel/actions/access';
import { RootState } from 'panel/initialState';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { Form } from './Form';

export const Access = () => {
    const dispatch = useDispatch();
    const { processingSet, ...values } = useSelector((state: RootState) => state.access, shallowEqual);

    const handleFormSubmit = (values: any) => {
        dispatch(setAccessList(values));
    };

    return (
        <div>
            <h2
                className={cn(
                    theme.layout.subtitle,
                    theme.layout.subtitle_compact,
                    theme.title.h5,
                    theme.title.h4_tablet,
                )}>
                {intl.getMessage('access_settings_title')}
            </h2>

            <Form initialValues={values} onSubmit={handleFormSubmit} processingSet={processingSet} />
        </div>
    );
};
