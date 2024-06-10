import { connect } from 'react-redux';

import * as actionCreators from '../actions';

import SetupGuide from '../components/SetupGuide';

const mapStateToProps = (state: any) => {
    const { dashboard } = state;
    const props = { dashboard };
    return props;
};

export default connect(mapStateToProps, actionCreators)(SetupGuide);
