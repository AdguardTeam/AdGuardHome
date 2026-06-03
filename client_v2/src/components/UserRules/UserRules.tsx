import React, { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import cn from 'clsx';
import { batch, useDispatch, useSelector } from 'react-redux';
import type { AppDispatch } from 'panel/store/types';

import intl from 'panel/common/intl';
import { getClients, initSettings } from 'panel/actions';
import { getFilteringStatus, setRules, checkHost } from 'panel/actions/filtering';
import { getRewritesList } from 'panel/actions/rewrites';
import { getBlockedServices, getAllBlockedServices } from 'panel/actions/services';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';
import { Loader } from 'panel/common/ui/Loader';
import { ConfigureRewritesModal } from 'panel/components/FilterLists/blocks/ConfigureRewritesModal/ConfigureRewritesModal';
import { DeleteRewriteModal } from 'panel/components/FilterLists/blocks/DeleteRewriteModal';

import { CheckForm } from './blocks/CheckForm';
import { CheckResult } from './blocks/CheckResult/CheckResult';
import { Examples } from './blocks/Examples';
import { RulesEditor } from './blocks/RulesEditor';
import { CheckFormValues, DNS_RECORD_TYPE_OPTIONS, UserRulesFormValues } from './types';
import { useUserRulesActions } from './useUserRulesActions';

import s from './UserRules.module.pcss';

export const UserRules = () => {
    const dispatch = useDispatch<AppDispatch>();

    const userRules = useSelector((state: RootState) => state.filtering.userRules);
    const filters = useSelector((state: RootState) => state.filtering.filters);
    const whitelistFilters = useSelector((state: RootState) => state.filtering.whitelistFilters);
    const filteringEnabled = useSelector((state: RootState) => state.filtering.enabled);
    const processingRules = useSelector((state: RootState) => state.filtering.processingRules);
    const processingCheck = useSelector((state: RootState) => state.filtering.processingCheck);
    const checkResult = useSelector((state: RootState) => state.filtering.check);
    const settingsList = useSelector((state: RootState) => state.settings.settingsList);
    const persistentClients = useSelector((state: RootState) => state.dashboard.clients || []);
    const processingClientUpdate = useSelector(
        (state: RootState) => state.clients?.processingUpdating || false,
    );
    const rewrites = useSelector((state: RootState) => state.rewrites);
    const services = useSelector((state: RootState) => state.services);

    const [lastSubmittedCheck, setLastSubmittedCheck] = useState<CheckFormValues | null>(null);
    const [isResultVisible, setIsResultVisible] = useState(Boolean(checkResult?.hostname));
    const [isResultRefreshing, setIsResultRefreshing] = useState(false);

    const isActionProcessing =
        processingRules ||
        processingCheck ||
        processingClientUpdate ||
        rewrites.processingDelete ||
        rewrites.processingUpdate ||
        services.processingSet;

    useEffect(() => {
        batch(() => {
            dispatch(getFilteringStatus());
            dispatch(initSettings());
            dispatch(getClients());
            dispatch(getRewritesList());
            dispatch(getBlockedServices());
            dispatch(getAllBlockedServices());
        });
    }, [dispatch]);

    useEffect(() => {
        if (checkResult?.hostname) {
            setIsResultVisible(true);
        }
    }, [checkResult?.hostname]);

    const {
        control: rulesControl,
        handleSubmit: handleRulesSubmit,
        reset: resetRulesForm,
    } = useForm<UserRulesFormValues>({
        defaultValues: {
            userRules: userRules || '',
        },
        mode: 'onChange',
    });

    useEffect(() => {
        resetRulesForm({ userRules: userRules || '' });
    }, [userRules, resetRulesForm]);

    const {
        control: checkControl,
        handleSubmit: handleCheckSubmit,
        formState: { isValid: isCheckValid },
    } = useForm<CheckFormValues>({
        defaultValues: {
            hostname: '',
            client: '',
            qtype: DNS_RECORD_TYPE_OPTIONS[0].value,
        },
        mode: 'onChange',
    });

    const {
        currentRewrite,
        setCurrentRewrite,
        matchedRewrite,
        hiddenActionKinds,
        handleAction,
        handleRewriteUpdate,
        handleRewriteDelete,
        openEditRewriteModal,
        openDeleteRewriteModal,
        resetCurrentRewrite,
    } = useUserRulesActions({
        checkResult,
        filters,
        whitelistFilters,
        filteringEnabled,
        settingsList,
        persistentClients,
        rewritesList: rewrites.list,
        services,
        lastSubmittedCheck,
        setIsResultVisible,
    });

    const runWithResultRefresh = async (callback: () => Promise<unknown>) => {
        setIsResultVisible(true);
        setIsResultRefreshing(true);

        try {
            await callback();
        } finally {
            setIsResultRefreshing(false);
        }
    };

    const onRulesSubmit = (data: UserRulesFormValues) => {
        dispatch(setRules(data.userRules));
    };

    const onCheckSubmit = async (data: CheckFormValues) => {
        const payload = {
            name: data.hostname.trim(),
            client: data.client.trim() || undefined,
            qtype: data.qtype || undefined,
        };

        setLastSubmittedCheck({
            hostname: payload.name,
            client: payload.client || '',
            qtype: payload.qtype || '',
        });

        await runWithResultRefresh(async () => {
            await dispatch(checkHost(payload));
        });
    };

    const showResultLoader = isResultVisible && isResultRefreshing;
    const showResultCard = isResultVisible && !isResultRefreshing && Boolean(checkResult?.hostname);

    return (
        <>
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

                    <div className={s.check}>
                        <div className={s.card}>
                            <h2 className={cn(theme.title.h6, s.checkTitle)}>
                                {intl.getMessage('user_rules_check_title')}
                            </h2>

                            <CheckForm
                                control={checkControl}
                                handleSubmit={handleCheckSubmit}
                                onSubmit={onCheckSubmit}
                                isValid={isCheckValid}
                                processingCheck={processingCheck}
                            />
                        </div>

                        {showResultLoader && (
                            <div
                                className={cn(s.card, s.resultLoadingCard)}
                                data-testid="user-rules-result-loader"
                                aria-busy="true"
                            >
                                <Loader className={s.resultLoaderIcon} />
                            </div>
                        )}

                        {showResultCard && (
                            <CheckResult
                                checkResult={checkResult}
                                processingRules={isActionProcessing}
                                onDismiss={() => setIsResultVisible(false)}
                                onAction={handleAction}
                                onEditRewrite={openEditRewriteModal}
                                onDeleteRewrite={openDeleteRewriteModal}
                                hasMatchedRewrite={Boolean(matchedRewrite)}
                                hiddenActionKinds={hiddenActionKinds}
                            />
                        )}
                    </div>
                </div>
            </div>

            <ConfigureRewritesModal
                modalId={MODAL_TYPE.EDIT_REWRITE}
                rewriteToEdit={currentRewrite}
                onSubmit={handleRewriteUpdate}
                onClose={resetCurrentRewrite}
            />

            <DeleteRewriteModal
                rewriteToDelete={currentRewrite}
                setRewriteToDelete={setCurrentRewrite}
                onConfirm={handleRewriteDelete}
            />
        </>
    );
};
