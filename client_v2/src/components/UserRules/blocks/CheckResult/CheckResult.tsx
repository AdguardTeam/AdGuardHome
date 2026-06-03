import React from 'react';
import cn from 'clsx';
import { useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';
import { FILTERED_STATUS } from 'panel/helpers/constants';
import { getServiceName } from 'panel/helpers/helpers';
import { RootState } from 'panel/initialState';

import { getCheckResultMeta } from '../../checkResultHelpers';
import { CheckResultData, ResultActionKind } from '../../types';

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

export const CheckResult = ({
    checkResult,
    processingRules,
    onDismiss,
    onAction,
    onEditRewrite,
    onDeleteRewrite,
    hasMatchedRewrite = false,
    hiddenActionKinds = [],
}: Props) => {
    const filters = useSelector((state: RootState) => state.filtering.filters);
    const whitelistFilters = useSelector((state: RootState) => state.filtering.whitelistFilters);
    const allServices = useSelector((state: RootState) => state.services.allServices);

    const { hostname, reason, rules, service_name, cname, ip_addrs } = checkResult;

    if (!hostname) {
        return null;
    }

    const meta = getCheckResultMeta({ reason, rules, filters, whitelistFilters });
    const statusClassName = getStatusClassName(meta.tone);
    const showRewriteActions = reason === FILTERED_STATUS.REWRITE && hasMatchedRewrite;
    const showSource = Boolean(meta.source);
    const hasStandaloneResultMessage = reason ? STANDALONE_RESULT_REASONS.has(reason) : false;
    const redirectedValue = cname || (ip_addrs && ip_addrs.length > 0 ? ip_addrs.join(', ') : null);
    const normalizedServiceName = service_name
        ? getServiceName(allServices, service_name) || service_name
        : null;
    const hiddenActionKindSet = new Set(hiddenActionKinds);

    const getReasonContent = () => {
        if (hasStandaloneResultMessage) {
            return meta.reason;
        }

        if (meta.tone === 'rewritten') {
            if (reason === FILTERED_STATUS.REWRITE_RULE) {
                return intl.getMessage('user_rules_reason', { reason: meta.reason });
            }

            return intl.getMessage('user_rules_status', { reason: meta.reason });
        }

        if (meta.reason) {
            return intl.getMessage('user_rules_reason', { reason: meta.reason });
        }

        return null;
    };

    const reasonContent = getReasonContent();

    return (
        <div className={cn(s.checkResult, theme.text.t3)} data-testid="user-rules-result-card">
            <div className={s.checkResultHeader}>
                <h3
                    className={cn(
                        s.checkResultTitle,
                        theme.text.t3,
                        theme.text.semibold,
                        statusClassName,
                    )}
                    data-testid="user-rules-result-title"
                >
                    {meta.title}
                </h3>

                {onDismiss && (
                    <button
                        type="button"
                        className={s.dismissButton}
                        aria-label={intl.getMessage('close_result')}
                        data-testid="user-rules-result-dismiss"
                        onClick={onDismiss}
                    >
                        <Icon icon="cross" color="gray" />
                    </button>
                )}
            </div>

            <div className={s.checkResultItems}>
                <div className={s.resultItem}>
                    {intl.getMessage('user_rules_domain', { value: hostname })}
                </div>

                {reasonContent && <div className={s.resultItem}>{reasonContent}</div>}

                {showSource && meta.source && (
                    <div className={s.resultItem}>
                        {renderSourceLabel(meta.source, meta.sourceListType)}
                    </div>
                )}

                {normalizedServiceName && (
                    <div className={s.resultItem}>
                        {intl.getMessage('user_rules_service', { service: normalizedServiceName })}
                    </div>
                )}

                {meta.tone !== 'rewritten' && meta.rule && (
                    <div className={s.resultItem}>
                        {intl.getMessage('user_rules_rule', { rule: meta.rule })}
                    </div>
                )}

                {meta.tone === 'rewritten' && redirectedValue && (
                    <div className={s.resultItem}>
                        {reason === FILTERED_STATUS.FILTERED_SAFE_SEARCH
                            ? intl.getMessage('user_rules_redirected_to', {
                                  value: redirectedValue,
                              })
                            : intl.getMessage('user_rules_rewritten_to', {
                                  value: redirectedValue,
                              })}
                    </div>
                )}

                {meta.tone === 'rewritten' && meta.rule && (
                    <div className={s.resultItem}>
                        {intl.getMessage('user_rules_rule', { rule: meta.rule })}
                    </div>
                )}

                {meta.tone !== 'rewritten' && cname && (
                    <div className={s.resultItem}>
                        {intl.getMessage('user_rules_cname', { cname })}
                    </div>
                )}

                {meta.tone !== 'rewritten' && ip_addrs && ip_addrs.length > 0 && (
                    <div className={s.resultItem}>
                        {intl.getMessage('user_rules_ip', { ip: ip_addrs.join(', ') })}
                    </div>
                )}
            </div>

            <div className={s.actionButtons}>
                {meta.actions
                    .filter((action) => !hiddenActionKindSet.has(action.kind))
                    .map((action) => (
                        <button
                            key={action.kind}
                            type="button"
                            disabled={processingRules}
                            className={s.actionLink}
                            data-testid={`user-rules-result-action-${action.kind}`}
                            onClick={() => onAction(action.kind)}
                        >
                            {action.label}
                        </button>
                    ))}

                {showRewriteActions && (
                    <button
                        type="button"
                        disabled={processingRules}
                        className={s.actionLink}
                        data-testid="user-rules-result-action-edit-rewrite"
                        onClick={onEditRewrite}
                    >
                        {intl.getMessage('user_rules_edit_dns_rewrite')}
                    </button>
                )}

                {showRewriteActions && (
                    <button
                        type="button"
                        disabled={processingRules}
                        className={s.actionLink}
                        data-testid="user-rules-result-action-delete-rewrite"
                        onClick={onDeleteRewrite}
                    >
                        {intl.getMessage('user_rules_remove_dns_rewrite')}
                    </button>
                )}
            </div>
        </div>
    );
};
