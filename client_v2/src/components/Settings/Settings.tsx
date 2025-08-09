import React, { useEffect } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Checkbox } from 'panel/common/controls/Checkbox';
import theme from 'panel/lib/theme';
import { PageLoader } from 'panel/common/ui/Loader';
import { SettingsData, StatsData, QueryLogsData, FilteringData } from 'panel/initialState';

import { StatsConfig } from './StatsConfig/StatsConfig';
import { LogsConfig } from './LogsConfig';
import { FiltersConfig } from './FiltersConfig';
import { getSafeSearchProviderTitle } from './helpers';
import type { StatsConfigPayload } from './StatsConfig/StatsConfig';
import type { LogsConfigPayload } from './LogsConfig/LogsConfig';
import type { FormValues as FiltersFormValues } from './FiltersConfig';
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

type ToggleSettingArgKey = keyof typeof SETTINGS | 'safesearch';
type ToggleSettingArgValue = boolean | Record<string, boolean>;

type Props = {
    settings: SettingsData;
    stats: StatsData;
    queryLogs: QueryLogsData;
    filtering: FilteringData;
    initSettings: () => void;
    toggleSetting: (key: ToggleSettingArgKey, value: ToggleSettingArgValue) => void;
    getStatsConfig: () => void;
    setStatsConfig: (config: StatsConfigPayload) => void;
    resetStats: () => void;
    setFiltersConfig: (values: FiltersFormValues) => void;
    getFilteringStatus: () => void;
    getLogsConfig: () => void;
    setLogsConfig: (values: LogsConfigPayload) => void;
    clearLogs: () => void;
};

export const Settings = ({
    initSettings,
    getStatsConfig,
    getLogsConfig,
    getFilteringStatus,
    settings,
    toggleSetting,
    setStatsConfig,
    resetStats,
    stats,
    queryLogs,
    setLogsConfig,
    clearLogs,
    filtering,
    setFiltersConfig,
}: Props) => {
    useEffect(() => {
        initSettings();
        getStatsConfig();
        getFilteringStatus();
        getLogsConfig();
    }, []);

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
                              onChange={(e) => toggleSetting(key, !e.target.checked)}
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

        return (
            <SwitchGroup
                id="safesearch"
                title={intl.getMessage('settings_safe_search')}
                description={intl.getMessage('settings_safe_search_desc')}
                checked={enabled}
                onChange={(e) => toggleSetting('safesearch', { ...safesearch, enabled: e.target.checked })}>
                <div>
                    {Object.keys(searches).map((searchKey) => (
                        <div key={searchKey} className={theme.form.checkbox}>
                            <Checkbox
                                id={searchKey}
                                checked={searches[searchKey]}
                                disabled={!enabled}
                                onChange={(e) => {
                                    toggleSetting('safesearch', { ...safesearch, [searchKey]: e.target.checked });
                                }}>
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
                        setFiltersConfig={setFiltersConfig}
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
                        setLogsConfig={setLogsConfig}
                        clearLogs={clearLogs}
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
                        setStatsConfig={setStatsConfig}
                        resetStats={resetStats}
                    />
                </>
            )}
        </div>
    );
};
