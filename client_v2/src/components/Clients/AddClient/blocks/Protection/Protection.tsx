import React, { useCallback } from 'react';
import type { ChangeEvent } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';
import { RootState } from 'panel/initialState';
import { updateClientFormField } from 'panel/actions/clientForm';
import {
    SAFE_SEARCH_PROVIDERS,
    SAFE_SEARCH_PROVIDER_KEYS,
    SafeSearchProviderKey,
} from 'panel/helpers/constants';
import theme from 'panel/lib/theme';

import { ClientsHeader } from '../ClientsHeader';

import s from './Protection.module.pcss';

export const Protection = () => {
    const dispatch = useDispatch();
    const form = useSelector((state: RootState) => state.clientForm);
    const disabled = form.use_global_settings;

    const handleToggle = useCallback(
        (field: string) => (e: ChangeEvent<HTMLInputElement>) => {
            dispatch(updateClientFormField({ field, value: e.target.checked }));
        },
        [dispatch],
    );

    const handleSafeSearchEnabled = useCallback(
        (e: ChangeEvent<HTMLInputElement>) => {
            const enabled = e.target.checked;
            const providersUpdate: Record<string, boolean> = {};
            if (enabled) {
                SAFE_SEARCH_PROVIDER_KEYS.forEach((key) => {
                    providersUpdate[key] = true;
                });
            }
            dispatch(
                updateClientFormField({
                    field: 'safe_search',
                    value: { ...form.safe_search, enabled, ...providersUpdate },
                }),
            );
        },
        [dispatch, form.safe_search],
    );

    const handleSafeSearchProvider = useCallback(
        (provider: SafeSearchProviderKey) => (e: ChangeEvent<HTMLInputElement>) => {
            dispatch(
                updateClientFormField({
                    field: 'safe_search',
                    value: { ...form.safe_search, [provider]: e.target.checked },
                }),
            );
        },
        [dispatch, form.safe_search],
    );

    return (
        <div className={cn(theme.layout.container, s.containerOverride)}>
            <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <ClientsHeader currentTitle={intl.getMessage('clients_protection')} />

                <SwitchGroup
                    id="filtering-enabled"
                    title={intl.getMessage('settings_filter_requests')}
                    checked={form.filtering_enabled}
                    onChange={handleToggle('filtering_enabled')}
                    disabled={disabled}
                />

                <SwitchGroup
                    id="safebrowsing-enabled"
                    title={intl.getMessage('settings_browsing_security')}
                    checked={form.safebrowsing_enabled}
                    onChange={handleToggle('safebrowsing_enabled')}
                    disabled={disabled}
                />

                <SwitchGroup
                    id="parental-enabled"
                    title={intl.getMessage('settings_parental_control')}
                    checked={form.parental_enabled}
                    onChange={handleToggle('parental_enabled')}
                    disabled={disabled}
                />

                <SwitchGroup
                    id="safe-search"
                    title={intl.getMessage('settings_safe_search')}
                    checked={form.safe_search.enabled}
                    onChange={handleSafeSearchEnabled}
                    disabled={disabled}
                >
                    {SAFE_SEARCH_PROVIDER_KEYS.map((key) => (
                        <div key={key} className={s.checkboxRow}>
                            <Checkbox
                                id={`safe-search-${key}`}
                                checked={form.safe_search[key]}
                                onChange={handleSafeSearchProvider(key)}
                                disabled={disabled || !form.safe_search.enabled}
                            >
                                {SAFE_SEARCH_PROVIDERS[key]}
                            </Checkbox>
                        </div>
                    ))}
                </SwitchGroup>

                <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                    {intl.getMessage('clients_logs_and_statistics')}
                </h2>

                <SwitchGroup
                    id="ignore-querylog"
                    title={intl.getMessage('clients_dont_log')}
                    checked={form.ignore_querylog}
                    onChange={handleToggle('ignore_querylog')}
                />

                <SwitchGroup
                    id="ignore-statistics"
                    title={intl.getMessage('clients_dont_collect_stats')}
                    checked={form.ignore_statistics}
                    onChange={handleToggle('ignore_statistics')}
                />
            </div>
        </div>
    );
};
