import { connect } from 'react-redux';
import {
    getRewritesList,
    addRewrite,
    deleteRewrite,
    toggleRewritesModal,
} from '../actions/rewrites';
import Rewrites from '../components/Filters/Rewrites';

const mapStateToProps = (state) => {
    const { rewrites } = state;
    const props = { rewrites };
    return props;
};

const mapDispatchToProps = {
    getRewritesList,
    addRewrite,
    deleteRewrite,
    toggleRewritesModal,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Rewrites);
