import { connect } from 'react-redux';
import { addSuccessToast, getClients } from '../actions';
import { getFilteringStatus, setRules } from '../actions/filtering';
import { getLogs, getLogsConfig, setLogsPagination } from '../actions/queryLogs';
import Logs from '../components/Logs';

const mapStateToProps = (state) => {
    const { queryLogs, dashboard, filtering } = state;
    const props = { queryLogs, dashboard, filtering };
    return props;
};

const mapDispatchToProps = {
    getLogs,
    getFilteringStatus,
    setRules,
    addSuccessToast,
    getClients,
    getLogsConfig,
    setLogsPagination,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Logs);
