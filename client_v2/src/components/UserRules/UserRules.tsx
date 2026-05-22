import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import cn from 'clsx';
import { batch, useDispatch, useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import { PageLoader } from 'panel/common/ui/Loader';
import { initSettings } from 'panel/actions';
import { getFilteringStatus, setRules } from 'panel/actions/filtering';
import { getRewritesList } from 'panel/actions/rewrites';
import { getBlockedServices, getAllBlockedServices } from 'panel/actions/services';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';

import { Examples } from './blocks/Examples';
import { RulesEditor } from './blocks/RulesEditor';
import { UserRulesFormValues } from './types';

import s from './UserRules.module.pcss';

export const UserRules = () => {
    const dispatch = useDispatch();

    const userRules = useSelector((state: RootState) => state.filtering.userRules);
    const processingRules = useSelector((state: RootState) => state.filtering.processingRules);
    const processingFilters = useSelector((state: RootState) => state.filtering.processingFilters);
    const isDataLoading = processingFilters;

    useEffect(() => {
        batch(() => {
            dispatch(getFilteringStatus());
            dispatch(initSettings());
            dispatch(getRewritesList());
            dispatch(getBlockedServices());
            dispatch(getAllBlockedServices());
        });
    }, [dispatch]);

    const {
        control: rulesControl,
        handleSubmit: handleRulesSubmit,
        reset: resetRulesForm,
    } = useForm<UserRulesFormValues>({
        defaultValues: {
            userRules: userRules || '',
        },
        mode: 'onBlur',
    });

    useEffect(() => {
        resetRulesForm({ userRules: userRules || '' });
    }, [userRules, resetRulesForm]);

    const onRulesSubmit = (data: UserRulesFormValues) => {
        dispatch(setRules(data.userRules));
    };

    if (isDataLoading) {
        return (
            <div className={theme.layout.container}>
                <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                    <PageLoader />
                </div>
            </div>
        );
    }

    return (
        <div className={theme.layout.container}>
            <div className={s.container}>
                <div className={s.wrapper}>
                    <h1 className={cn(theme.title.h4, theme.title.h3_tablet, s.pageTitle)}>
                        {intl.getMessage('user_rules_title')}
                    </h1>

                    <RulesEditor
                        control={rulesControl}
                        handleSubmit={handleRulesSubmit}
                        onSubmit={onRulesSubmit}
                        processingRules={processingRules}
                    />

                    <Examples />
                </div>
            </div>
        </div>
    );
};
