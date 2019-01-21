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

    renderButtons(step) {
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
                    <div className="btn-list">
                        <button
                            type="button"
                            className="btn btn-secondary btn-standard btn-lg"
                            onClick={this.props.prevStep}
                        >
                            <Trans>back</Trans>
                        </button>
                        <button
                            type="submit"
                            className="btn btn-success btn-standard btn-lg"
                            disabled={this.props.invalid || this.props.pristine}
                        >
                            <Trans>next</Trans>
                        </button>
                    </div>
                );
            case 4:
                return (
                    <div className="btn-list">
                        <button
                            type="button"
                            className="btn btn-secondary btn-standard btn-lg"
                            onClick={this.props.prevStep}
                        >
                            <Trans>back</Trans>
                        </button>
                        <button
                            type="button"
                            className="btn btn-success btn-standard btn-lg"
                            onClick={this.props.nextStep}
                        >
                            <Trans>next</Trans>
                        </button>
                    </div>
                );
            case 5:
                return (
                    <button
                        type="submit"
                        className="btn btn-success btn-standard btn-lg"
                        disabled={this.props.submitting || this.props.pristine}
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
                {this.renderButtons(this.props.step)}
            </div>
        );
    }
}

Controls.propTypes = {
    step: PropTypes.number.isRequired,
    nextStep: PropTypes.func,
    prevStep: PropTypes.func,
    pristine: PropTypes.bool,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
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
