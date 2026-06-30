import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';

type RewriteConfig = {
    answer: string;
    domain: string;
    enabled: boolean;
};

type RewritesState = {
    processing: boolean;
    processingAdd: boolean;
    processingDelete: boolean;
    processingUpdate: boolean;
    processingSettings: boolean;
    isModalOpen: boolean;
    modalType: string;
    currentRewrite: RewriteConfig | Record<string, never>;
    list: RewriteConfig[];
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

export const toggleRewritesModal = (modalType?: string, currentRewrite?: RewriteConfig) => {
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
        const data = await apiClient.getRewritesList();
        setState({ list: data || [], processing: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const addRewrite = async (config: RewriteConfig) => {
    setState('processingAdd', true);
    try {
        await apiClient.addRewrite(config);
        setState('processingAdd', false);
        toggleRewritesModal();
        await getRewritesList();
    } catch (error) {
        addErrorToast({ error });
        setState('processingAdd', false);
    }
};

export const updateRewrite = async (
    config: { target: RewriteConfig; update: RewriteConfig },
    options: { showToast?: boolean; closeModal?: boolean } = {},
): Promise<boolean> => {
    setState('processingUpdate', true);
    try {
        await apiClient.updateRewrite(config);
        setState('processingUpdate', false);
        if (options.closeModal !== false) {
            toggleRewritesModal();
        }
        await getRewritesList();
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingUpdate', false);
        return false;
    }
};

export const deleteRewrite = async (config: RewriteConfig): Promise<boolean> => {
    setState('processingDelete', true);
    try {
        await apiClient.deleteRewrite(config);
        setState('processingDelete', false);
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
        const data = await apiClient.getRewriteSettings();
        setState({ enabled: data.enabled ?? true, processingSettings: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSettings', false);
    }
};

export const updateRewriteSettings = async (values: any) => {
    setState('processingSettings', true);
    try {
        await apiClient.updateRewriteSettings(values);
        setState({ ...values, processingSettings: false });
        addSuccessToast(intl.getMessage('rewrite_settings_updated'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingSettings', false);
    }
};

export const rewritesState = untrack(() => state);
