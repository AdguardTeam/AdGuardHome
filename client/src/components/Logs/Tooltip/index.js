import React from 'react';
import PropTypes from 'prop-types';
import ReactTooltip from 'react-tooltip';
import classNames from 'classnames';
import './ReactTooltip.css';
import { touchMediaQuery } from '../../../helpers/constants';

const Tooltip = ({
    id, children, className = '', place = 'right', trigger = 'hover', overridePosition, scrollHide = true,
}) => {
    const tooltipClassName = classNames('custom-tooltip', className);

    return (
        <ReactTooltip
            id={id}
            aria-haspopup="true"
            effect="solid"
            place={place}
            className={tooltipClassName}
            backgroundColor="#fff"
            arrowColor="transparent"
            textColor="#4d4d4d"
            delayHide={300}
            scrollHide={window.matchMedia(touchMediaQuery).matches ? false : scrollHide}
            trigger={trigger}
            overridePosition={overridePosition}
            globalEventOff="click touchend"
            clickable
        >
            {children}
        </ReactTooltip>
    );
};

Tooltip.propTypes = {
    id: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
    className: PropTypes.string,
    place: PropTypes.string,
    overridePosition: PropTypes.func,
    scrollHide: PropTypes.bool,
    trigger: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.arrayOf(PropTypes.string),
    ]),
};

export default Tooltip;
