import { createMemo, createEffect, onMount, For, Show, untrack } from 'solid-js';
import cn from 'clsx';

import { SCROLL_QUERY_KEY } from 'panel/components/Routes/Paths';

import { useSearchParams } from '@solidjs/router';

import intl from 'panel/common/intl';
import { Checkbox } from 'panel/common/controls/Checkbox';
import theme from 'panel/lib/theme';
import { PageLoader } from 'panel/common/ui/Loader';
import { initSettings, toggleSetting, settingsState } from 'panel/stores/settings';
import { getStatsConfig, statsState } from 'panel/stores/stats';
import { getLogsConfig, queryLogsState } from 'panel/stores/queryLogs';
import { getFilteringStatus, filteringState } from 'panel/stores/filtering';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';

import { StatsConfig } from './StatsConfig/StatsConfig';
import { LogsConfig } from './LogsConfig';
import { FiltersConfig } from './FiltersConfig';
import { getSafeSearchProviderTitle } from './helpers';

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
    onMount(() => {
        initSettings();
        getStatsConfig();
        getFilteringStatus();
        getLogsConfig();
    });

    const handleSettingToggle = (key: keyof typeof SETTINGS) => (e: Event) =>
        toggleSetting(key, !(e.target as HTMLInputElement).checked);

    const settingsKeys = Object.keys(SETTINGS) as Array<keyof typeof SETTINGS>;

    const safesearch = createMemo(() => settingsState.settingsList?.safesearch);
    const safesearchEnabled = createMemo(() => safesearch()?.enabled ?? false);
    const safesearchProviders = createMemo(() => {
        const ss = safesearch();
        if (!ss) return [];
        const { enabled, ...providers } = ss;
        void enabled;
        return Object.entries(providers).map(([key, value]) => ({ key, value: value as boolean }));
    });

    const onSafeSearchEnabledChange = (e: Event) => {
        const ss = untrack(safesearch);
        if (!ss) return;
        const payload = { ...ss, enabled: (e.target as HTMLInputElement).checked };
        toggleSetting('safesearch', payload);
    };

    const onProviderChange = (searchKey: string) => (e: Event) => {
        const ss = untrack(safesearch);
        if (!ss) return;
        const payload = { ...ss, [searchKey]: (e.target as HTMLInputElement).checked };
        toggleSetting('safesearch', payload);
    };

    const isLoading = createMemo(() => {
        const hasCachedData = Object.keys(settingsState.settingsList || {}).length > 0;
        return (
            !hasCachedData &&
            (settingsState.processing ||
                statsState.processingGetConfig ||
                queryLogsState.processingGetConfig)
        );
    });

    const [searchParams] = useSearchParams<{ [SCROLL_QUERY_KEY]?: string }>();

    createEffect(() => {
        if (!isLoading()) {
            const section = searchParams[SCROLL_QUERY_KEY];
            if (section) {
                requestAnimationFrame(() => {
                    const el = document.getElementById(section);
                    if (el) {
                        const top = el.getBoundingClientRect().top + window.scrollY - 80;
                        window.scrollTo({ top, behavior: 'smooth' });
                    }
                });
            }
        }
    });

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('settings_general_short')}
                </h1>

                <Show
                    when={isLoading()}
                    fallback={
                        <>
                            <h2
                                class={cn(
                                    theme.layout.subtitle,
                                    theme.title.h5,
                                    theme.title.h4_tablet,
                                )}
                            >
                                {intl.getMessage('settings_filtering_and_security')}
                            </h2>

                            <FiltersConfig
                                initialValues={{
                                    interval: filteringState.interval,
                                    enabled: filteringState.enabled,
                                }}
                                processing={filteringState.processingSetConfig}
                            />

                            <For each={settingsKeys}>
                                {(key) => {
                                    const { title, subtitle } = SETTINGS[key];
                                    const enabled = () =>
                                        Boolean(settingsState.settingsList?.[key]?.enabled);
                                    return (
                                        <div>
                                            <SwitchGroup
                                                title={title}
                                                description={subtitle}
                                                id={String(key)}
                                                checked={enabled()}
                                                onChange={handleSettingToggle(key)}
                                            />
                                        </div>
                                    );
                                }}
                            </For>

                            <Show when={safesearch()}>
                                <SwitchGroup
                                    id="safesearch"
                                    title={intl.getMessage('settings_safe_search')}
                                    description={intl.getMessage('settings_safe_search_desc')}
                                    checked={safesearchEnabled()}
                                    onChange={onSafeSearchEnabledChange}
                                >
                                    <div>
                                        <For each={safesearchProviders()}>
                                            {(provider) => (
                                                <div class={theme.form.checkbox}>
                                                    <Checkbox
                                                        id={provider.key}
                                                        checked={provider.value}
                                                        disabled={!safesearchEnabled()}
                                                        onChange={onProviderChange(provider.key)}
                                                    >
                                                        {getSafeSearchProviderTitle(provider.key)}
                                                    </Checkbox>
                                                </div>
                                            )}
                                        </For>
                                    </div>
                                </SwitchGroup>
                            </Show>

                            <h2
                                class={cn(
                                    theme.layout.subtitle,
                                    theme.title.h5,
                                    theme.title.h4_tablet,
                                )}
                            >
                                {intl.getMessage('query_log')}
                            </h2>

                            <LogsConfig
                                enabled={queryLogsState.enabled}
                                ignored={queryLogsState.ignored}
                                interval={queryLogsState.interval}
                                customInterval={queryLogsState.customInterval}
                                anonymize_client_ip={queryLogsState.anonymize_client_ip}
                                processing={queryLogsState.processingSetConfig}
                                processingClear={queryLogsState.processingClear}
                            />

                            <h2
                                id="stats_config"
                                class={cn(
                                    theme.layout.subtitle,
                                    theme.title.h5,
                                    theme.title.h4_tablet,
                                )}
                            >
                                {intl.getMessage('settings_statistics')}
                            </h2>

                            <StatsConfig
                                interval={statsState.interval}
                                customInterval={statsState.customInterval}
                                ignored={statsState.ignored}
                                enabled={statsState.enabled}
                                processing={statsState.processingSetConfig}
                                processingReset={statsState.processingReset}
                            />
                        </>
                    }
                >
                    <PageLoader />
                </Show>
            </div>
        </div>
    );
};
