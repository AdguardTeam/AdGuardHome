import { connect } from 'react-redux';
import * as actionCreators from '../actions';
import Filters from '../components/Filters';

const mapStateToProps = (state) => {
    const { filtering } = state;
    const props = { filtering };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(Filters);
