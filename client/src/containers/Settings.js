import { connect } from 'react-redux';
import {
    initSettings,
    toggleSetting,
    handleUpstreamChange,
    setUpstream,
    testUpstream,
    addErrorToast,
    toggleDhcp,
    getDhcpStatus,
    getDhcpInterfaces,
    setDhcpConfig,
    findActiveDhcp,
} from '../actions';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const { settings, dashboard, dhcp } = state;
    const props = { settings, dashboard, dhcp };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
    handleUpstreamChange,
    setUpstream,
    testUpstream,
    addErrorToast,
    toggleDhcp,
    getDhcpStatus,
    getDhcpInterfaces,
    setDhcpConfig,
    findActiveDhcp,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
