import { connect } from 'react-redux';
import {
    addErrorToast,
    toggleDhcp,
    getDhcpStatus,
    getDhcpInterfaces,
    setDhcpConfig,
    findActiveDhcp,
    toggleLeaseModal,
    addStaticLease,
    removeStaticLease,
} from '../actions';
import Dhcp from '../components/Settings/Dhcp';

const mapStateToProps = (state) => {
    const { dhcp } = state;
    const props = {
        dhcp,
    };
    return props;
};

const mapDispatchToProps = {
    addErrorToast,
    toggleDhcp,
    getDhcpStatus,
    getDhcpInterfaces,
    setDhcpConfig,
    findActiveDhcp,
    toggleLeaseModal,
    addStaticLease,
    removeStaticLease,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dhcp);
