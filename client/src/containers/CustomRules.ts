import { connect } from 'react-redux';
import { setRules, getFilteringStatus, handleRulesChange, checkHost } from '../actions/filtering';

import CustomRules from '../components/Filters/CustomRules';
import { RootState } from '../initialState';

const mapStateToProps = (state: RootState) => {
    const { filtering } = state;
    const props = { filtering };
    return props;
};

type DispatchProps = {
    setRules: (...args: unknown[]) => unknown;
    getFilteringStatus: (...args: unknown[]) => unknown;
    handleRulesChange: (...args: unknown[]) => unknown;
    checkHost: (dispatch: any) => void;
}

const mapDispatchToProps: DispatchProps = {
    setRules,
    getFilteringStatus,
    handleRulesChange,
    checkHost,
};

export default connect(mapStateToProps, mapDispatchToProps)(CustomRules);
