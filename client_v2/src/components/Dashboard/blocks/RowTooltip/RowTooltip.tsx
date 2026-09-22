import { type JSX } from 'solid-js';
import cn from 'clsx';

import { Tooltip } from 'panel/common/ui/Tooltip';
import { useIsTouchDevice } from 'panel/hooks/useMediaQuery';

import s from './RowTooltip.module.pcss';

type Props = {
    content: JSX.Element;
    class?: string;
    /** Overrides the overlay, e.g. to make a non-interactive card transparent to the pointer. */
    overlayClass?: string;
    children: JSX.Element;
};

/**
 * Makes a whole Dashboard row a tooltip trigger.
 *
 * Hover-only: on touch devices the tooltip is disabled entirely, so a tap
 * follows the row link instead of revealing the details.
 */
export const RowTooltip = (props: Props) => {
    const isTouch = useIsTouchDevice();

    return (
        <Tooltip
            content={props.content}
            position="bottomLeft"
            disabled={isTouch()}
            overlayClass={props.overlayClass}
            class={cn(s.rowTrigger, props.class)}
        >
            {props.children}
        </Tooltip>
    );
};
