import React from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';

import { DayKey, ScheduleDayData, formatTimePeriod, isFullDay } from './helpers';
import { getDayName } from './getDayName';
import s from './InactivitySchedule.module.pcss';

type Props = {
    day: DayKey;
    data: ScheduleDayData | undefined;
    onEdit: (day: DayKey) => void;
    onDelete: (day: DayKey) => void;
    onAdd: (day: DayKey) => void;
};

export const ScheduleRow = ({ day, data, onEdit, onDelete, onAdd }: Props) => {
    const dayName = getDayName(day);
    const isConfigured = !!data;

    const getTimeDisplay = () => {
        if (!data) {
            return (
                <>
                    <span className={s.noScheduleTextDesktop}>
                        {intl.getMessage('inactivity_schedule_no_schedules')}
                    </span>
                    <span className={s.noScheduleTextMobile}>–</span>
                </>
            );
        }

        if (isFullDay(data.start, data.end)) {
            return intl.getMessage('inactivity_schedule_all_day');
        }

        return formatTimePeriod(data.start, data.end);
    };

    return (
        <div className={s.scheduleRow}>
            <div className={s.dayName}>{dayName}</div>
            <div className={s.timeDisplay}>{getTimeDisplay()}</div>
            {isConfigured ? (
                <>
                    <div className={cn(s.actions, s.actionsDesktop)}>
                        <button
                            type="button"
                            className={s.actionButton}
                            onClick={() => onEdit(day)}
                            aria-label={intl.getMessage('inactivity_schedule_edit')}
                        >
                            <Icon icon="edit" />
                        </button>
                    </div>
                    <div className={cn(s.actions, s.actionsDesktop)}>
                        <button
                            type="button"
                            className={cn(s.actionButton, s.actionButtonDelete)}
                            onClick={() => onDelete(day)}
                            aria-label={intl.getMessage('delete_table_action')}
                        >
                            <Icon icon="delete" />
                        </button>
                    </div>
                    <div className={cn(s.actions, s.actionsMobile)}>
                        <Dropdown
                            trigger="click"
                            position="bottomRight"
                            noIcon
                            menu={
                                <div className={theme.dropdown.menu}>
                                    <button
                                        type="button"
                                        className={theme.dropdown.item}
                                        onClick={() => onEdit(day)}
                                    >
                                        {intl.getMessage('inactivity_schedule_edit')}
                                    </button>
                                    <button
                                        type="button"
                                        className={cn(theme.dropdown.item, s.dropdownItemDanger)}
                                        onClick={() => onDelete(day)}
                                    >
                                        {intl.getMessage('inactivity_schedule_delete')}
                                    </button>
                                </div>
                            }
                        >
                            <div className={s.actionButton}>
                                <Icon icon="bullets" />
                            </div>
                        </Dropdown>
                    </div>
                </>
            ) : (
                <>
                    <div className={s.actions}>
                        <button
                            type="button"
                            className={cn(s.actionButton, s.actionButtonAdd)}
                            onClick={() => onAdd(day)}
                            aria-label={intl.getMessage('inactivity_schedule_add')}
                        >
                            <Icon icon="plus" />
                        </button>
                    </div>
                    <div className={cn(s.actions, s.actionsDesktop)} />
                </>
            )}
        </div>
    );
};
