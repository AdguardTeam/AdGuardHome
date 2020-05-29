import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import classnames from 'classnames';

import './Overlay.css';

const UpdateOverlay = (props) => {
    const overlayClass = classnames({
        overlay: true,
        'overlay--visible': props.processingUpdate,
    });

    return (
        <div className={overlayClass}>
            <div className="overlay__loading"></div>
            <Trans>processing_update</Trans>
        </div>
    );
};

UpdateOverlay.propTypes = {
    processingUpdate: PropTypes.bool,
};

export default withTranslation()(UpdateOverlay);
