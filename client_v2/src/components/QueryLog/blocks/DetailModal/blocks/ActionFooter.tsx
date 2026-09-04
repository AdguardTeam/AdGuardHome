import { Show, For } from 'solid-js';

import intl from 'panel/common/intl';
import { Button, type ButtonVariant } from 'panel/common/ui/Button';
import { type Filter } from 'panel/helpers/helpers';
import type { NormalizedQueryLogItem } from 'panel/helpers/helpers';
import { findRewriteRuleByDomain } from 'panel/stores/rewrites';
import type { RewriteEntry } from 'panel/api/model/rewriteEntry';
import { getQueryReasonKey } from 'panel/components/QueryLog/helpers';

import { getDetailModalActions, type DetailModalActionId } from '../actions';

import s from '../DetailModal.module.pcss';

type Props = {
    entry: NormalizedQueryLogItem;
    filters: Filter[];
    onClose: () => void;
    onBlock: (domain: string) => void;
    onAddToAllowlist: (domain: string) => void;
    onAllowService: (serviceId: string) => void;
    onDisableFilter: (filter: Filter) => void;
    onDisableSafeBrowsing: () => void;
    onDisableParental: () => void;
    onDisableSafeSearch: () => void;
    onRemoveRewrite: (rewrite: RewriteEntry) => void;
    onEditRewrite: (rewrite: RewriteEntry) => void;
};

export const ActionFooter = (props: Props) => {
    const reasonKey = () => getQueryReasonKey(props.entry.reason, props.entry.rules ?? []);
    const serviceId = () => props.entry.serviceName || props.entry.service_name;
    const filterToDisable = () => {
        const filterListId = (props.entry.rules ?? []).find(
            ({ filter_list_id }) => filter_list_id != null,
        )?.filter_list_id;
        if (filterListId == null) {
            return undefined;
        }
        return props.filters.find((f) => f.id === filterListId);
    };
    const rewriteRule = () => findRewriteRuleByDomain(props.entry.domain);
    const actions = () =>
        getDetailModalActions(reasonKey(), {
            hasServiceId: !!serviceId(),
            canDisableFilter: !!filterToDisable(),
            hasRewriteRule: !!rewriteRule(),
        });

    const handleBlock = () => {
        props.onBlock(props.entry.domain);
        props.onClose();
    };

    const handleAddToAllowlist = () => {
        props.onAddToAllowlist(props.entry.domain);
        props.onClose();
    };

    const handleAllowService = () => {
        const sid = serviceId();
        if (!sid) {
            return;
        }
        props.onAllowService(sid);
        props.onClose();
    };

    const handleDisableFilter = () => {
        const filter = filterToDisable();
        if (!filter) return;
        props.onDisableFilter(filter);
        props.onClose();
    };

    const handleDisableSafeBrowsing = () => {
        props.onDisableSafeBrowsing();
        props.onClose();
    };

    const handleDisableParental = () => {
        props.onDisableParental();
        props.onClose();
    };

    const handleDisableSafeSearch = () => {
        props.onDisableSafeSearch();
        props.onClose();
    };

    const handleRemoveRewrite = () => {
        const rewrite = rewriteRule();
        if (!rewrite) return;
        props.onRemoveRewrite(rewrite);
        props.onClose();
    };

    const handleEditRewrite = () => {
        const rewrite = rewriteRule();
        if (!rewrite) return;
        props.onEditRewrite(rewrite);
        props.onClose();
    };

    type ActionConfig = {
        variant: ButtonVariant;
        secondaryVariant?: ButtonVariant;
        testId: string;
        dataAction: string;
        labelKey: string;
        onClick: () => void;
    };

    const ACTION_CONFIG: Record<DetailModalActionId, ActionConfig> = {
        block: {
            variant: 'danger',
            secondaryVariant: 'secondary-danger',
            testId: 'query-log-detail-action-block',
            dataAction: 'block',
            labelKey: 'block',
            onClick: handleBlock,
        },
        'add-to-allowlist': {
            variant: 'primary',
            testId: 'query-log-detail-action-allowlist',
            dataAction: 'allowlist',
            labelKey: 'user_rules_add_to_allowlist',
            onClick: handleAddToAllowlist,
        },
        'allow-service': {
            variant: 'secondary',
            testId: 'query-log-detail-action-allow-service',
            dataAction: 'allow-service',
            labelKey: 'user_rules_allow_service',
            onClick: handleAllowService,
        },
        'disable-filter': {
            variant: 'secondary',
            testId: 'query-log-detail-action-disable-filter',
            dataAction: 'disable-filter',
            labelKey: 'user_rules_disable_filter',
            onClick: handleDisableFilter,
        },
        'disable-browsing-security': {
            variant: 'secondary',
            testId: 'query-log-detail-action-disable-browsing-security',
            dataAction: 'disable-browsing-security',
            labelKey: 'user_rules_disable_browsing_security',
            onClick: handleDisableSafeBrowsing,
        },
        'disable-parental': {
            variant: 'secondary',
            testId: 'query-log-detail-action-disable-parental',
            dataAction: 'disable-parental',
            labelKey: 'user_rules_disable_parental_control',
            onClick: handleDisableParental,
        },
        'disable-safe-search': {
            variant: 'secondary',
            testId: 'query-log-detail-action-disable-safe-search',
            dataAction: 'disable-safe-search',
            labelKey: 'user_rules_disable_safe_search',
            onClick: handleDisableSafeSearch,
        },
        'remove-dns-rewrite': {
            variant: 'primary',
            testId: 'query-log-detail-action-remove-dns-rewrite',
            dataAction: 'remove-dns-rewrite',
            labelKey: 'user_rules_remove_dns_rewrite',
            onClick: handleRemoveRewrite,
        },
        'edit-dns-rewrite': {
            variant: 'secondary',
            testId: 'query-log-detail-action-edit-dns-rewrite',
            dataAction: 'edit-dns-rewrite',
            labelKey: 'user_rules_edit_dns_rewrite',
            onClick: handleEditRewrite,
        },
    };

    return (
        <Show when={actions().length > 0}>
            <div class={s.actionFooter} data-testid="query-log-detail-action-footer">
                <For each={actions()}>
                    {(actionId, index) => {
                        const config = ACTION_CONFIG[actionId];
                        return (
                            <Button
                                data-testid={config.testId}
                                data-action={config.dataAction}
                                type="button"
                                variant={
                                    index() > 0 && config.secondaryVariant
                                        ? config.secondaryVariant
                                        : config.variant
                                }
                                size="small"
                                compact
                                class={s.actionButton}
                                onClick={config.onClick}
                            >
                                {intl.getMessage(config.labelKey)}
                            </Button>
                        );
                    }}
                </For>
            </div>
        </Show>
    );
};
