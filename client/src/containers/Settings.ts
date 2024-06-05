import { connect } from 'react-redux';

import { initSettings, toggleSetting } from '../actions';
import { getBlockedServices, updateBlockedServices } from '../actions/services';
import { getStatsConfig, setStatsConfig, resetStats } from '../actions/stats';
import { clearLogs, getLogsConfig, setLogsConfig } from '../actions/queryLogs';
import { getFilteringStatus, setFiltersConfig } from '../actions/filtering';

import Settings from '../components/Settings';

const mapStateToProps = (state: any) => {
    const { settings, services, stats, queryLogs, filtering } = state;
    const props = {
        settings,
        services,
        stats,
        queryLogs,
        filtering,
    };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
    getBlockedServices,
    updateBlockedServices,
    getStatsConfig,
    setStatsConfig,
    resetStats,
    clearLogs,
    getLogsConfig,
    setLogsConfig,
    getFilteringStatus,
    setFiltersConfig,
};

export default connect(mapStateToProps, mapDispatchToProps)(Settings);
