import { Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';
import { getProtocolName } from 'panel/components/QueryLog/helpers';
import { QueryDetailsTooltipContent } from 'panel/components/QueryLog/blocks/LogTable/blocks/QueryDetailsTooltipContent';
import { LogEntry } from 'panel/components/QueryLog/types';

import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
};

export const RequestCell = (props: Props) => {
    return (
        <div class={s.requestCell} data-testid="query-log-request-cell">
            <div class={s.requestContent}>
                <div class={s.requestPrimary}>
                    <span
                        class={cn(s.domain, theme.text.t3)}
                        title={props.row.unicodeName || props.row.domain}
                    >
                        {props.row.unicodeName || props.row.domain}
                    </span>

                    <div class={s.requestIcons}>
                        <Dropdown
                            trigger="hover"
                            position="bottomLeft"
                            overlayClass={s.iconTooltipOverlay}
                            menu={
                                <div class={cn(theme.dropdown.menu, s.queryDetailsTooltipMenu)}>
                                    <QueryDetailsTooltipContent row={props.row} />
                                </div>
                            }
                            childrenClass={s.iconTooltipTrigger}
                            noIcon
                        >
                            <button
                                type="button"
                                class={s.queryDetailsTooltipButton}
                                aria-label={intl.getMessage('query_details')}
                                title={intl.getMessage('query_details')}
                                onClick={(event) => event.stopPropagation()}
                            >
                                <Icon
                                    icon="tracking"
                                    color={props.row.tracker ? 'green' : 'gray'}
                                    class={s.requestIcon}
                                />
                            </button>
                        </Dropdown>

                        <Show when={props.row.answer_dnssec}>
                            <Dropdown
                                trigger="hover"
                                position="bottomLeft"
                                overlayClass={s.iconTooltipOverlay}
                                menu={
                                    <div class={cn(theme.dropdown.menu, s.iconTooltipMenu)}>
                                        {intl.getMessage('validated_with_dnssec')}
                                    </div>
                                }
                                childrenClass={s.iconTooltipTrigger}
                                noIcon
                            >
                                <Icon icon="lock" color="green" class={s.requestIcon} />
                            </Dropdown>
                        </Show>
                    </div>
                </div>
                <span class={cn(s.secondaryLine, theme.text.t4)}>
                    {intl.getMessage('type_value', { value: props.row.type })},{' '}
                    {getProtocolName(props.row.client_proto)}
                </span>
            </div>
        </div>
    );
};
