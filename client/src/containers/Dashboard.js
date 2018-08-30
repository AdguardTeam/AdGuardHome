import { connect } from 'react-redux';
import * as actionCreators from '../actions';
import Dashboard from '../components/Dashboard';

const mapStateToProps = (state) => {
    const { dashboard } = state;
    const props = { dashboard };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(Dashboard);
