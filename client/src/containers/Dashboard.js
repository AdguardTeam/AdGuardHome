import { connect } from 'react-redux';
import { toggleProtection, getClients } from '../actions';
import { getStats, getStatsConfig, setStatsConfig } from '../actions/stats';
import Dashboard from '../components/Dashboard';

const mapStateToProps = (state) => {
    const { dashboard, stats } = state;
    const props = { dashboard, stats };
    return props;
};

const mapDispatchToProps = {
    toggleProtection,
    getClients,
    getStats,
    getStatsConfig,
    setStatsConfig,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dashboard);
