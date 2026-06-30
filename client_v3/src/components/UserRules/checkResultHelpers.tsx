import intl from 'panel/common/intl';
import { FILTERED_STATUS, SPECIAL_FILTER_ID } from 'panel/helpers/constants';
import { checkFiltered, getFilterName, type Filter } from 'panel/helpers/helpers';

import { CheckResultData, ResultAction, ResultActionKind } from './types';

type CheckResultRule = NonNullable<CheckResultData['rules']>[number];

type CheckResultMeta = {
    tone: 'blocked' | 'allowed' | 'rewritten' | 'processed';
    title: string;
    reason?: string;
    actions: ResultAction[];
    rule?: string;
    source?: string;
    sourceListType?: 'blocklist' | 'allowlist';
};

const getActionLabel = (action: ResultActionKind) => {
    switch (action) {
        case 'allow':
            return intl.getMessage('user_rules_add_to_allowlist');
        case 'block':
            return intl.getMessage('block');
        case 'disable-parental':
            return intl.getMessage('user_rules_disable_parental_control');
        case 'disable-safebrowsing':
            return intl.getMessage('user_rules_disable_browsing_security');
        case 'disable-safesearch':
            return intl.getMessage('user_rules_disable_safe_search');
        case 'disable-blocked-service':
            return intl.getMessage('user_rules_allow_service');
        case 'disable-filter':
            return intl.getMessage('user_rules_disable_filter');
        case 'edit-rewrite':
            return intl.getMessage('user_rules_edit_dns_rewrite');
        case 'delete-rewrite':
        case 'remove-rewrite-rule':
            return intl.getMessage('user_rules_remove_dns_rewrite');
        default:
            return '';
    }
};

const createAction = (kind: ResultActionKind): ResultAction => ({
    kind,
    label: getActionLabel(kind),
});

const getPrimaryRule = (rules?: CheckResultData['rules']): CheckResultRule | undefined =>
    rules?.[0];

const getSourceName = (
    rules: CheckResultData['rules'],
    filters: Filter[],
    whitelistFilters: Filter[],
) => {
    const primaryRule = getPrimaryRule(rules);
    const filterListId = primaryRule?.filter_list_id;

    if (filterListId === undefined) {
        return undefined;
    }

    if (Object.values(SPECIAL_FILTER_ID).includes(filterListId)) {
        return getFilterName(filters, whitelistFilters, filterListId);
    }

    return (
        filters.find((filter) => filter.id === filterListId)?.name ||
        whitelistFilters.find((filter) => filter.id === filterListId)?.name
    );
};

const getSourceListType = (
    filterListId: number | undefined,
    filters: Filter[],
    whitelistFilters: Filter[],
): 'blocklist' | 'allowlist' | undefined => {
    if (filterListId === undefined) {
        return undefined;
    }

    const specialIds = Object.values(SPECIAL_FILTER_ID) as number[];
    if (specialIds.includes(filterListId)) {
        return undefined;
    }

    if (whitelistFilters.some((f) => f.id === filterListId)) {
        return 'allowlist';
    }

    if (filters.some((f) => f.id === filterListId)) {
        return 'blocklist';
    }

    return undefined;
};

const getBlockedReason = (
    isCustomRule: boolean,
    sourceName: string | undefined,
    sourceListType: 'blocklist' | 'allowlist' | undefined,
) => {
    if (isCustomRule) {
        return intl.getMessage('user_rules_reason_blocked_by', {
            source: intl.getMessage('custom_filtering_rules'),
        });
    }

    if (sourceListType) {
        return undefined;
    }

    if (!sourceName) {
        return undefined;
    }

    return intl.getMessage('user_rules_reason_filtered_by', { source: sourceName });
};

const getAllowedReason = (
    isCustomRule: boolean,
    sourceName: string | undefined,
    sourceListType: 'blocklist' | 'allowlist' | undefined,
) => {
    if (isCustomRule || sourceListType) {
        return undefined;
    }

    if (!sourceName) {
        return undefined;
    }

    return intl.getMessage('user_rules_reason_allowed_by', { source: sourceName });
};

export const getCheckResultMeta = ({
    reason,
    rules,
    filters,
    whitelistFilters,
}: {
    reason?: string;
    rules?: CheckResultData['rules'];
    filters: Filter[];
    whitelistFilters: Filter[];
}): CheckResultMeta => {
    const primaryRule = getPrimaryRule(rules);
    const sourceName = getSourceName(rules, filters, whitelistFilters);
    const isCustomRule = primaryRule?.filter_list_id === SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES;
    const sourceListType = getSourceListType(
        primaryRule?.filter_list_id,
        filters,
        whitelistFilters,
    );

    switch (reason) {
        case FILTERED_STATUS.FILTERED_BLACK_LIST:
            return {
                tone: 'blocked',
                title: intl.getMessage('user_rules_domain_blocked'),
                reason: getBlockedReason(isCustomRule, sourceName, sourceListType),
                actions: isCustomRule
                    ? [createAction('allow')]
                    : [createAction('allow'), createAction('disable-filter')],
                rule: primaryRule?.text,
                source: sourceName,
                sourceListType,
            };
        case FILTERED_STATUS.NOT_FILTERED_WHITE_LIST:
            return {
                tone: 'allowed',
                title: intl.getMessage('user_rules_domain_is_allowed'),
                reason: getAllowedReason(isCustomRule, sourceName, sourceListType),
                actions: isCustomRule ? [createAction('block')] : [createAction('disable-filter')],
                rule: primaryRule?.text,
                source: sourceName,
                sourceListType,
            };
        case FILTERED_STATUS.NOT_FILTERED_NOT_FOUND:
        case FILTERED_STATUS.NOT_FILTERED_ERROR:
        case FILTERED_STATUS.FILTERED_INVALID:
            return {
                tone: 'processed',
                title: intl.getMessage('user_rules_domain_is_processed'),
                reason: intl.getMessage('user_rules_no_rules_matched'),
                actions: [createAction('allow'), createAction('block')],
            };
        case FILTERED_STATUS.FILTERED_SAFE_BROWSING:
            return {
                tone: 'blocked',
                title: intl.getMessage('user_rules_domain_blocked'),
                reason: intl.getMessage('blocked_threats'),
                actions: [createAction('allow'), createAction('disable-safebrowsing')],
                rule: primaryRule?.text,
                source: intl.getMessage('safe_browsing'),
            };
        case FILTERED_STATUS.FILTERED_PARENTAL:
            return {
                tone: 'blocked',
                title: intl.getMessage('user_rules_domain_blocked'),
                reason: intl.getMessage('user_rules_reason_blocked_by', {
                    source: intl.getMessage('parental_control'),
                }),
                actions: [createAction('allow'), createAction('disable-parental')],
            };
        case FILTERED_STATUS.FILTERED_SAFE_SEARCH:
            return {
                tone: 'rewritten',
                title: intl.getMessage('user_rules_rewrite_rule_is_applied'),
                reason: intl.getMessage('settings_safe_search'),
                actions: [createAction('allow'), createAction('disable-safesearch')],
            };
        case FILTERED_STATUS.FILTERED_BLOCKED_SERVICE:
            return {
                tone: 'blocked',
                title: intl.getMessage('user_rules_domain_blocked'),
                reason: intl.getMessage('blocked_services'),
                actions: [createAction('allow'), createAction('disable-blocked-service')],
                rule: primaryRule?.text,
            };
        case FILTERED_STATUS.REWRITE:
            return {
                tone: 'rewritten',
                title: intl.getMessage('user_rules_rewrite_rule_is_applied'),
                reason: intl.getMessage('rewritten'),
                actions: [],
            };
        case FILTERED_STATUS.REWRITE_RULE:
            return {
                tone: 'rewritten',
                title: intl.getMessage('user_rules_rewrite_rule_is_applied'),
                reason: intl.getMessage('custom_filtering_rules'),
                actions: isCustomRule ? [createAction('remove-rewrite-rule')] : [],
                rule: primaryRule?.text,
            };
        case FILTERED_STATUS.REWRITE_HOSTS:
            return {
                tone: 'rewritten',
                title: intl.getMessage('user_rules_rewrite_rule_is_applied'),
                reason: intl.getMessage('rewrite_hosts_applied'),
                actions: [],
                source: intl.getMessage('system_host_files'),
            };
        default: {
            const isFilteredReason = reason ? checkFiltered(reason) : false;

            return {
                tone: isFilteredReason ? 'blocked' : 'processed',
                title: isFilteredReason
                    ? intl.getMessage('user_rules_domain_blocked')
                    : intl.getMessage('user_rules_domain_is_processed'),
                reason: reason || intl.getMessage('check_not_found'),
                actions: [],
                rule: primaryRule?.text,
                source: sourceName,
                sourceListType,
            };
        }
    }
};
