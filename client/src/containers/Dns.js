import { connect } from 'react-redux';
import { testUpstream } from '../actions';
import { getAccessList, setAccessList } from '../actions/access';
import {
    getRewritesList,
    addRewrite,
    deleteRewrite,
    toggleRewritesModal,
} from '../actions/rewrites';
import { getDnsConfig, setDnsConfig } from '../actions/dnsConfig';
import Dns from '../components/Settings/Dns';

const mapStateToProps = (state) => {
    const {
        dashboard, settings, access, rewrites, dnsConfig,
    } = state;
    const props = {
        dashboard,
        settings,
        access,
        rewrites,
        dnsConfig,
    };
    return props;
};

const mapDispatchToProps = {
    testUpstream,
    getAccessList,
    setAccessList,
    getRewritesList,
    addRewrite,
    deleteRewrite,
    toggleRewritesModal,
    getDnsConfig,
    setDnsConfig,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dns);
