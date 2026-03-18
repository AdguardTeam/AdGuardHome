import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { captitalizeWords } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';
import { getProtocolName } from 'panel/components/QueryLog/helpers';
import { LogEntry } from 'panel/components/QueryLog/types';

import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
};

export const RequestCell = ({ row }: Props) => {
    const dnssecTooltip = intl.getMessage('validated_with_dnssec');
    const trackerSource = row.tracker?.sourceData;

    const trackerTooltip = row.tracker ? (
        <div className={s.iconTooltipContent}>
            <div className={cn(s.iconTooltipTitle, theme.text.t2, theme.text.semibold)}>
                {intl.getMessage('known_tracker')}
            </div>
            <div className={s.iconTooltipGrid}>
                <span className={cn(s.iconTooltipLabel, theme.text.t3)}>{intl.getMessage('name')}</span>
                <span className={cn(s.iconTooltipValue, theme.text.t3)}>{row.tracker.name}</span>

                <span className={cn(s.iconTooltipLabel, theme.text.t3)}>{intl.getMessage('category_label')}</span>
                <span className={cn(s.iconTooltipValue, theme.text.t3)}>{captitalizeWords(row.tracker.category)}</span>

                {trackerSource?.name && (
                    <>
                        <span className={cn(s.iconTooltipLabel, theme.text.t3)}>{intl.getMessage('source_label')}</span>
                        <span className={cn(s.iconTooltipValue, theme.text.t3)}>
                            {trackerSource.url ? (
                                <a
                                    href={trackerSource.url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className={s.iconTooltipLink}
                                >
                                    {trackerSource.name}
                                </a>
                            ) : (
                                trackerSource.name
                            )}
                        </span>
                    </>
                )}
            </div>
        </div>
    ) : null;

    const renderDnsSec = () =>
        row.answer_dnssec ? (
            <Dropdown
                trigger="hover"
                position="bottomLeft"
                overlayClassName={s.iconTooltipOverlay}
                menu={<div className={cn(theme.dropdown.menu, s.iconTooltipMenu)}>{dnssecTooltip}</div>}
                childrenClassName={s.iconTooltipTrigger}
                noIcon
            >
                <Icon icon="lock" color="green" className={s.requestIcon} />
            </Dropdown>
        ) : (
            <Icon icon="lock" color="gray" className={s.requestIcon} />
        );

    return (
        <div className={s.requestCell}>
            <div className={s.requestIcons}>
                {renderDnsSec()}

                <Dropdown
                    trigger="hover"
                    position="bottomLeft"
                    overlayClassName={s.iconTooltipOverlay}
                    menu={<div className={cn(theme.dropdown.menu, s.iconTooltipMenu)}>{trackerTooltip}</div>}
                    disabled={!row.tracker}
                    childrenClassName={s.iconTooltipTrigger}
                    noIcon
                >
                    <Icon icon="tracking" color={row.tracker ? 'green' : 'gray'} className={s.requestIcon} />
                </Dropdown>
            </div>

            <div className={s.requestContent}>
                <div className={s.requestPrimary}>
                    <span className={cn(s.domain, theme.text.t3)} title={row.unicodeName || row.domain}>
                        {row.unicodeName || row.domain}
                    </span>
                </div>
                <span className={cn(s.secondaryLine, s.requestDetails, theme.text.t3)}>
                    {intl.getMessage('type_value', { value: row.type })}, {getProtocolName(row.client_proto)}
                </span>
            </div>
        </div>
    );
};
