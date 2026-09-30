import React, { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { LOGS_AUTO_REFRESH_DEFAULT_INTERVAL_MS, LOGS_AUTO_REFRESH_INTERVALS_MS } from '../../../helpers/constants';
import { msToMinutes, msToSeconds } from '../../../helpers/helpers';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from '../../../helpers/localStorageHelper';

const getIntervalLabel = (t: (key: string, options?: Record<string, unknown>) => string, intervalMs: number) => {
    if (intervalMs < 60 * 1000) {
        return t('auto_refresh_seconds', { count: msToSeconds(intervalMs) });
    }

    return t('auto_refresh_minutes', { count: msToMinutes(intervalMs) });
};

const AutoRefreshInterval = () => {
    const { t } = useTranslation();

    const [intervalMs, setIntervalMs] = useState<number>(
        () =>
            Number(LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_INTERVAL_MS)) ||
            LOGS_AUTO_REFRESH_DEFAULT_INTERVAL_MS,
    );

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const nextIntervalMs = Number(e.target.value);
        setIntervalMs(nextIntervalMs);
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LOGS_AUTO_REFRESH_INTERVAL_MS, nextIntervalMs);
    };

    return (
        <div className="form__group form__group--settings">
            <div className="form__label">
                <Trans>auto_refresh_interval</Trans>
            </div>

            <div className="custom-controls-stacked">
                {LOGS_AUTO_REFRESH_INTERVALS_MS.map((interval) => (
                    <label key={interval} className="custom-control custom-radio">
                        <input
                            type="radio"
                            className="custom-control-input"
                            data-testid={`logs_auto_refresh_interval_${interval}`}
                            value={interval}
                            checked={intervalMs === interval}
                            onChange={handleChange}
                        />

                        <span className="custom-control-label">{getIntervalLabel(t, interval)}</span>
                    </label>
                ))}
            </div>
        </div>
    );
};

export default AutoRefreshInterval;
