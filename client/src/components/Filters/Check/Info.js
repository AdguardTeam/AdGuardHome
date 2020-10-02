import React from 'react';
import { useTranslation } from 'react-i18next';
import classNames from 'classnames';

import i18next from 'i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import {
    checkFiltered,
    checkRewrite,
    checkRewriteHosts,
    checkWhiteList,
    checkSafeSearch,
    checkSafeBrowsing,
    checkParental,
    getFilterName,
} from '../../../helpers/helpers';
import { BLOCK_ACTIONS, FILTERED, FILTERED_STATUS } from '../../../helpers/constants';
import { toggleBlocking } from '../../../actions';

const renderBlockingButton = (isFiltered, domain) => {
    const processingRules = useSelector((state) => state.filtering.processingRules);
    const dispatch = useDispatch();
    const { t } = useTranslation();

    const buttonType = isFiltered ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;

    const onClick = async () => {
        await dispatch(toggleBlocking(buttonType, domain));
    };

    const buttonClass = classNames('mt-3 button-action button-action--main button-action--active button-action--small', {
        'button-action--unblock': isFiltered,
    });

    return <button type="button"
                className={buttonClass}
                onClick={onClick}
                disabled={processingRules}
        >
            {t(buttonType)}
        </button>;
};

const getTitle = (reason) => {
    const { t } = useTranslation();

    const filters = useSelector((state) => state.filtering.filters, shallowEqual);
    const whitelistFilters = useSelector((state) => state.filtering.whitelistFilters, shallowEqual);
    const filter_id = useSelector((state) => state.filtering.check.filter_id);

    const filterName = getFilterName(
        filters,
        whitelistFilters,
        filter_id,
        'filtered_custom_rules',
        (filter) => (filter?.name ? t('query_log_filtered', { filter: filter.name }) : ''),
    );

    const getReasonFiltered = (reason) => {
        const filterKey = reason.replace(FILTERED, '');
        return i18next.t('query_log_filtered', { filter: filterKey });
    };

    const REASON_TO_TITLE_MAP = {
        [FILTERED_STATUS.NOT_FILTERED_NOT_FOUND]: t('check_not_found'),
        [FILTERED_STATUS.REWRITE]: t('rewrite_applied'),
        [FILTERED_STATUS.REWRITE_HOSTS]: t('rewrite_hosts_applied'),
        [FILTERED_STATUS.FILTERED_BLACK_LIST]: filterName,
        [FILTERED_STATUS.NOT_FILTERED_WHITE_LIST]: filterName,
        [FILTERED_STATUS.FILTERED_SAFE_SEARCH]: getReasonFiltered(reason),
        [FILTERED_STATUS.FILTERED_SAFE_BROWSING]: getReasonFiltered(reason),
        [FILTERED_STATUS.FILTERED_PARENTAL]: getReasonFiltered(reason),
    };

    if (Object.prototype.hasOwnProperty.call(REASON_TO_TITLE_MAP, reason)) {
        return REASON_TO_TITLE_MAP[reason];
    }

    return <>
        <div>{t('check_reason', { reason })}</div>
        <div>{filterName}</div>
    </>;
};

const Info = () => {
    const {
        hostname,
        reason,
        rule,
        service_name,
        cname,
        ip_addrs,
    } = useSelector((state) => state.filtering.check, shallowEqual);
    const { t } = useTranslation();

    const title = getTitle(reason);

    const className = classNames('card mb-0 p-3', {
        'logs__row--red': checkFiltered(reason),
        'logs__row--blue': checkRewrite(reason) || checkRewriteHosts(reason),
        'logs__row--green': checkWhiteList(reason),
    });

    const onlyFiltered = checkSafeSearch(reason)
        || checkSafeBrowsing(reason)
        || checkParental(reason);

    const isFiltered = checkFiltered(reason);

    return <div className={className}>
        <div><strong>{hostname}</strong></div>
        <div>{title}</div>
        {!onlyFiltered
        && <>
            {rule && <div>{t('check_rule', { rule })}</div>}
            {service_name && <div>{t('check_service', { service: service_name })}</div>}
            {cname && <div>{t('check_cname', { cname })}</div>}
            {ip_addrs && <div>{t('check_ip', { ip: ip_addrs.join(', ') })}</div>}
            {renderBlockingButton(isFiltered, hostname)}
        </>}
    </div>;
};

export default Info;
