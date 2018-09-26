import { connect } from 'react-redux';
import { initSettings, toggleSetting, handleUpstreamChange, setUpstream, testUpstream, addErrorToast } from '../actions';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const { settings, dashboard } = state;
    const props = { settings, dashboard };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
    handleUpstreamChange,
    setUpstream,
    testUpstream,
    addErrorToast,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
