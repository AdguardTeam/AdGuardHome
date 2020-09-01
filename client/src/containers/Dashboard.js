import { connect } from 'react-redux';
import { toggleProtection, getClients } from '../actions';
import { getStats, getStatsConfig, setStatsConfig } from '../actions/stats';
import { getAccessList } from '../actions/access';
import Dashboard from '../components/Dashboard';

const mapStateToProps = (state) => {
    const { dashboard, stats, access } = state;
    const props = { dashboard, stats, access };
    return props;
};

const mapDispatchToProps = {
    toggleProtection,
    getClients,
    getStats,
    getStatsConfig,
    setStatsConfig,
    getAccessList,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dashboard);
