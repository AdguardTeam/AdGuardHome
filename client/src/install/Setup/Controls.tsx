import React, { Component } from 'react';
import { connect } from 'react-redux';
import { Trans } from 'react-i18next';

import * as actionCreators from '../../actions/install';

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
}

class Controls extends Component<ControlsProps> {
    renderPrevButton(step: any) {
        switch (step) {
            case 2:
            case 3:
                return (
                    <button
                        type="button"
                        className="btn btn-secondary btn-lg setup__button"
                        onClick={this.props.prevStep}>
                        <Trans>back</Trans>
                    </button>
                );
            default:
                return false;
        }
    }

    renderNextButton(step: any) {
        const {
            nextStep,

            invalid,

            pristine,

            install,

            ip,

            port,
        } = this.props;

        switch (step) {
            case 1:
                return (
                    <button type="button" className="btn btn-success btn-lg setup__button" onClick={nextStep}>
                        <Trans>get_started</Trans>
                    </button>
                );
            case 2:
            case 3:
                return (
                    <button
                        type="submit"
                        className="btn btn-success btn-lg setup__button"
                        disabled={invalid || pristine || install.processingSubmit}>
                        <Trans>next</Trans>
                    </button>
                );
            case 4:
                return (
                    <button type="button" className="btn btn-success btn-lg setup__button" onClick={nextStep}>
                        <Trans>next</Trans>
                    </button>
                );
            case 5:
                return (
                    <button
                        type="button"
                        className="btn btn-success btn-lg setup__button"
                        onClick={() => this.props.openDashboard(ip, port)}>
                        <Trans>open_dashboard</Trans>
                    </button>
                );
            default:
                return false;
        }
    }

    render() {
        const { install } = this.props;

        return (
            <div className="setup__nav">
                <div className="btn-list">
                    {this.renderPrevButton(install.step)}
                    {this.renderNextButton(install.step)}
                </div>
            </div>
        );
    }
}

const mapStateToProps = (state: any) => {
    const { install } = state;
    return { install };
};

export default connect(mapStateToProps, actionCreators)(Controls);
