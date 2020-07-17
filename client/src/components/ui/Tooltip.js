import React from 'react';
import TooltipTrigger from 'react-popper-tooltip';
import propTypes from 'prop-types';
import { useTranslation } from 'react-i18next';
import { HIDE_TOOLTIP_DELAY } from '../../helpers/constants';
import 'react-popper-tooltip/dist/styles.css';
import './Tooltip.css';

const Tooltip = ({
    children,
    content,
    triggerClass = 'tooltip-custom__trigger',
    className = 'tooltip-container',
    placement = 'bottom',
    trigger = 'hover',
    delayHide = HIDE_TOOLTIP_DELAY,
}) => {
    const { t } = useTranslation();

    return <TooltipTrigger
        placement={placement}
        trigger={trigger}
        delayHide={delayHide}
        tooltip={({
            tooltipRef,
            getTooltipProps,
        }) => <div {...getTooltipProps({
            ref: tooltipRef,
            className,
        })}>
            {typeof content === 'string' ? t(content) : content}
        </div>
        }>{({ getTriggerProps, triggerRef }) => <span
        {...getTriggerProps({
            ref: triggerRef,
            className: triggerClass,
        })}
    >{children}</span>}
    </TooltipTrigger>;
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
    delayHide: propTypes.string,
    className: propTypes.string,
    triggerClass: propTypes.string,
};

export default Tooltip;
