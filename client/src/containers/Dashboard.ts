import { connect } from 'react-redux';

import { toggleProtection, getClients } from '../actions';
import { getStats, getStatsConfig, setStatsConfig } from '../actions/stats';
import { getAccessList } from '../actions/access';

import Dashboard from '../components/Dashboard';
import { RootState } from '../initialState';

const mapStateToProps = (state: RootState) => {
    const { dashboard, stats, access } = state;
    const props = { dashboard, stats, access };
    return props;
};

type DispatchProps = {
    toggleProtection: (...args: unknown[]) => unknown;
    getClients: (...args: unknown[]) => unknown;
    getStats: (...args: unknown[]) => unknown;
    getStatsConfig: (...args: unknown[]) => unknown;
    getAccessList: () => (dispatch: any) => void;
}

const mapDispatchToProps: DispatchProps = {
    toggleProtection,
    getClients,
    getStats,
    getStatsConfig,
    getAccessList,
};

export default connect(mapStateToProps, mapDispatchToProps)(Dashboard);
