import { connect } from 'react-redux';
import { initSettings, toggleSetting } from '../actions';
import Settings from '../components/Settings';

const mapStateToProps = (state) => {
    const { settings } = state;
    const props = {
        settings,
    };
    return props;
};

const mapDispatchToProps = {
    initSettings,
    toggleSetting,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Settings);
