import { createSignal, createEffect, createMemo, Show, onMount } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { addSuccessToast } from 'panel/stores/toasts';
import { filteringState, checkHost, getFilteringStatus, setRules } from 'panel/stores/filtering';
import { settingsState, initSettings } from 'panel/stores/settings';
import { dashboardState, getClients } from 'panel/stores/dashboard';
import { clientsState } from 'panel/stores/clients';
import { servicesState, getBlockedServices, getAllBlockedServices } from 'panel/stores/services';
import { rewritesState, getRewritesList } from 'panel/stores/rewrites';
import { MODAL_TYPE } from 'panel/helpers/constants';
import theme from 'panel/lib/theme';
import { Loader } from 'panel/common/ui/Loader';
import { ConfigureRewritesModal } from 'panel/components/FilterLists/blocks/ConfigureRewritesModal/ConfigureRewritesModal';
import { DeleteRewriteModal } from 'panel/components/FilterLists/blocks/DeleteRewriteModal';

import { CheckForm } from './blocks/CheckForm';
import { CheckResult } from './blocks/CheckResult/CheckResult';
import { Examples } from './blocks/Examples';
import { RulesEditor } from './blocks/RulesEditor';
import { DNS_RECORD_TYPE_OPTIONS } from './types';
import { useUserRulesActions } from './useUserRulesActions';

import type { CheckFormValues } from './types';

import s from './UserRules.module.pcss';

export const UserRules = () => {
    const [lastSubmittedCheck, setLastSubmittedCheck] = createSignal<CheckFormValues | null>(null);
    const [isResultVisible, setIsResultVisible] = createSignal(false);
    const [isResultRefreshing, setIsResultRefreshing] = createSignal(false);

    const [userRulesValue, setUserRulesValue] = createSignal(filteringState.userRules || '');

    const [checkHostname, setCheckHostname] = createSignal('');
    const [checkClient, setCheckClient] = createSignal('');
    const [checkQtype, setCheckQtype] = createSignal(DNS_RECORD_TYPE_OPTIONS[0].value);

    const isActionProcessing = createMemo(
        () =>
            filteringState.processingRules ||
            filteringState.processingCheck ||
            clientsState.processingUpdating ||
            rewritesState.processingDelete ||
            rewritesState.processingUpdate ||
            servicesState.processingSet,
    );

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
        checkResult: () => filteringState.check,
        filteringEnabled: () => filteringState.enabled,
        settingsList: () => settingsState.settingsList,
        persistentClients: () => dashboardState.clients || [],
        rewritesList: () => rewritesState.list,
        lastSubmittedCheck,
        setIsResultVisible,
    });

    onMount(async () => {
        await Promise.all([
            getFilteringStatus(),
            initSettings(),
            getClients(),
            getRewritesList(),
            getBlockedServices(),
            getAllBlockedServices(),
        ]);
    });

    createEffect(() => {
        if (filteringState.check?.hostname) {
            setIsResultVisible(true);
        }
    });

    createEffect(() => {
        setUserRulesValue(filteringState.userRules || '');
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

    const onRulesSubmit = async () => {
        const ok = await setRules(userRulesValue());
        if (ok) {
            addSuccessToast(intl.getMessage('changes_saved_success'));
        }
    };

    const onCheckSubmit = async () => {
        const payload = {
            name: checkHostname().trim(),
            client: checkClient().trim() || undefined,
            qtype: checkQtype() || undefined,
        };

        setLastSubmittedCheck({
            hostname: payload.name,
            client: payload.client || '',
            qtype: payload.qtype || '',
        });

        await runWithResultRefresh(async () => {
            await checkHost(payload);
        });
    };

    const showResultLoader = createMemo(() => isResultVisible() && isResultRefreshing());
    const showResultCard = createMemo(
        () => isResultVisible() && !isResultRefreshing() && Boolean(filteringState.check?.hostname),
    );

    return (
        <>
            <div class={theme.layout.container}>
                <div class={s.container}>
                    <div class={s.wrapper}>
                        <h1 class={cn(theme.title.h4, theme.title.h3_tablet, s.pageTitle)}>
                            {intl.getMessage('user_rules_title')}
                        </h1>

                        <RulesEditor
                            value={userRulesValue()}
                            onChange={setUserRulesValue}
                            handleSubmit={onRulesSubmit}
                            processingRules={filteringState.processingRules}
                        />

                        <Examples />
                    </div>

                    <div class={s.check}>
                        <div class={s.card}>
                            <h2 class={cn(theme.title.h6, s.checkTitle)}>
                                {intl.getMessage('user_rules_check_title')}
                            </h2>

                            <CheckForm
                                hostname={checkHostname()}
                                client={checkClient()}
                                qtype={checkQtype()}
                                onHostnameChange={setCheckHostname}
                                onClientChange={setCheckClient}
                                onQtypeChange={setCheckQtype}
                                handleSubmit={onCheckSubmit}
                                processingCheck={filteringState.processingCheck}
                            />
                        </div>

                        <Show when={showResultLoader()}>
                            <div
                                class={cn(s.card, s.resultLoadingCard)}
                                data-testid="user-rules-result-loader"
                                aria-busy="true"
                            >
                                <Loader class={s.resultLoaderIcon} />
                            </div>
                        </Show>

                        <Show when={showResultCard()}>
                            <CheckResult
                                checkResult={filteringState.check}
                                processingRules={isActionProcessing()}
                                onDismiss={() => setIsResultVisible(false)}
                                onAction={handleAction}
                                onEditRewrite={openEditRewriteModal}
                                onDeleteRewrite={openDeleteRewriteModal}
                                hasMatchedRewrite={Boolean(matchedRewrite())}
                                hiddenActionKinds={hiddenActionKinds()}
                            />
                        </Show>
                    </div>
                </div>
            </div>

            <ConfigureRewritesModal
                modalId={MODAL_TYPE.EDIT_REWRITE}
                rewriteToEdit={currentRewrite()}
                onSubmit={handleRewriteUpdate}
                onClose={resetCurrentRewrite}
            />

            <DeleteRewriteModal
                rewriteToDelete={currentRewrite()}
                setRewriteToDelete={setCurrentRewrite}
                onConfirm={handleRewriteDelete}
            />
        </>
    );
};
