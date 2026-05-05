import React, { Component, Fragment } from 'react';
import { withTranslation } from 'react-i18next';

import i18next from 'i18next';
import StatsConfig from './StatsConfig';

import LogsConfig from './LogsConfig';

import { FiltersConfig } from './FiltersConfig';

import { Checkbox } from '../ui/Controls/Checkbox';

import Loading from '../ui/Loading';

import PageTitle from '../ui/PageTitle';

import Card from '../ui/Card';

import { captitalizeWords } from '../../helpers/helpers';
import './Settings.css';
import { SettingsData } from '../../initialState';

interface SettingsProps {
    initSettings: (...args: unknown[]) => unknown;
    settings: SettingsData;
    toggleSetting: (...args: unknown[]) => unknown;
    getStatsConfig: (...args: unknown[]) => unknown;
    setStatsConfig: (...args: unknown[]) => unknown;
    resetStats: (...args: unknown[]) => unknown;
    setFiltersConfig: (...args: unknown[]) => unknown;
    getFilteringStatus: (...args: unknown[]) => unknown;
    t: (...args: unknown[]) => string;
    getLogsConfig?: (...args: unknown[]) => unknown;
    setLogsConfig?: (...args: unknown[]) => unknown;
    clearLogs?: (...args: unknown[]) => unknown;
    stats?: {
        processingGetConfig?: boolean;
        interval?: number;
        customInterval?: number;
        enabled?: boolean;
        ignored?: unknown[];
        ignored_enabled?: boolean;
        processingSetConfig?: boolean;
        processingReset?: boolean;
    };
    queryLogs?: {
        enabled?: boolean;
        interval?: number;
        customInterval?: number;
        anonymize_client_ip?: boolean;
        processingSetConfig?: boolean;
        processingClear?: boolean;
        processingGetConfig?: boolean;
        ignored?: unknown[];
        ignored_enabled?: boolean;
    };
    filtering?: {
        interval?: number;
        enabled?: boolean;
        processingSetConfig?: boolean;
    };
}

class Settings extends Component<SettingsProps> {
    componentDidMount() {
        this.props.initSettings();

        this.props.getStatsConfig();

        this.props.getLogsConfig();

        this.props.getFilteringStatus();
    }

    renderSafeSearch = () => {
        const safesearch = this.props.settings.settingsList?.safesearch || {};
        const { enabled } = safesearch;
        const searches = { ...(safesearch || {}) };
        delete searches.enabled;

        return (
            <>
                <div className="form__group form__group--checkbox">
                    <Checkbox
                        data-testid="safesearch"
                        value={enabled}
                        title={i18next.t('enforce_safe_search')}
                        subtitle={i18next.t('enforce_save_search_hint')}
                        onChange={(checked) =>
                            this.props.toggleSetting('safesearch', { ...safesearch, enabled: checked })
                        }
                    />
                </div>

                <div className="form__group--inner">
                    {Object.keys(searches).map((searchKey) => (
                        <div key={searchKey} className="form__group form__group--checkbox">
                            <Checkbox
                                value={searches[searchKey]}
                                title={captitalizeWords(searchKey)}
                                disabled={!safesearch.enabled}
                                onChange={(checked) =>
                                    this.props.toggleSetting('safesearch', { ...safesearch, [searchKey]: checked })
                                }
                            />
                        </div>
                    ))}
                </div>
            </>
        );
    };

    render() {
        const {
            settings,
            setStatsConfig,
            resetStats,
            stats,
            queryLogs,
            setLogsConfig,
            clearLogs,
            filtering,
            setFiltersConfig,
            t,
        } = this.props;
        const safebrowsingEnabled = settings.settingsList?.safebrowsing?.enabled ?? false;
        const parentalEnabled = settings.settingsList?.parental?.enabled ?? false;

        const isDataReady = !settings.processing && !stats.processingGetConfig && !queryLogs.processingGetConfig;

        return (
            <Fragment>
                <PageTitle title={t('general_settings')} />

                {!isDataReady && <Loading />}

                {isDataReady && (
                    <div className="content">
                        <div className="row">
                            <div className="col-md-12">
                                <Card bodyType="card-body box-body--settings">
                                    <div className="form">
                                        <FiltersConfig
                                            initialValues={{
                                                interval: filtering.interval,
                                                enabled: filtering.enabled,
                                            }}
                                            processing={filtering.processingSetConfig}
                                            setFiltersConfig={setFiltersConfig}
                                        />
                                        <div className="form__group form__group--checkbox">
                                            <Checkbox
                                                data-testid="safebrowsing"
                                                value={safebrowsingEnabled}
                                                title={t('use_adguard_browsing_sec')}
                                                subtitle={t('use_adguard_browsing_sec_hint')}
                                                onChange={(checked) => this.props.toggleSetting('safebrowsing', !checked)}
                                            />
                                        </div>
                                        <div className="form__group form__group--checkbox">
                                            <Checkbox
                                                data-testid="parental"
                                                value={parentalEnabled}
                                                title={t('use_adguard_parental')}
                                                subtitle={t('use_adguard_parental_hint')}
                                                onChange={(checked) => this.props.toggleSetting('parental', !checked)}
                                            />
                                        </div>
                                        {this.renderSafeSearch()}
                                    </div>
                                </Card>
                            </div>

                            <div className="col-md-12">
                                <LogsConfig
                                    enabled={queryLogs.enabled}
                                    ignored={queryLogs.ignored}
                                    ignoredEnabled={queryLogs.ignored_enabled}
                                    interval={queryLogs.interval}
                                    customInterval={queryLogs.customInterval}
                                    anonymize_client_ip={queryLogs.anonymize_client_ip}
                                    processing={queryLogs.processingSetConfig}
                                    processingClear={queryLogs.processingClear}
                                    setLogsConfig={setLogsConfig}
                                    clearLogs={clearLogs}
                                />
                            </div>

                            <div className="col-md-12">
                                <StatsConfig
                                    interval={stats.interval}
                                    customInterval={stats.customInterval}
                                    ignored={stats.ignored}
                                    ignoredEnabled={stats.ignored_enabled}
                                    enabled={stats.enabled}
                                    processing={stats.processingSetConfig}
                                    processingReset={stats.processingReset}
                                    setStatsConfig={setStatsConfig}
                                    resetStats={resetStats}
                                />
                            </div>
                        </div>
                    </div>
                )}
            </Fragment>
        );
    }
}

export default withTranslation()(Settings);
