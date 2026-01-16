import React, { Component } from 'react';
import { connect } from 'react-redux';

import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import * as actionCreators from '../../actions/install';
import styles from './styles.module.pcss';

interface ControlsProps {
    install: {
        step: number;
        processingSubmit: boolean;
        dns: {
            status: string;
        };
        web: {
            status: string;
        };
    };
    nextStep?: (...args: unknown[]) => unknown;
    prevStep?: (...args: unknown[]) => unknown;
    openDashboard?: (...args: unknown[]) => unknown;
    submitting?: boolean;
    invalid?: boolean;
    pristine?: boolean;
    ip?: string;
    port?: number;
    isDirty?: boolean;
    isValid?: boolean;
}

class Controls extends Component<ControlsProps> {
    renderPrevButton(step: any) {
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
                        onClick={this.props.prevStep}>
                        {intl.getMessage('back')}
                    </Button>
                );
            case 6:
                return false;
            default:
                return false;
        }
    }

    renderNextButton(step: any) {
        const { nextStep, ip, port, isValid, invalid } = this.props;

        const isNextDisabled = invalid === true || isValid === false;

        switch (step) {
            case 1:
                return (
                    <Button
                        id="install_get_started"
                        type="button"
                        onClick={nextStep}
                        size="small"
                        variant="primary"
                        className={styles.button}>
                        {intl.getMessage("setup_guide_greeting_button")}
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
                        onClick={nextStep}
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
                        onClick={() => this.props.openDashboard && this.props.openDashboard(ip!, port!)}>
                        {intl.getMessage("open_dashboard")}
                    </Button>
                );
            default:
                return false;
        }
    }

    render() {
        const { install } = this.props;

        return (
            <div className={styles.nav}>
                {this.renderNextButton(install.step)}
                {this.renderPrevButton(install.step)}
            </div>
        );
    }
}

const mapStateToProps = (state: any) => {
    const { install } = state;
    return { install };
};

export default connect(mapStateToProps, actionCreators)(Controls);
