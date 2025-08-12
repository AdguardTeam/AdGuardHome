import React, { useEffect } from 'react';
import cn from 'clsx';
import { shallowEqual, useDispatch, useSelector, batch } from 'react-redux';

import intl from 'panel/common/intl';
import { Checkbox } from 'panel/common/controls/Checkbox';
import theme from 'panel/lib/theme';
import { PageLoader } from 'panel/common/ui/Loader';
import { RootState, SettingsData } from 'panel/initialState';
import { initSettings, toggleSetting } from 'panel/actions';
import { getStatsConfig } from 'panel/actions/stats';
import { getLogsConfig } from 'panel/actions/queryLogs';
import { getFilteringStatus } from 'panel/actions/filtering';

import { StatsConfig } from './StatsConfig/StatsConfig';
import { LogsConfig } from './LogsConfig';
import { FiltersConfig } from './FiltersConfig';
import { getSafeSearchProviderTitle } from './helpers';
import { SwitchGroup } from './SettingsGroup';

const SETTINGS = {
    safebrowsing: {
        enabled: false,
        title: intl.getMessage('settings_browsing_security'),
        subtitle: intl.getMessage('settings_browsing_security_desc'),
    },
    parental: {
        enabled: false,
        title: intl.getMessage('settings_parental_control'),
        subtitle: intl.getMessage('settings_parental_control_desc'),
    },
};

export const Settings = () => {
    const dispatch = useDispatch();

    const settings = useSelector((state: RootState) => state.settings, shallowEqual);
    const stats = useSelector((state: RootState) => state.stats, shallowEqual);
    const queryLogs = useSelector((state: RootState) => state.queryLogs, shallowEqual);
    const filtering = useSelector((state: RootState) => state.filtering, shallowEqual);

    useEffect(() => {
        batch(() => {
            dispatch(initSettings());
            dispatch(getStatsConfig());
            dispatch(getFilteringStatus());
            dispatch(getLogsConfig());
        });
    }, []);

    const handleSettingToggle =
        (key: keyof typeof SETTINGS) => (e: React.ChangeEvent<HTMLInputElement>) =>
            dispatch(toggleSetting(key, !e.target.checked));

    const renderSettings = (settingsList?: SettingsData['settingsList']) =>
        settingsList
            ? (Object.keys(SETTINGS) as Array<keyof typeof SETTINGS>).map((key) => {
                  const { title, subtitle } = SETTINGS[key];
                  const enabled = Boolean(settingsList[key]?.enabled);
                  return (
                      <div key={key}>
                          <SwitchGroup
                              title={title}
                              description={subtitle}
                              id={String(key)}
                              checked={enabled}
                              onChange={handleSettingToggle(key)}
                          />
                      </div>
                  );
              })
            : null;

    const renderSafeSearch = () => {
        const safesearch = settings.settingsList?.safesearch;

        if (!safesearch) {
            return null;
        }

        const { enabled, ...searches } = safesearch;

        type SafeSearchConfigShape = Record<string, boolean> & { enabled: boolean };

        const onSafeSearchEnabledChange =
            (e: React.ChangeEvent<HTMLInputElement>) => {
                const payload = { ...safesearch, enabled: e.target.checked } as SafeSearchConfigShape;
                dispatch(toggleSetting('safesearch', payload));
            };

        const onProviderChange =
            (searchKey: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
                const payload = {
                    ...safesearch,
                    [searchKey]: e.target.checked,
                } as SafeSearchConfigShape;
                dispatch(toggleSetting('safesearch', payload));
            };

        return (
            <SwitchGroup
                id="safesearch"
                title={intl.getMessage('settings_safe_search')}
                description={intl.getMessage('settings_safe_search_desc')}
                checked={enabled}
                onChange={onSafeSearchEnabledChange}>
                <div>
                    {Object.keys(searches).map((searchKey) => (
                        <div key={searchKey} className={theme.form.checkbox}>
                            <Checkbox
                                id={searchKey}
                                checked={searches[searchKey]}
                                disabled={!enabled}
                                onChange={onProviderChange(searchKey)}>
                                {getSafeSearchProviderTitle(searchKey)}
                            </Checkbox>
                        </div>
                    ))}
                </div>
            </SwitchGroup>
        );
    };

    const isLoading = settings.processing || stats.processingGetConfig || queryLogs.processingGetConfig;

    return (
        <div className={theme.layout.container}>
            <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                {intl.getMessage('general_settings')}
            </h1>

            {isLoading && <PageLoader />}

            {!isLoading && (
                <>
                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('settings_filter_requests')}
                    </h2>

                    <FiltersConfig
                        initialValues={{
                            interval: filtering.interval,
                            enabled: filtering.enabled,
                        }}
                        processing={filtering.processingSetConfig}
                    />

                    {renderSettings(settings.settingsList)}

                    {renderSafeSearch()}

                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('query_log')}
                    </h2>

                    <LogsConfig
                        enabled={queryLogs.enabled}
                        ignored={queryLogs.ignored}
                        interval={queryLogs.interval}
                        customInterval={queryLogs.customInterval}
                        anonymize_client_ip={queryLogs.anonymize_client_ip}
                        processing={queryLogs.processingSetConfig}
                        processingClear={queryLogs.processingClear}
                    />

                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('settings_statistics')}
                    </h2>

                    <StatsConfig
                        interval={stats.interval}
                        customInterval={stats.customInterval}
                        ignored={stats.ignored}
                        enabled={stats.enabled}
                        processing={stats.processingSetConfig}
                        processingReset={stats.processingReset}
                    />
                </>
            )}
        </div>
    );
};
