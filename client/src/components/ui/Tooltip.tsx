import React from 'react';
import type { Placement } from '@popperjs/core';
import { usePopperTooltip, TriggerType } from 'react-popper-tooltip';
import { useTranslation } from 'react-i18next';

import { HIDE_TOOLTIP_DELAY, MEDIUM_SCREEN_SIZE, SHOW_TOOLTIP_DELAY } from '@/helpers/constants';
import 'react-popper-tooltip/dist/styles.css';
import './Tooltip.css';

interface TooltipProps {
    children: React.ReactElement;
    content: string | React.ReactElement | React.ReactElement[];
    placement?: Placement;
    trigger?: TriggerType;
    delayHide?: number;
    delayShow?: number;
    className?: string;
    triggerClass?: string;
    onVisibilityChange?: (visible: boolean) => void;
    defaultTooltipShown?: boolean;
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

    const { getTooltipProps, setTooltipRef, setTriggerRef, visible } = usePopperTooltip({
        placement,
        trigger: triggerValue,
        delayHide: delayHideValue,
        delayShow: delayShowValue,
        onVisibleChange: onVisibilityChange,
        defaultVisible: defaultTooltipShown,
    });

    return (
        <>
            <span ref={setTriggerRef} className={triggerClass}>
                {children}
            </span>

            {visible && (
                <div ref={setTooltipRef} {...getTooltipProps({ className })}>
                    {typeof content === 'string' ? t(content) : content}
                </div>
            )}
        </>
    );
};

export default Tooltip;
