import { connect } from 'react-redux';
import { initSettings, toggleSetting } from '../actions';
import { getBlockedServices, setBlockedServices } from '../actions/services';
import { getStatsConfig, setStatsConfig, resetStats } from '../actions/stats';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const { settings, services, stats } = state;
    const props = {
        settings,
        services,
        stats,
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
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
