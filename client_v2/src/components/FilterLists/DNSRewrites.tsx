import { createSignal, createMemo, Show, onMount, createEffect } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { MODAL_TYPE } from 'panel/helpers/constants';
import theme from 'panel/lib/theme';
import {
    getRewritesList,
    updateRewrite,
    getRewriteSettings,
    updateRewriteSettings,
    rewritesState,
} from 'panel/stores/rewrites';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { openModal } from 'panel/stores/modals';
import { DeleteRewriteModal } from 'panel/components/FilterLists/blocks/DeleteRewriteModal';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { ConfigureRewritesModal } from 'panel/components/FilterLists/blocks/ConfigureRewritesModal/ConfigureRewritesModal';
import { RewritesTable } from './blocks/RewritesTable/RewritesTable';

import s from './FilterLists.module.pcss';

export type Rewrite = {
    answer: string;
    domain: string;
    enabled: boolean;
};

export const DNSRewrites = () => {
    const [currentRewrite, setCurrentRewrite] = createSignal<Rewrite>({
        answer: '',
        domain: '',
        enabled: false,
    });

    const [isConfirmOpen, setIsConfirmOpen] = createSignal(false);
    const [targetEnabled, setTargetEnabled] = createSignal<boolean | null>(null);
    const [settingsLoaded, setSettingsLoaded] = createSignal(false);

    // Only show loader during initial fetch, not on subsequent enable/disable toggles.
    createEffect(() => {
        if (!rewritesState.processingSettings && !settingsLoaded()) {
            setSettingsLoaded(true);
        }
    });

    const isInitialSettingsLoad = createMemo(
        () => rewritesState.processingSettings && !settingsLoaded(),
    );

    onMount(() => {
        getRewritesList();
        getRewriteSettings();
    });

    const openAddRewiresModal = () => {
        openModal(MODAL_TYPE.ADD_REWRITE);
    };

    const openEditRewriteModal = (rewrite: Rewrite) => {
        setCurrentRewrite(rewrite);
        openModal(MODAL_TYPE.EDIT_REWRITE);
    };

    const openDeleteRewriteModal = (rewrite: Rewrite) => {
        setCurrentRewrite(rewrite);
        openModal(MODAL_TYPE.DELETE_REWRITE);
    };

    const toggleRewrite = (rewrite: Rewrite) => {
        const updatedRewrite = { ...rewrite, enabled: !rewrite.enabled };

        updateRewrite({
            target: rewrite,
            update: updatedRewrite,
        });
    };

    const openGlobalToggleConfirm = (value: boolean) => {
        setTargetEnabled(value);
        setIsConfirmOpen(true);
    };

    const closeGlobalToggleConfirm = () => {
        setIsConfirmOpen(false);
        setTargetEnabled(null);
    };

    const confirmGlobalToggle = () => {
        if (targetEnabled() === null) {
            return;
        }

        updateRewriteSettings({ enabled: targetEnabled()! });
        closeGlobalToggleConfirm();
    };

    return (
        <div class={cn(theme.layout.container, s.dnsRewritesContainer)}>
            <div class={theme.layout.containerIn}>
                <SettingRow
                    variant="switch"
                    id="rewrite_global_enabled"
                    title={intl.getMessage('dns_rewrites')}
                    titleClass={cn(theme.title.h4, theme.title.h3_tablet)}
                    description={intl.getMessage('dns_rewrites_desc')}
                    descriptionClass={s.settingRowDesc}
                    checked={rewritesState.enabled}
                    disabled={isInitialSettingsLoad()}
                    onChange={(value: boolean) => openGlobalToggleConfirm(value)}
                    align="center"
                    class={s.dnsRewritesSettingRow}
                    inputClass={s.dnsRewritesSettingRowInput}
                />

                <div class={cn(s.group, s.buttonGroup)}>
                    <PlusButton onClick={openAddRewiresModal} testId="add-rewrite">
                        {intl.getMessage('rewrite_add')}
                    </PlusButton>
                </div>

                <Show when={rewritesState.list.length > 0}>
                    <div class={cn(s.group, s.tableGroup)}>
                        <RewritesTable
                            list={rewritesState.list}
                            processing={rewritesState.processing}
                            processingAdd={rewritesState.processingAdd}
                            processingUpdate={rewritesState.processingUpdate}
                            processingDelete={rewritesState.processingDelete}
                            addRewritesList={openAddRewiresModal}
                            deleteRewrite={openDeleteRewriteModal}
                            editRewrite={openEditRewriteModal}
                            toggleRewrite={toggleRewrite}
                        />
                    </div>
                </Show>

                <ConfigureRewritesModal modalId={MODAL_TYPE.ADD_REWRITE} />

                <ConfigureRewritesModal
                    modalId={MODAL_TYPE.EDIT_REWRITE}
                    rewriteToEdit={currentRewrite()}
                />

                <DeleteRewriteModal
                    rewriteToDelete={currentRewrite()}
                    setRewriteToDelete={setCurrentRewrite}
                />

                <Show when={isConfirmOpen() && targetEnabled() !== null}>
                    <ConfirmDialog
                        onClose={closeGlobalToggleConfirm}
                        title={
                            targetEnabled()
                                ? intl.getMessage('enable_dns_rewrites')
                                : intl.getMessage('disable_dns_rewrites')
                        }
                        text={
                            targetEnabled()
                                ? intl.getMessage('all_rewrites_enabled')
                                : intl.getMessage('all_rewrites_disabled')
                        }
                        buttonText={
                            targetEnabled()
                                ? intl.getMessage('enable')
                                : intl.getMessage('yes_disable')
                        }
                        cancelText={intl.getMessage('cancel')}
                        buttonVariant={targetEnabled() ? 'primary' : 'danger'}
                        onConfirm={confirmGlobalToggle}
                        submitTestId={
                            targetEnabled() ? 'confirm-enable-rewrites' : 'confirm-disable-rewrites'
                        }
                        cancelTestId="cancel-toggle-rewrites"
                    />
                </Show>
            </div>
        </div>
    );
};
