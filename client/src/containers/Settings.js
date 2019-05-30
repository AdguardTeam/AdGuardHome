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
    addStaticLease,
    removeStaticLease,
    toggleLeaseModal,
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
import {
    getAccessList,
    setAccessList,
} from '../actions/access';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const {
        settings,
        dashboard,
        dhcp,
        encryption,
        clients,
        access,
    } = state;
    const props = {
        settings,
        dashboard,
        dhcp,
        encryption,
        clients,
        access,
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
    addStaticLease,
    removeStaticLease,
    toggleLeaseModal,
    getAccessList,
    setAccessList,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
