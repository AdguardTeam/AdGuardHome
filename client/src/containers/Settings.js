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
import {
    getTlsStatus,
    setTlsConfig,
    validateTlsConfig,
} from '../actions/encryption';
import {
    addClient,
    updateClient,
    deleteClient,
    toggleClientModal,
} from '../actions/clients';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const {
        settings,
        dashboard,
        dhcp,
        encryption,
        clients,
    } = state;
    const props = {
        settings,
        dashboard,
        dhcp,
        encryption,
        clients,
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
    validateTlsConfig,
    addClient,
    updateClient,
    deleteClient,
    toggleClientModal,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
