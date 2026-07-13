import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast, createUndoToast } from './toasts';
import type { Filter } from 'panel/helpers/helpers';
import { normalizeFilteringStatus, normalizeRulesTextarea } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';

type FilteringState = {
    isModalOpen: boolean;
    processingFilters: boolean;
    processingRules: boolean;
    processingAddFilter: boolean;
    processingRefreshFilters: boolean;
    processingConfigFilter: boolean;
    processingRemoveFilter: boolean;
    processingSetConfig: boolean;
    processingCheck: boolean;
    isFilterAdded: boolean;
    isFilterRemoved: boolean;
    isFilterEdited: boolean;
    filters: Filter[];
    whitelistFilters: any[];
    userRules: string;
    interval: number;
    enabled: boolean;
    modalType: string;
    modalFilterUrl: string;
    check: any;
};

const initialState: FilteringState = {
    isModalOpen: false,
    processingFilters: false,
    processingRules: false,
    processingAddFilter: false,
    processingRefreshFilters: false,
    processingConfigFilter: false,
    processingRemoveFilter: false,
    processingSetConfig: false,
    processingCheck: false,
    isFilterAdded: false,
    isFilterRemoved: false,
    isFilterEdited: false,
    filters: [],
    whitelistFilters: [],
    userRules: '',
    interval: 24,
    enabled: true,
    modalType: '',
    modalFilterUrl: '',
    check: {},
};

const [state, setState] = createStore<FilteringState>(initialState);

export const getFilteringStatus = async () => {
    setState('processingFilters', true);
    try {
        const data = await apiClient.getFilteringStatus();
        setState({
            ...normalizeFilteringStatus(data),
            processingFilters: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingFilters', false);
    }
};

export const setRules = async (rules: string): Promise<boolean> => {
    setState('processingRules', true);
    try {
        const normalizedRules = {
            rules: (normalizeRulesTextarea(rules)?.split('\n') || []).filter(Boolean),
        };
        await apiClient.setRules(normalizedRules);
        setState({ userRules: rules, processingRules: false });
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingRules', false);
        return false;
    }
};

const splitByNewLine = (str: string): string[] => str.split('\n').filter(Boolean);

export const blockDomain = async (domain: string): Promise<boolean> => {
    const previousRules = state.userRules || '';
    const rule = `||${domain}^$important`;
    const currentRules = splitByNewLine(previousRules);

    if (currentRules.includes(rule)) {
        return true;
    }

    const updatedRules = [...currentRules.filter((r: string) => r !== `@@${rule}`), rule];
    const didSave = await setRules(`${updatedRules.join('\n')}\n`);

    if (!didSave) {
        return false;
    }

    addSuccessToast(
        createUndoToast(
            intl.getMessage('user_rules_rule_added', { rule }),
            intl.getMessage('notify_undo'),
            async () => {
                const didUndo = await setRules(previousRules);
                if (didUndo) {
                    await getFilteringStatus();
                }
            },
        ),
    );
    await getFilteringStatus();
    return true;
};

export const unblockDomain = async (domain: string): Promise<boolean> => {
    const previousRules = state.userRules || '';
    const rule = `||${domain}^$important`;
    const desiredRule = `@@${rule}`;
    const currentRules = splitByNewLine(previousRules);

    if (currentRules.includes(desiredRule)) {
        return true;
    }

    const updatedRules = [...currentRules.filter((r: string) => r !== rule), desiredRule];
    const didSave = await setRules(`${updatedRules.join('\n')}\n`);

    if (!didSave) {
        return false;
    }

    addSuccessToast(
        createUndoToast(
            intl.getMessage('user_rules_rule_added', { rule: desiredRule }),
            intl.getMessage('notify_undo'),
            async () => {
                const didUndo = await setRules(previousRules);
                if (didUndo) {
                    await getFilteringStatus();
                }
            },
        ),
    );
    await getFilteringStatus();
    return true;
};

export const blockDomainForClient = async (domain: string, client: string): Promise<boolean> => {
    const previousRules = state.userRules || '';
    const rule = `||${domain}^$client=${client}`;
    const currentRules = splitByNewLine(previousRules);

    if (currentRules.includes(rule)) {
        return true;
    }

    const updatedRules = [...currentRules.filter((r: string) => r !== `@@${rule}`), rule];
    const didSave = await setRules(`${updatedRules.join('\n')}\n`);

    if (!didSave) {
        return false;
    }

    addSuccessToast(
        createUndoToast(
            intl.getMessage('user_rules_rule_added_to_custom_filtering_rules'),
            intl.getMessage('notify_undo'),
            async () => {
                const didUndo = await setRules(previousRules);
                if (didUndo) {
                    await getFilteringStatus();
                }
            },
        ),
    );
    await getFilteringStatus();
    return true;
};

export const BLOCK_ACTIONS = {
    BLOCK: 'block' as const,
    UNBLOCK: 'unblock' as const,
};

export type BlockAction = (typeof BLOCK_ACTIONS)[keyof typeof BLOCK_ACTIONS];

export const toggleBlocking = async (
    type: BlockAction,
    domain: string,
    baseRule?: string,
    baseUnblocking?: string,
    matchedRuleToReplace?: string,
): Promise<boolean> => {
    const baseBlockingRule = baseRule || `||${domain}^$important`;
    const baseUnblockingRule = baseUnblocking || `@@${baseBlockingRule}`;
    const previousRules = state.userRules || '';
    const desiredRule = type === BLOCK_ACTIONS.BLOCK ? baseBlockingRule : baseUnblockingRule;
    const oppositeRule = type === BLOCK_ACTIONS.BLOCK ? baseUnblockingRule : baseBlockingRule;
    const currentRules = splitByNewLine(previousRules);
    const hasDesiredRule = currentRules.includes(desiredRule);
    const rulesToReplace = [oppositeRule, matchedRuleToReplace].filter(
        (rule): rule is string => Boolean(rule) && rule !== desiredRule,
    );
    const hasRuleToReplace = rulesToReplace.some((rule) => currentRules.includes(rule));

    if (hasDesiredRule && !hasRuleToReplace) {
        return true;
    }

    const rulesToRemove = new Set([desiredRule, ...rulesToReplace]);
    const updatedRules = currentRules.filter((rule: string) => !rulesToRemove.has(rule));
    updatedRules.push(desiredRule);

    const didSave = await setRules(`${updatedRules.join('\n')}\n`);

    if (!didSave) {
        return false;
    }

    addSuccessToast(
        createUndoToast(
            intl.getMessage('user_rules_rule_added', { rule: desiredRule }),
            intl.getMessage('notify_undo'),
            async () => {
                const didUndo = await setRules(previousRules);
                if (didUndo) {
                    await getFilteringStatus();
                }
            },
        ),
    );
    await getFilteringStatus();
    return true;
};

export const toggleBlockingForClient = (type: BlockAction, domain: string, client: string) => {
    const escapedClientName = client
        .replace(/'/g, "\\'")
        .replace(/"/g, '\\"')
        .replace(/,/g, '\\,')
        .replace(/\|/g, '\\|');
    const baseRule = `||${domain}^$client='${escapedClientName}'`;
    const baseUnblocking = `@@${baseRule}`;

    return toggleBlocking(type, domain, baseRule, baseUnblocking);
};

export const addFilter = async (url: string, name: string, whitelist: boolean) => {
    setState('processingAddFilter', true);
    try {
        await apiClient.addFilter({ url, name, whitelist });
        setState('processingAddFilter', false);
        setState('isModalOpen', false);
        if (whitelist) {
            addSuccessToast({
                message: intl.getMessage('filter_added_successfully_allowlist', { value: name }),
            });
        } else {
            addSuccessToast({
                message: intl.getMessage('filter_added_successfully', { value: name }),
            });
        }
        await getFilteringStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingAddFilter', false);
    }
};

export const removeFilter = async (url: string, whitelist: boolean, name?: string) => {
    setState('processingRemoveFilter', true);
    try {
        await apiClient.removeFilter({ url, whitelist });
        setState('processingRemoveFilter', false);
        setState('isModalOpen', false);
        if (whitelist) {
            addSuccessToast({
                message: intl.getMessage('filter_removed_successfully_allowlist', { value: name }),
            });
        } else {
            addSuccessToast({
                message: intl.getMessage('filter_removed_successfully', { value: name }),
            });
        }
        await getFilteringStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingRemoveFilter', false);
    }
};

export const toggleFilterStatus = async (url: string, data: any, whitelist: boolean) => {
    setState('processingConfigFilter', true);
    try {
        await apiClient.setFilterUrl({ url, data, whitelist });
        setState('processingConfigFilter', false);
        await getFilteringStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfigFilter', false);
    }
};

export const editFilter = async (url: string, data: any, whitelist: boolean) => {
    setState('processingConfigFilter', true);
    try {
        await apiClient.setFilterUrl({ url, data, whitelist });
        setState({ processingConfigFilter: false, isModalOpen: false });
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getFilteringStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfigFilter', false);
    }
};

export const refreshFilters = async (config: any) => {
    setState('processingRefreshFilters', true);
    try {
        const data = await apiClient.refreshFilters(config);
        setState('processingRefreshFilters', false);
        const updated = (data as any)?.updated || 0;
        if (updated > 0) {
            addSuccessToast(intl.getPlural('list_updated', updated));
        } else {
            addSuccessToast(intl.getMessage('all_lists_up_to_date_toast'));
        }
        await getFilteringStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingRefreshFilters', false);
    }
};

export const setFiltersConfig = async (config: any) => {
    setState('processingSetConfig', true);
    try {
        await apiClient.setFiltersConfig(config);
        setState({ ...config, processingSetConfig: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
    }
};

export const checkHost = async (
    host: string | { name: string; client?: string; qtype?: string },
): Promise<boolean> => {
    setState('processingCheck', true);
    try {
        const data = await apiClient.checkHost(host);
        const hostname = typeof host === 'string' ? host : host.name;
        setState({ check: { hostname, ...data }, processingCheck: false });
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingCheck', false);
        return false;
    }
};

export const addFiltersBatch = async (
    filters: Array<{ url: string; name: string }>,
): Promise<void> => {
    setState('processingAddFilter', true);
    try {
        const results = await Promise.allSettled(
            filters.map(({ url, name }) => apiClient.addFilter({ url, name, whitelist: false })),
        );

        const successes: Array<{ url: string; name: string }> = [];
        const failures: Array<{ filter: { url: string; name: string }; error: unknown }> = [];

        results.forEach((result, index) => {
            if (result.status === 'fulfilled') {
                successes.push(filters[index]);
            } else {
                failures.push({ filter: filters[index], error: result.reason });
            }
        });

        if (successes.length === 1) {
            addSuccessToast({
                message: intl.getMessage('filter_added_successfully', {
                    value: successes[0].name || successes[0].url,
                }),
            });
        } else if (successes.length > 1) {
            addSuccessToast({
                message: intl.getMessage('filter_added_successfully_more', {
                    value: successes[0].name || successes[0].url,
                    more: String(successes.length - 1),
                }),
            });
        }

        failures.forEach(({ error }) => {
            addErrorToast({ error });
        });

        if (successes.length > 0) {
            await getFilteringStatus();
        }

        setState('processingAddFilter', false);
    } catch (error) {
        addErrorToast({ error });
        setState('processingAddFilter', false);
    }
};

export const toggleFilteringModal = (modalType?: string) => {
    if (modalType) {
        setState({ isModalOpen: !untrack(() => state.isModalOpen), modalType });
    } else {
        setState('isModalOpen', (prev) => !prev);
    }
};

export const setFilterModalUrl = (url: string) => {
    setState('modalFilterUrl', url);
};

export const handleRulesChange = (rules: string) => {
    setState('userRules', rules);
};

export const filteringState = untrack(() => state);
