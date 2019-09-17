import { connect } from 'react-redux';
import { handleUpstreamChange, setUpstream, testUpstream, getDnsSettings } from '../actions';
import { getAccessList, setAccessList } from '../actions/access';
import {
    getRewritesList,
    addRewrite,
    deleteRewrite,
    toggleRewritesModal,
} from '../actions/rewrites';
import Dns from '../components/Settings/Dns';

const mapStateToProps = (state) => {
    const {
        dashboard, settings, access, rewrites,
    } = state;
    const props = {
        dashboard,
        settings,
        access,
        rewrites,
    };
    return props;
};

const mapDispatchToProps = {
    handleUpstreamChange,
    setUpstream,
    testUpstream,
    getAccessList,
    setAccessList,
    getRewritesList,
    addRewrite,
    deleteRewrite,
    toggleRewritesModal,
    getDnsSettings,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Dns);
