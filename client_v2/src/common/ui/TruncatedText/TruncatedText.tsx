import { createSignal } from 'solid-js';
import cn from 'clsx';

import { Tooltip } from 'panel/common/ui/Tooltip';
import { useIsTouchDevice } from 'panel/hooks/useMediaQuery';
import { useIsTruncated } from 'panel/hooks/useIsTruncated';
import theme from 'panel/lib/theme';

import s from './TruncatedText.module.pcss';

type Props = {
    text: string;
    /** Typography and width limits for the text element. */
    class?: string;
    testId?: string;
};

/**
 * A single-line text that shows its full value in a tooltip once the browser
 * clips it with an ellipsis.
 *
 * Replaces the native `title` attribute, which cannot be styled or delayed and
 * renders a browser tooltip on top of the app's own overlays.  Touch devices
 * get no tooltip: there is no hover to open it with.
 */
export const TruncatedText = (props: Props) => {
    const [textRef, setTextRef] = createSignal<HTMLSpanElement>();

    const isTouch = useIsTouchDevice();
    const isTruncated = useIsTruncated(textRef, {
        enabled: () => !isTouch(),
        source: () => props.text,
    });

    return (
        <Tooltip
            content={<div class={cn(theme.text.t3, s.tooltipContent)}>{props.text}</div>}
            position="bottomLeft"
            disabled={isTouch() || !isTruncated()}
            class={s.wrapper}
        >
            <span
                ref={setTextRef}
                data-testid={props.testId}
                class={cn(theme.common.textOverflow, s.text, props.class)}
            >
                {props.text}
            </span>
        </Tooltip>
    );
};
