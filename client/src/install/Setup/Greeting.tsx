import React from 'react';
import { Trans, withTranslation } from 'react-i18next';

import Controls from './Controls';

const Greeting = () => (
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

export default withTranslation()(Greeting);
