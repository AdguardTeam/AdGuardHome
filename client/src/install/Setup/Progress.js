import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import { INSTALL_TOTAL_STEPS } from '../../helpers/constants';

const getProgressPercent = (step) => (step / INSTALL_TOTAL_STEPS) * 100;

const Progress = (props) => (
    <div className="setup__progress">
        <Trans>install_step</Trans> {props.step}/{INSTALL_TOTAL_STEPS}
        <div className="setup__progress-wrap">
            <div
                className="setup__progress-inner"
                style={{ width: `${getProgressPercent(props.step)}%` }}
            />
        </div>
    </div>
);

Progress.propTypes = {
    step: PropTypes.number.isRequired,
};

export default withTranslation()(Progress);
