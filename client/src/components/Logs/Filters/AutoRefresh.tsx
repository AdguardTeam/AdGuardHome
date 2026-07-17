import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import classNames from 'classnames';

import Dropdown from '../../ui/Dropdown';
import { LOGS_AUTO_REFRESH_DEFAULT_INTERVAL_MS, LOGS_AUTO_REFRESH_INTERVALS_MS } from '../../../helpers/constants';
import { msToMinutes, msToSeconds } from '../../../helpers/helpers';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from '../../../helpers/localStorageHelper';

type Props = {
    refreshLogs: (silently?: boolean) => Promise<void>;
};

const getIntervalLabel = (t: (key: string, options?: Record<string, unknown>) => string, intervalMs: number) => {
    if (intervalMs < 60 * 1000) {
        return t('auto_refresh_seconds', { count: msToSeconds(intervalMs) });
    }

    return t('auto_refresh_minutes', { count: msToMinutes(intervalMs) });
};

const AutoRefresh = ({ refreshLogs }: Props) => {
    const { t } = useTranslation();

    const [isAutoRefreshEnabled, setIsAutoRefreshEnabled] = useState<boolean>(
        () => !!LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_ENABLED),
    );

    const [intervalMs, setIntervalMs] = useState<number>(
        () =>
            Number(LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_INTERVAL_MS)) ||
            LOGS_AUTO_REFRESH_DEFAULT_INTERVAL_MS,
    );

    const setAutoRefreshEnabled = (enabled: boolean) => {
        setIsAutoRefreshEnabled(enabled);
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_ENABLED, enabled);
    };

    const handleSelectInterval = (nextIntervalMs: number) => {
        setIntervalMs(nextIntervalMs);
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_INTERVAL_MS, nextIntervalMs);
        setAutoRefreshEnabled(true);
    };

    useEffect(() => {
        if (!isAutoRefreshEnabled) {
            return undefined;
        }

        const interval = setInterval(() => {
            refreshLogs(true);
        }, intervalMs);

        return () => {
            clearInterval(interval);
        };
    }, [isAutoRefreshEnabled, intervalMs]);

    const colorClass = isAutoRefreshEnabled ? 'btn-success' : 'logs__refresh--off';

    return (
        <div className="logs__refresh-group">
            <button
                type="button"
                className={classNames('btn btn-sm logs__refresh-button', colorClass)}
                title={t('refresh_btn')}
                onClick={() => refreshLogs()}>
                <svg className="icons icon12">
                    <use xlinkHref="#update" />
                </svg>
                {t('refresh_btn')}
            </button>

            <Dropdown
                label=""
                baseClassName="logs__refresh-dropdown"
                controlClassName={classNames('btn btn-sm logs__refresh-toggle', colorClass)}
                menuClassName="dropdown-menu logs__refresh-menu">
                <div className="dropdown-header logs__refresh-status">
                    {t('automatic_refresh')}: {isAutoRefreshEnabled ? t('on') : t('off')}
                </div>

                <div className="dropdown-divider" />

                <div
                    className={classNames('dropdown-item', { active: !isAutoRefreshEnabled })}
                    onClick={() => setAutoRefreshEnabled(false)}>
                    {t('off')}
                </div>

                {LOGS_AUTO_REFRESH_INTERVALS_MS.map((interval) => (
                    <div
                        key={interval}
                        className={classNames('dropdown-item', {
                            active: isAutoRefreshEnabled && intervalMs === interval,
                        })}
                        onClick={() => handleSelectInterval(interval)}>
                        {getIntervalLabel(t, interval)}
                    </div>
                ))}
            </Dropdown>
        </div>
    );
};

export default AutoRefresh;
