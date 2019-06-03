import { connect } from 'react-redux';
import { handleUpstreamChange, setUpstream, testUpstream, addErrorToast } from '../actions';
import { getAccessList, setAccessList } from '../actions/access';
import Dns from '../components/Settings/Dns';

const mapStateToProps = (state) => {
    const { dashboard, settings, access } = state;
    const props = {
        dashboard,
        settings,
        access,
    };
    return props;
};

const mapDispatchToProps = {
    handleUpstreamChange,
    setUpstream,
    testUpstream,
    addErrorToast,
    getAccessList,
    setAccessList,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dns);
