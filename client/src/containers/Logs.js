import { connect } from 'react-redux';
import { getLogs, toggleLogStatus, downloadQueryLog } from '../actions';
import Logs from '../components/Logs';

const mapStateToProps = (state) => {
    const { queryLogs, dashboard } = state;
    const props = { queryLogs, dashboard };
    return props;
};

const mapDispatchToProps = {
    getLogs,
    toggleLogStatus,
    downloadQueryLog,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Logs);
