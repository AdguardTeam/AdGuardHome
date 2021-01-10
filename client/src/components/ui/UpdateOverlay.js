import React from 'react';
import { Trans } from 'react-i18next';
import classnames from 'classnames';
import { useSelector } from 'react-redux';
import './Overlay.css';

const UpdateOverlay = () => {
    const processingUpdate = useSelector((state) => state.dashboard.processingUpdate);
    const overlayClass = classnames('overlay', {
        'overlay--visible': processingUpdate,
    });

    return (
        <div className={overlayClass}>
            <div className="overlay__loading"></div>
            <Trans>processing_update</Trans>
        </div>
    );
};

export default UpdateOverlay;
