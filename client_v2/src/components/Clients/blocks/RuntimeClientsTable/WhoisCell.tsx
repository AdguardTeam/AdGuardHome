import { createMemo, Show } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import type { WhoisInfo } from 'panel/initialState';
import s from './RuntimeClientsTable.module.pcss';

type Props = {
    whoisInfo: WhoisInfo;
    ip: string;
};

const stripHtml = (str: string) => str.replace(/<\/?span>/g, '');

export const WhoisCell = (props: Props) => {
    const raw = createMemo(() => props.whoisInfo || {});
    const country = createMemo(() => raw().country);
    const orgname = createMemo(() => raw().orgname || raw().org);
    const hasData = createMemo(() => country() || orgname());

    return (
        <Show when={hasData()} fallback={<span>-</span>}>
            <div class={s.whoisCell}>
                <div class={s.whoisInline}>
                    <Show when={country()}>
                        <span class={s.whoisRow}>
                            <Icon icon="location" color="green" class={s.whoisIcon} />
                            <span class={s.whoisText}>{country()}</span>
                        </span>
                    </Show>
                    <Show when={orgname()}>
                        <span class={s.whoisRow}>
                            <Icon icon="wifi" color="green" class={s.whoisIcon} />
                            <span class={cn(theme.common.textOverflow, s.whoisText)}>
                                {orgname()}
                            </span>
                        </span>
                    </Show>
                </div>

                <div class={s.tooltip}>
                    <div class={s.tooltipTitle}>{intl.getMessage('client_details')}</div>
                    <div class={s.tooltipRow}>
                        {stripHtml(
                            intl.getMessage('query_log_detail_address', {
                                value: props.ip,
                                span: (v: string) => v,
                            }),
                        )}
                    </div>
                    <Show when={country()}>
                        <div class={s.tooltipRow}>
                            {stripHtml(
                                intl.getMessage('query_log_detail_country', {
                                    value: country(),
                                    span: (v: string) => v,
                                }),
                            )}
                        </div>
                    </Show>
                    <Show when={orgname()}>
                        <div class={s.tooltipRow}>
                            {stripHtml(
                                intl.getMessage('query_log_detail_network', {
                                    value: orgname(),
                                    span: (v: string) => v,
                                }),
                            )}
                        </div>
                    </Show>
                </div>
            </div>
        </Show>
    );
};
