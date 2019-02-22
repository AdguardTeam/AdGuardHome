import { connect } from 'react-redux';
import * as actionCreators from '../actions';
import App from '../components/App';

const mapStateToProps = (state) => {
    const { dashboard, encryption } = state;
    const props = { dashboard, encryption };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(App);
