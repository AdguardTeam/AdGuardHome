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
    getTlsStatus,
    setTlsConfig,
} from '../actions';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const {
        settings,
        dashboard,
        dhcp,
        encryption,
    } = state;
    const props = {
        settings,
        dashboard,
        dhcp,
        encryption,
    };
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
    getTlsStatus,
    setTlsConfig,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
