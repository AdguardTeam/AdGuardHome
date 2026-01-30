import React from 'react';
import { useDispatch, useSelector } from 'react-redux';

import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import { InstallState } from 'panel/initialState';
import * as actionCreators from '../../actions/install';
import styles from './styles.module.pcss';

interface ControlsProps {
    invalid?: boolean;
    ip?: string;
    port?: number;
    isDirty?: boolean;
    isValid?: boolean;
    openDashboard?: (ip: string, port: number) => void;
}

const Controls = ({ invalid, isValid, ip, port, openDashboard }: ControlsProps) => {
    const dispatch = useDispatch();
    const install = useSelector((state: InstallState) => state.install);

    const handleNextStep = () => {
        dispatch(actionCreators.nextStep());
    };

    const handlePrevStep = () => {
        dispatch(actionCreators.prevStep());
    };

    const renderPrevButton = (step: number) => {
        switch (step) {
            case 2:
            case 3:
            case 4:
            case 5:
                return (
                    <Button
                        id="install_back"
                        type="button"
                        size="small"
                        variant="secondary"
                        className={styles.button}
                        onClick={handlePrevStep}>
                        {intl.getMessage('back')}
                    </Button>
                );
            case 6:
                return false;
            default:
                return false;
        }
    };

    const renderNextButton = (step: number) => {
        const isNextDisabled = invalid === true || isValid === false;

        switch (step) {
            case 1:
                return (
                    <Button
                        id="install_get_started"
                        type="button"
                        onClick={handleNextStep}
                        size="small"
                        variant="primary"
                        className={styles.button}>
                        {intl.getMessage('setup_guide_greeting_button')}
                    </Button>
                );
            case 2:
            case 3:
            case 4:
                return (
                    <Button
                        id="install_next"
                        type="submit"
                        size="small"
                        variant="primary"
                        className={styles.button}
                        disabled={isNextDisabled}>
                        {intl.getMessage('next')}
                    </Button>
                );
            case 5:
                return (
                    <Button
                        id="install_next"
                        type="button"
                        onClick={handleNextStep}
                        size="small"
                        variant="primary"
                        className={styles.button}
                        disabled={isNextDisabled}>
                        {intl.getMessage('next')}
                    </Button>
                );
            case 6:
                return (
                    <Button
                        id="open_dashboard"
                        type="button"
                        size="small"
                        variant="primary"
                        className={styles.button}
                        onClick={() => {
                            if (openDashboard && ip && port) {
                                openDashboard(ip, port);
                            }
                        }}>
                        {intl.getMessage('open_dashboard')}
                    </Button>
                );
            default:
                return false;
        }
    };

    return (
        <div className={styles.nav}>
            {renderNextButton(install.step)}
            {renderPrevButton(install.step)}
        </div>
    );
};

export default Controls;
