import { type JSX } from 'solid-js';

import intl from 'panel/common/intl';
import { formatNumber } from 'panel/helpers/helpers';
import { Tooltip } from '../Tooltip';

import s from './QueriesTooltip.module.pcss';

const MIN_COUNT_FOR_TOOLTIP = 1000;

type Props = {
    count: number;
    children?: JSX.Element;
};

export const QueriesTooltip = (props: Props) => (
    <Tooltip
        position="top"
        disabled={props.count <= MIN_COUNT_FOR_TOOLTIP}
        overlayClass={s.queryTooltipOverlay}
        content={
            <div class={s.queryTooltip}>
                {intl.getMessage('queries_tooltip', {
                    value: formatNumber(props.count),
                })}
            </div>
        }
    >
        {props.children}
    </Tooltip>
);
