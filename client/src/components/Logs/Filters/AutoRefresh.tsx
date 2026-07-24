import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { LOGS_AUTO_REFRESH_DEFAULT_INTERVAL_MS } from '../../../helpers/constants';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from '../../../helpers/localStorageHelper';

type Props = {
    refreshLogs: (silently?: boolean) => Promise<void>;
};

const AutoRefresh = ({ refreshLogs }: Props) => {
    const { t } = useTranslation();

    const [isAutoRefreshEnabled, setIsAutoRefreshEnabled] = useState<boolean>(
        () => !!LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_ENABLED),
    );

    const [intervalMs] = useState<number>(
        () =>
            Number(LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_INTERVAL_MS)) ||
            LOGS_AUTO_REFRESH_DEFAULT_INTERVAL_MS,
    );

    const toggleAutoRefresh = () => {
        setIsAutoRefreshEnabled((prevEnabled) => {
            const nextEnabled = !prevEnabled;
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_ENABLED, nextEnabled);
            return nextEnabled;
        });
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

    return (
        <label className="custom-switch logs__auto-refresh" title={t('auto_refresh')}>
            <input
                type="checkbox"
                className="custom-switch-input"
                checked={isAutoRefreshEnabled}
                onChange={toggleAutoRefresh}
            />

            <span className="custom-switch-indicator" />

            <span className="custom-switch-description">{t('auto_refresh')}</span>
        </label>
    );
};

export default AutoRefresh;
