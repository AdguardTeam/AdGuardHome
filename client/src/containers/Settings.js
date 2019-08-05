import { connect } from 'react-redux';
import { initSettings, toggleSetting } from '../actions';
import { getBlockedServices, setBlockedServices } from '../actions/services';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const { settings, services } = state;
    const props = {
        settings,
        services,
    };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
    getBlockedServices,
    setBlockedServices,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
