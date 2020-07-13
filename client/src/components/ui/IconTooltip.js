import React from 'react';
import PropTypes from 'prop-types';

import './IconTooltip.css';
import { useTranslation } from 'react-i18next';

const IconTooltip = ({ text, type = '' }) => {
    const { t } = useTranslation();

    return <div data-tooltip={t(text)}
                className={`tooltip-custom ml-1 ${type}`} />;
};

IconTooltip.propTypes = {
    text: PropTypes.string.isRequired,
    type: PropTypes.string,
};

export default IconTooltip;
