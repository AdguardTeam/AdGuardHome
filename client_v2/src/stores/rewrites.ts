import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import {
    rewriteList,
    rewriteAdd,
    rewriteUpdate,
    rewriteDelete,
    rewriteSettingsGet,
    rewriteSettingsUpdate,
} from 'panel/api/generated';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import type { RewriteEntry } from 'panel/api/model/rewriteEntry';
import type { RewriteSettings } from 'panel/api/model/rewriteSettings';

type RewritesState = {
    processing: boolean;
    processingAdd: boolean;
    processingDelete: boolean;
    processingUpdate: boolean;
    processingSettings: boolean;
    isModalOpen: boolean;
    modalType: string;
    currentRewrite: RewriteEntry;
    list: (RewriteEntry & { enabled?: boolean })[];
    enabled: boolean;
};

const initialState: RewritesState = {
    processing: true,
    processingAdd: false,
    processingDelete: false,
    processingUpdate: false,
    processingSettings: false,
    isModalOpen: false,
    modalType: '',
    currentRewrite: {},
    list: [],
    enabled: true,
};

const [state, setState] = createStore<RewritesState>(initialState);

export const toggleRewritesModal = (modalType?: string, currentRewrite?: RewriteEntry) => {
    if (modalType !== undefined) {
        setState({
            isModalOpen: !state.isModalOpen,
            modalType: modalType || '',
            currentRewrite: currentRewrite || {},
        });
    } else {
        setState('isModalOpen', (prev) => !prev);
        // modalType and currentRewrite left unchanged
    }
};

export const getRewritesList = async () => {
    setState('processing', true);
    try {
        const data = await rewriteList();
        setState({ list: data || [], processing: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const addRewrite = async (config: RewriteEntry) => {
    setState('processingAdd', true);
    try {
        await rewriteAdd(config);
        setState('processingAdd', false);
        toggleRewritesModal();
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getRewritesList();
    } catch (error) {
        addErrorToast({ error });
        setState('processingAdd', false);
    }
};

export const updateRewrite = async (
    config: { target: RewriteEntry; update: RewriteEntry },
    options: { showToast?: boolean; closeModal?: boolean } = {},
): Promise<boolean> => {
    setState('processingUpdate', true);
    try {
        await rewriteUpdate(config);
        setState('processingUpdate', false);
        if (options.closeModal !== false) {
            toggleRewritesModal();
        }
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getRewritesList();
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingUpdate', false);
        return false;
    }
};

export const deleteRewrite = async (config: RewriteEntry): Promise<boolean> => {
    setState('processingDelete', true);
    try {
        await rewriteDelete(config);
        setState('processingDelete', false);
        addSuccessToast(intl.getMessage('dns_rewrite_removed'));
        await getRewritesList();
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingDelete', false);
        return false;
    }
};

export const getRewriteSettings = async () => {
    setState('processingSettings', true);
    try {
        const data = await rewriteSettingsGet();
        setState({ enabled: data.enabled ?? true, processingSettings: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSettings', false);
    }
};

export const updateRewriteSettings = async (values: RewriteSettings) => {
    setState('processingSettings', true);
    try {
        await rewriteSettingsUpdate(values);
        setState({ ...values, processingSettings: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSettings', false);
    }
};

export const rewritesState = untrack(() => state);
