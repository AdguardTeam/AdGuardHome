import { connect } from 'react-redux';
import { initSettings, toggleSetting } from '../actions';
import { getBlockedServices, setBlockedServices } from '../actions/services';
import { getStatsConfig, setStatsConfig, resetStats } from '../actions/stats';
import { clearLogs, getLogsConfig, setLogsConfig } from '../actions/queryLogs';
import { getFilteringStatus, setFiltersConfig } from '../actions/filtering';
import { getDnsConfig, setDnsConfig } from '../actions/dnsConfig';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const {
        settings, services, stats, queryLogs, filtering, dnsConfig,
    } = state;
    const props = {
        settings,
        services,
        stats,
        queryLogs,
        filtering,
        dnsConfig,
    };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
    getBlockedServices,
    setBlockedServices,
    getStatsConfig,
    setStatsConfig,
    resetStats,
    clearLogs,
    getLogsConfig,
    setLogsConfig,
    getFilteringStatus,
    setFiltersConfig,
    getDnsConfig,
    setDnsConfig,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
