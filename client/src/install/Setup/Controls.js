import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';

import * as actionCreators from '../../actions/install';
import { INSTALL_FIRST_STEP, INSTALL_TOTAL_STEPS } from '../../helpers/constants';

class Controls extends Component {
    nextStep = () => {
        if (this.props.step < INSTALL_TOTAL_STEPS) {
            this.props.nextStep();
        }
    }

    prevStep = () => {
        if (this.props.step > INSTALL_FIRST_STEP) {
            this.props.prevStep();
        }
    }

    renderPrevButton(step) {
        switch (step) {
            case 2:
            case 3:
            case 4:
                return (
                    <button
                            type="button"
                            className="btn btn-secondary btn-standard btn-lg"
                            onClick={this.props.prevStep}
                        >
                            <Trans>back</Trans>
                        </button>
                );
            default:
                return false;
        }
    }

    renderNextButton(step) {
        switch (step) {
            case 1:
                return (
                    <button
                        type="button"
                        className="btn btn-success btn-standard btn-lg"
                        onClick={this.props.nextStep}
                    >
                        <Trans>get_started</Trans>
                    </button>
                );
            case 2:
            case 3:
                return (
                    <button
                        type="submit"
                        className="btn btn-success btn-standard btn-lg"
                        disabled={this.props.invalid || this.props.pristine}
                    >
                        <Trans>next</Trans>
                    </button>
                );
            case 4:
                return (
                    <button
                        type="button"
                        className="btn btn-success btn-standard btn-lg"
                        onClick={this.props.nextStep}
                    >
                        <Trans>next</Trans>
                    </button>
                );
            case 5:
                return (
                    <button
                        type="button"
                        className="btn btn-success btn-standard btn-lg"
                        onClick={this.props.openDashboard}
                    >
                        <Trans>open_dashboard</Trans>
                    </button>
                );
            default:
                return false;
        }
    }

    render() {
        return (
            <div className="setup__nav">
                <div className="btn-list">
                    {this.renderPrevButton(this.props.step)}
                    {this.renderNextButton(this.props.step)}
                </div>
            </div>
        );
    }
}

Controls.propTypes = {
    step: PropTypes.number.isRequired,
    nextStep: PropTypes.func,
    prevStep: PropTypes.func,
    openDashboard: PropTypes.func,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
    pristine: PropTypes.bool,
};

const mapStateToProps = (state) => {
    const { step } = state.install;
    const props = { step };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(Controls);
