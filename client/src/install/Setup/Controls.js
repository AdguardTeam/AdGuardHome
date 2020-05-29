import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';

import * as actionCreators from '../../actions/install';

class Controls extends Component {
    renderPrevButton(step) {
        switch (step) {
            case 2:
            case 3:
                return (
                    <button
                            type="button"
                            className="btn btn-secondary btn-lg setup__button"
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
                    <button
                        type="button"
                        className="btn btn-success btn-lg setup__button"
                        onClick={nextStep}
                    >
                        <Trans>get_started</Trans>
                    </button>
                );
            case 2:
            case 3:
                return (
                    <button
                        type="submit"
                        className="btn btn-success btn-lg setup__button"
                        disabled={
                            invalid
                            || pristine
                            || install.processingSubmit
                            || install.dns.status
                            || install.web.status
                        }
                    >
                        <Trans>next</Trans>
                    </button>
                );
            case 4:
                return (
                    <button
                        type="button"
                        className="btn btn-success btn-lg setup__button"
                        onClick={nextStep}
                    >
                        <Trans>next</Trans>
                    </button>
                );
            case 5:
                return (
                    <button
                        type="button"
                        className="btn btn-success btn-lg setup__button"
                        onClick={() => this.props.openDashboard(ip, port)}
                    >
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

Controls.propTypes = {
    install: PropTypes.object.isRequired,
    nextStep: PropTypes.func,
    prevStep: PropTypes.func,
    openDashboard: PropTypes.func,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
    pristine: PropTypes.bool,
    ip: PropTypes.string,
    port: PropTypes.number,
};

const mapStateToProps = (state) => {
    const { install } = state;
    const props = { install };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(Controls);
