import { connect } from 'react-redux';
import {
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
