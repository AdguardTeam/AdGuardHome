import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { reduxForm } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';

class Submit extends Component {
    render() {
        const {
            handleSubmit,
            pristine,
            submitting,
        } = this.props;

        return (
            <div className="setup__step">
                <div className="setup__group">
                    <h1 className="setup__title">
                        <Trans>install_submit_title</Trans>
                    </h1>
                    <p className="setup__desc">
                        <Trans>install_submit_desc</Trans>
                    </p>
                </div>
                <form onSubmit={handleSubmit}>
                    <Controls submitting={submitting} pristine={pristine} />
                </form>
            </div>
        );
    }
}

Submit.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    pristine: PropTypes.bool.isRequired,
    submitting: PropTypes.bool.isRequired,
};

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'install',
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
    }),
])(Submit);
