import { useEffect } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';

import { ONE_SECOND_IN_MS } from '../../helpers/constants';
import { setProtectionTimerTime, toggleProtectionSuccess } from '../../actions';

let interval = null;

const ProtectionTimer = ({
    protectionDisabledDuration,
    toggleProtectionSuccess,
    setProtectionTimerTime,
}) => {
    useEffect(() => {
        if (protectionDisabledDuration !== null && protectionDisabledDuration < ONE_SECOND_IN_MS) {
            toggleProtectionSuccess({ disabledDuration: null });
        }

        if (protectionDisabledDuration) {
            interval = setInterval(() => {
                setProtectionTimerTime(protectionDisabledDuration - ONE_SECOND_IN_MS);
            }, ONE_SECOND_IN_MS);
        }

        return () => {
            clearInterval(interval);
        };
    }, [protectionDisabledDuration]);

    return null;
};

ProtectionTimer.propTypes = {
    protectionDisabledDuration: PropTypes.number,
    toggleProtectionSuccess: PropTypes.func.isRequired,
    setProtectionTimerTime: PropTypes.func.isRequired,
};

const mapStateToProps = (state) => {
    const { dashboard } = state;
    const { protectionEnabled, protectionDisabledDuration } = dashboard;
    return { protectionEnabled, protectionDisabledDuration };
};

const mapDispatchToProps = {
    toggleProtectionSuccess,
    setProtectionTimerTime,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(ProtectionTimer);
