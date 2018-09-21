import { connect } from 'react-redux';
import { initSettings, toggleSetting, handleUpstreamChange, setUpstream, testUpstream } from '../actions';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const { settings } = state;
    const props = { settings };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
    handleUpstreamChange,
    setUpstream,
    testUpstream,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
