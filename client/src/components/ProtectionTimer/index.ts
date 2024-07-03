import { useEffect } from 'react';
import { connect } from 'react-redux';

import { ONE_SECOND_IN_MS } from '../../helpers/constants';

import { setProtectionTimerTime, toggleProtectionSuccess } from '../../actions';

let interval: any = null;

interface ProtectionTimerProps {
    protectionDisabledDuration?: number;
    toggleProtectionSuccess: (...args: unknown[]) => unknown;
    setProtectionTimerTime: (...args: unknown[]) => unknown;
}

const ProtectionTimer = ({
    protectionDisabledDuration,
    toggleProtectionSuccess,
    setProtectionTimerTime,
}: ProtectionTimerProps) => {
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

const mapStateToProps = (state: any) => {
    const { dashboard } = state;
    const { protectionEnabled, protectionDisabledDuration } = dashboard;
    return { protectionEnabled, protectionDisabledDuration };
};

const mapDispatchToProps = {
    toggleProtectionSuccess,
    setProtectionTimerTime,
};

export default connect(mapStateToProps, mapDispatchToProps)(ProtectionTimer);
