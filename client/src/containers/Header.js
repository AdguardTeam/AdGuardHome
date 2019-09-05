import { connect } from 'react-redux';
import { getVersion } from '../actions';
import Header from '../components/Header';

const mapStateToProps = (state) => {
    const { dashboard } = state;
    const props = { dashboard };
    return props;
};

const mapDispatchToProps = {
    getVersion,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Header);
