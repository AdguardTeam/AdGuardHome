import { connect } from 'react-redux';
import {
    setRules,
    getFilteringStatus,
    handleRulesChange,
    checkHost,
} from '../actions/filtering';
import CustomRules from '../components/Filters/CustomRules';

const mapStateToProps = (state) => {
    const { filtering } = state;
    const props = { filtering };
    return props;
};

const mapDispatchToProps = {
    setRules,
    getFilteringStatus,
    handleRulesChange,
    checkHost,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(CustomRules);
