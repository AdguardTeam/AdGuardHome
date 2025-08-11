import React, { Component } from 'react';
import { connect } from 'react-redux';
import { Trans } from 'react-i18next';

import * as actionCreators from '../../actions/install';
import { Button } from 'panel/common/ui/Button';

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
                return (
                    <Button
                        data-testid="install_back"
                        type="button"
                        size="small"
                        variant="secondary"
                        className="setup__button"
                        onClick={this.props.prevStep}>
                        <Trans>back</Trans>
                    </Button>
                );
            default:
                return false;
        }
    }

    renderNextButton(step: any) {
        const { nextStep, invalid, pristine, install, ip, port } = this.props;

        switch (step) {
            case 1:
                return (
                    <Button
                        data-testid="install_get_started"
                        type="button"
                        onClick={nextStep}
                        size="small"
                        variant="primary"
                        className="setup__button">
                        <Trans>get_started</Trans>
                    </Button>
                );
            case 2:
            case 3:
                return (
                    <Button
                        data-testid="install_next"
                        type="submit"
                        size="small"
                        variant="primary"
                        className="setup__button"
                        disabled={invalid || pristine || install.processingSubmit}>
                        <Trans>next</Trans>
                    </Button>
                );
            case 4:
                return (
                    <Button
                        data-testid="install_next"
                        type="button"
                        size="small"
                        variant="primary"
                        className="setup__button"
                        onClick={nextStep}>
                        <Trans>next</Trans>
                    </Button>
                );
            case 5:
                return (
                    <Button
                        data-testid="install_open_dashboard"
                        type="button"
                        size="small"
                        variant="primary"
                        className="setup__button"
                        onClick={() => this.props.openDashboard(ip, port)}>
                        <Trans>open_dashboard</Trans>
                    </Button>
                );
            default:
                return false;
        }
    }

    render() {
        const { install } = this.props;

        return (
            <div className="setup__nav">
                {this.renderPrevButton(install.step)}
                {this.renderNextButton(install.step)}
            </div>
        );
    }
}

const mapStateToProps = (state: any) => {
    const { install } = state;
    return { install };
};

export default connect(mapStateToProps, actionCreators)(Controls);
