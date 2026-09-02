import { Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import type { WhoisInfo } from 'panel/initialState';

import s from './ClientTooltip.module.pcss';

type Props = {
    address: string;
    whoisInfo?: WhoisInfo;
    blocked: boolean;
};

const renderValue = (content: string) => <span class={s.tooltipValue}>{content}</span>;

const renderStatusValue = (content: string) => (
    <span class={cn(s.tooltipValue, s.tooltipStatusValue)}>{content}</span>
);

export const ClientTooltip = (props: Props) => {
    const whois = () => props.whoisInfo || {};
    const country = () => whois().country;
    const network = () => whois().orgname || whois().org;

    return (
        <div class={s.tooltip}>
            <div class={cn(theme.text.t2, theme.text.semibold, s.tooltipTitle)}>
                {intl.getMessage('client_details')}
            </div>

            <div class={cn(theme.text.t3, s.tooltipRow)}>
                {intl.getMessage('query_log_detail_address', {
                    value: props.address,
                    span: renderValue,
                })}
            </div>

            <Show when={country()}>
                {(value) => (
                    <div class={cn(theme.text.t3, s.tooltipRow)}>
                        {intl.getMessage('query_log_detail_country', {
                            value: value(),
                            span: renderValue,
                        })}
                    </div>
                )}
            </Show>

            <Show when={network()}>
                {(value) => (
                    <div class={cn(theme.text.t3, s.tooltipRow)}>
                        {intl.getMessage('query_log_detail_network', {
                            value: value(),
                            span: renderValue,
                        })}
                    </div>
                )}
            </Show>

            <Show when={props.blocked}>
                <div class={cn(theme.text.t3, s.tooltipRow)}>
                    {intl.getMessage('query_log_detail_status', {
                        value: intl.getMessage('blocked'),
                        span: renderStatusValue,
                    })}
                </div>
            </Show>
        </div>
    );
};
