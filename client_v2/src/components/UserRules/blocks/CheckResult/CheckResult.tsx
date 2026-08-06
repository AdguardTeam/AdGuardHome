import { createMemo, Show, For } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';
import { filteringState } from 'panel/stores/filtering';
import { servicesState } from 'panel/stores/services';
import { FILTERED_STATUS } from 'panel/helpers/constants';
import { getServiceName } from 'panel/helpers/helpers';

import { getCheckResultMeta } from '../../checkResultHelpers';
import { type CheckResultData, type ResultActionKind } from '../../types';

import s from './CheckResult.module.pcss';

const STANDALONE_RESULT_REASONS = new Set([
    FILTERED_STATUS.NOT_FILTERED_NOT_FOUND,
    FILTERED_STATUS.NOT_FILTERED_ERROR,
    FILTERED_STATUS.FILTERED_INVALID,
]);

type Props = {
    checkResult: CheckResultData;
    processingRules: boolean;
    onDismiss?: () => void;
    onAction: (action: ResultActionKind) => void;
    onEditRewrite: () => void;
    onDeleteRewrite: () => void;
    hasMatchedRewrite?: boolean;
    hiddenActionKinds?: ResultActionKind[];
};

const renderSourceLabel = (source: string, sourceListType?: 'blocklist' | 'allowlist') => {
    if (sourceListType === 'blocklist') {
        return intl.getMessage('user_rules_source_blocklist', { value: source });
    }

    if (sourceListType === 'allowlist') {
        return intl.getMessage('user_rules_source_allowlist', { value: source });
    }

    return intl.getMessage('user_rules_source', { value: source });
};

type CheckResultTone = ReturnType<typeof getCheckResultMeta>['tone'];

const getStatusClassName = (tone: CheckResultTone) => {
    if (tone === 'blocked') {
        return s.checkResultTitleBlocked;
    }

    if (tone === 'allowed') {
        return s.checkResultTitleAllowed;
    }

    if (tone === 'rewritten') {
        return s.checkResultTitleRewritten;
    }

    return '';
};

export const CheckResult = (props: Props) => {
    const meta = createMemo(() =>
        getCheckResultMeta({
            reason: props.checkResult.reason,
            rules: props.checkResult.rules,
            filters: filteringState.filters,
            whitelistFilters: filteringState.whitelistFilters,
        }),
    );

    const statusClassName = createMemo(() => getStatusClassName(meta().tone));
    const showRewriteActions = () =>
        props.checkResult.reason === FILTERED_STATUS.REWRITE && props.hasMatchedRewrite;
    const showSource = () => Boolean(meta().source);
    const hasStandaloneResultMessage = () =>
        props.checkResult.reason ? STANDALONE_RESULT_REASONS.has(props.checkResult.reason) : false;
    const redirectedValue = () =>
        props.checkResult.cname ||
        (props.checkResult.ip_addrs && props.checkResult.ip_addrs.length > 0
            ? props.checkResult.ip_addrs.join(', ')
            : null);
    const normalizedServiceName = () =>
        props.checkResult.service_name
            ? getServiceName(servicesState.allServices, props.checkResult.service_name) ||
              props.checkResult.service_name
            : null;
    const hiddenActionKindSet = () => new Set(props.hiddenActionKinds || []);

    const reasonContent = () => {
        if (hasStandaloneResultMessage()) {
            return meta().reason;
        }

        if (meta().tone === 'rewritten') {
            if (props.checkResult.reason === FILTERED_STATUS.REWRITE_RULE) {
                return intl.getMessage('user_rules_reason', { reason: meta().reason });
            }
            return intl.getMessage('user_rules_status', { reason: meta().reason });
        }

        if (meta().reason) {
            return intl.getMessage('user_rules_reason', { reason: meta().reason });
        }

        return null;
    };

    return (
        <Show when={props.checkResult.hostname}>
            <div class={cn(s.checkResult, theme.text.t3)} data-testid="user-rules-result-card">
                <div class={s.checkResultHeader}>
                    <h3
                        class={cn(
                            s.checkResultTitle,
                            theme.text.t3,
                            theme.text.semibold,
                            statusClassName(),
                        )}
                        data-testid="user-rules-result-title"
                    >
                        {meta().title}
                    </h3>

                    <Show when={props.onDismiss}>
                        <button
                            type="button"
                            class={s.dismissButton}
                            aria-label={intl.getMessage('close_result')}
                            data-testid="user-rules-result-dismiss"
                            onClick={() => props.onDismiss?.()}
                        >
                            <Icon icon="cross" color="gray" />
                        </button>
                    </Show>
                </div>

                <div class={s.checkResultItems}>
                    <div class={s.resultItem}>
                        {intl.getMessage('user_rules_domain', {
                            value: props.checkResult.hostname,
                        })}
                    </div>

                    <Show when={reasonContent()}>
                        <div class={s.resultItem}>{reasonContent()}</div>
                    </Show>

                    <Show when={showSource() && meta().source}>
                        <div class={s.resultItem}>
                            {renderSourceLabel(meta().source!, meta().sourceListType)}
                        </div>
                    </Show>

                    <Show when={normalizedServiceName()}>
                        <div class={s.resultItem}>
                            {intl.getMessage('user_rules_service', {
                                service: normalizedServiceName(),
                            })}
                        </div>
                    </Show>

                    <Show when={meta().tone !== 'rewritten' && meta().rule}>
                        <div class={s.resultItem}>
                            {intl.getMessage('user_rules_rule', { rule: meta().rule })}
                        </div>
                    </Show>

                    <Show when={meta().tone === 'rewritten' && redirectedValue()}>
                        <div class={s.resultItem}>
                            {props.checkResult.reason === FILTERED_STATUS.FILTERED_SAFE_SEARCH
                                ? intl.getMessage('user_rules_redirected_to', {
                                      value: redirectedValue(),
                                  })
                                : intl.getMessage('user_rules_rewritten_to', {
                                      value: redirectedValue(),
                                  })}
                        </div>
                    </Show>

                    <Show when={meta().tone === 'rewritten' && meta().rule}>
                        <div class={s.resultItem}>
                            {intl.getMessage('user_rules_rule', { rule: meta().rule })}
                        </div>
                    </Show>

                    <Show when={meta().tone !== 'rewritten' && props.checkResult.cname}>
                        <div class={s.resultItem}>
                            {intl.getMessage('user_rules_cname', {
                                cname: props.checkResult.cname,
                            })}
                        </div>
                    </Show>

                    <Show
                        when={
                            meta().tone !== 'rewritten' &&
                            props.checkResult.ip_addrs &&
                            props.checkResult.ip_addrs!.length > 0
                        }
                    >
                        <div class={s.resultItem}>
                            {intl.getMessage('user_rules_ip', {
                                ip: props.checkResult.ip_addrs!.join(', '),
                            })}
                        </div>
                    </Show>
                </div>

                <div class={s.actionButtons}>
                    <For
                        each={meta().actions.filter(
                            (action) => !hiddenActionKindSet().has(action.kind),
                        )}
                    >
                        {(action) => (
                            <button
                                type="button"
                                disabled={props.processingRules}
                                class={s.actionLink}
                                data-testid={`user-rules-result-action-${action.kind}`}
                                onClick={() => props.onAction(action.kind)}
                            >
                                {action.label}
                            </button>
                        )}
                    </For>

                    <Show when={showRewriteActions()}>
                        <button
                            type="button"
                            disabled={props.processingRules}
                            class={s.actionLink}
                            data-testid="user-rules-result-action-edit-rewrite"
                            onClick={() => props.onEditRewrite?.()}
                        >
                            {intl.getMessage('user_rules_edit_dns_rewrite')}
                        </button>
                    </Show>

                    <Show when={showRewriteActions()}>
                        <button
                            type="button"
                            disabled={props.processingRules}
                            class={s.actionLink}
                            data-testid="user-rules-result-action-delete-rewrite"
                            onClick={() => props.onDeleteRewrite?.()}
                        >
                            {intl.getMessage('user_rules_remove_dns_rewrite')}
                        </button>
                    </Show>
                </div>
            </div>
        </Show>
    );
};
