import { Show } from 'solid-js';
import cn from 'clsx';

import { Tooltip } from 'panel/common/ui/Tooltip';
import { useMediaQuery } from 'panel/helpers/useMediaQuery';
import { useIsTruncated } from 'panel/hooks/useIsTruncated';
import theme from 'panel/lib/theme';

import s from './HeaderLabel.module.pcss';

export type HeaderLabelProps = {
    columnKey: string;
    text: string;
    tooltip?: boolean;
};

export const HeaderLabel = (props: HeaderLabelProps) => {
    let labelRef: HTMLSpanElement | undefined;

    const isTouch = useMediaQuery('(hover: none)');
    const isTruncated = useIsTruncated(() => labelRef);

    const useTooltip = () => !!props.tooltip && !isTouch();

    const label = (
        <span
            ref={labelRef}
            data-testid={`table-header-${props.columnKey}`}
            class={cn(theme.text.t3, theme.text.condenced, theme.text.semibold, s.text)}
        >
            {props.text}
        </span>
    );

    return (
        <Show when={props.tooltip} fallback={label}>
            <Tooltip
                content={<div class={cn(theme.text.t3, s.tooltipContent)}>{props.text}</div>}
                position="bottomLeft"
                disabled={!useTooltip() || !isTruncated()}
                class={s.labelWrapper}
            >
                {label}
            </Tooltip>
        </Show>
    );
};
