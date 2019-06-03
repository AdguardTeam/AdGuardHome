import { connect } from 'react-redux';
import { handleUpstreamChange, setUpstream, testUpstream, addErrorToast } from '../actions';
import Dns from '../components/Settings/Dns';

const mapStateToProps = (state) => {
    const { dashboard, settings } = state;
    const props = {
        dashboard,
        settings,
    };
    return props;
};

const mapDispatchToProps = {
    handleUpstreamChange,
    setUpstream,
    testUpstream,
    addErrorToast,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dns);
