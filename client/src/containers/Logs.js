import { connect } from 'react-redux';
import { getFilteringStatus, setRules } from '../actions/filtering';
import {
    getLogs, setLogsPagination, setLogsPage, toggleDetailedLogs,
} from '../actions/queryLogs';
import Logs from '../components/Logs';
import { addSuccessToast } from '../actions/toasts';

const mapStateToProps = (state) => {
    const {
        queryLogs, dashboard, filtering, dnsConfig,
    } = state;

    const props = {
        queryLogs,
        dashboard,
        filtering,
        dnsConfig,
    };
    return props;
};

const mapDispatchToProps = {
    getLogs,
    getFilteringStatus,
    setRules,
    addSuccessToast,
    setLogsPagination,
    setLogsPage,
    toggleDetailedLogs,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Logs);
