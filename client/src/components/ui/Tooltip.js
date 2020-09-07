import React from 'react';
import TooltipTrigger from 'react-popper-tooltip';
import propTypes from 'prop-types';
import { useTranslation } from 'react-i18next';

import {
    HIDE_TOOLTIP_DELAY,
    MEDIUM_SCREEN_SIZE,
    SHOW_TOOLTIP_DELAY,
} from '../../helpers/constants';
import 'react-popper-tooltip/dist/styles.css';
import './Tooltip.css';

const Tooltip = ({
    children,
    content,
    triggerClass = 'tooltip-custom__trigger',
    className = 'tooltip-container',
    placement = 'bottom',
    trigger = 'hover',
    delayShow = SHOW_TOOLTIP_DELAY,
    delayHide = HIDE_TOOLTIP_DELAY,
    onVisibilityChange,
}) => {
    const { t } = useTranslation();
    const touchEventsAvailable = 'ontouchstart' in window;

    let triggerValue = trigger;
    let delayHideValue = delayHide;
    let delayShowValue = delayShow;

    if (window.matchMedia(`(max-width: ${MEDIUM_SCREEN_SIZE}px)`).matches || touchEventsAvailable) {
        triggerValue = 'click';
        delayHideValue = 0;
        delayShowValue = 0;
    }

    const renderTooltip = ({ tooltipRef, getTooltipProps }) => (
            <div
                    {...getTooltipProps({
                        ref: tooltipRef,
                        className,
                    })}
            >
                {typeof content === 'string' ? t(content) : content}
            </div>
    );

    const renderTrigger = ({ getTriggerProps, triggerRef }) => (
            <span
                    {...getTriggerProps({
                        ref: triggerRef,
                        className: triggerClass,
                    })}
            >
                    {children}
                </span>
    );

    renderTooltip.propTypes = {
        tooltipRef: propTypes.object,
        getTooltipProps: propTypes.func,
    };

    renderTrigger.propTypes = {
        triggerRef: propTypes.object,
        getTriggerProps: propTypes.func,
    };

    return (
        <TooltipTrigger
            placement={placement}
            trigger={triggerValue}
            delayHide={delayHideValue}
            delayShow={delayShowValue}
            tooltip={renderTooltip}
            onVisibilityChange={onVisibilityChange}
        >
            {renderTrigger}
        </TooltipTrigger>
    );
};

Tooltip.propTypes = {
    children: propTypes.element.isRequired,
    content: propTypes.oneOfType(
        [
            propTypes.string,
            propTypes.element,
            propTypes.arrayOf(propTypes.element),
        ],
    ).isRequired,
    placement: propTypes.string,
    trigger: propTypes.string,
    delayHide: propTypes.number,
    delayShow: propTypes.number,
    className: propTypes.string,
    triggerClass: propTypes.string,
    onVisibilityChange: propTypes.func,
};

export default Tooltip;
