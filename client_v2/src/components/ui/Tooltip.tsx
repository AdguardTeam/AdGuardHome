import React from 'react';
import PopperJS from 'popper.js';
import TooltipTrigger, { TriggerTypes } from 'react-popper-tooltip';
import { useTranslation } from 'react-i18next';

import { HIDE_TOOLTIP_DELAY, MEDIUM_SCREEN_SIZE, SHOW_TOOLTIP_DELAY } from '../../helpers/constants';
import 'react-popper-tooltip/dist/styles.css';
import './Tooltip.css';

interface TooltipProps {
    children: React.ReactElement;
    content: string | React.ReactElement | React.ReactElement[];
    placement?: PopperJS.Placement;
    trigger?: TriggerTypes;
    delayHide?: number;
    delayShow?: number;
    className?: string;
    triggerClass?: string;
    onVisibilityChange?: (...args: unknown[]) => unknown;
    defaultTooltipShown?: boolean;
}

interface renderTooltipProps {
    tooltipRef?: object;
    getTooltipProps?: (...args: unknown[]) => Record<any, any>;
}

interface renderTriggerProps {
    triggerRef?: object;
    getTriggerProps?: (...args: unknown[]) => Record<any, any>;
}

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
    defaultTooltipShown,
}: TooltipProps) => {
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

    const renderTooltip = ({ tooltipRef, getTooltipProps }: renderTooltipProps) => (
        <div
            {...getTooltipProps({
                ref: tooltipRef,
                className,
            })}>
            {typeof content === 'string' ? t(content) : content}
        </div>
    );

    const renderTrigger = ({ getTriggerProps, triggerRef }: renderTriggerProps) => (
        <span
            {...getTriggerProps({
                ref: triggerRef,
                className: triggerClass,
            })}>
            {children}
        </span>
    );

    return (
        <TooltipTrigger
            placement={placement}
            trigger={triggerValue}
            delayHide={delayHideValue}
            delayShow={delayShowValue}
            tooltip={renderTooltip}
            onVisibilityChange={onVisibilityChange}
            defaultTooltipShown={defaultTooltipShown}>
            {renderTrigger}
        </TooltipTrigger>
    );
};

export default Tooltip;
