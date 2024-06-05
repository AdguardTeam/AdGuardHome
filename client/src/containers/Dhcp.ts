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
    resetDhcp,
} from '../actions';

import Dhcp from '../components/Settings/Dhcp';

const mapStateToProps = (state: any) => {
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
    resetDhcp,
};

export default connect(mapStateToProps, mapDispatchToProps)(Dhcp);
