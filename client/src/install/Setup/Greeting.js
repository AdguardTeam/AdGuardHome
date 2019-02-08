import React, { Component } from 'react';
import { Trans, withNamespaces } from 'react-i18next';
import Controls from './Controls';

class Greeting extends Component {
    render() {
        return (
            <div className="setup__step">
                <div className="setup__group">
                    <h1 className="setup__title">
                        <Trans>install_welcome_title</Trans>
                    </h1>
                    <p className="setup__desc text-center">
                        <Trans>install_welcome_desc</Trans>
                    </p>
                </div>
                <Controls />
            </div>
        );
    }
}

export default withNamespaces()(Greeting);
