import React, { useEffect, useState } from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog';
import { Button } from 'panel/common/ui/Button';
import { Select } from 'panel/common/controls/Select';
import { Checkbox } from 'panel/common/controls/Checkbox';
import theme from 'panel/lib/theme';

import {
    DayKey,
    FULL_DAY_END_MS,
    HOURS_OPTIONS,
    MINUTES_OPTIONS,
    ScheduleDayData,
    getEndTimeOptions,
    getNormalizedEndTime,
    isFullDay,
    msToTime,
    timeToMs,
} from './helpers';
import { getDayName } from './getDayName';
import s from './ScheduleModal.module.pcss';

type Props = {
    visible: boolean;
    currentDay?: DayKey;
    currentData?: ScheduleDayData;
    onClose: () => void;
    onSave: (day: DayKey, start: number, end: number) => void;
};

export const ScheduleModal = ({ visible, currentDay, currentData, onClose, onSave }: Props) => {
    const [startHour, setStartHour] = useState(0);
    const [startMinute, setStartMinute] = useState(0);
    const [endHour, setEndHour] = useState(23);
    const [endMinute, setEndMinute] = useState(59);
    const [allDay, setAllDay] = useState(false);

    useEffect(() => {
        if (!currentData) {
            setAllDay(false);
            setStartHour(0);
            setStartMinute(0);
            setEndHour(23);
            setEndMinute(59);
            return;
        }

        const isFullDayData = isFullDay(currentData.start, currentData.end);
        const start = msToTime(currentData.start);
        const end = msToTime(currentData.end);
        const normalizedEnd = getNormalizedEndTime(start, end) ?? end;

        setAllDay(isFullDayData);
        setStartHour(start.hours);
        setStartMinute(start.minutes);
        setEndHour(normalizedEnd.hours);
        setEndMinute(normalizedEnd.minutes);
    }, [currentData]);

    const handleAllDayChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { checked } = e.target;
        setAllDay(checked);
        if (checked) {
            setStartHour(0);
            setStartMinute(0);
            setEndHour(23);
            setEndMinute(59);
        }
    };

    const applyStartTime = (nextStartHour: number, nextStartMinute: number) => {
        setStartHour(nextStartHour);
        setStartMinute(nextStartMinute);

        if (allDay) {
            return;
        }

        const normalizedEnd = getNormalizedEndTime(
            { hours: nextStartHour, minutes: nextStartMinute },
            { hours: endHour, minutes: endMinute },
        );

        if (!normalizedEnd) {
            return;
        }

        setEndHour(normalizedEnd.hours);
        setEndMinute(normalizedEnd.minutes);
    };

    const applyEndTime = (nextEndHour: number, nextEndMinute: number) => {
        const normalizedEnd = getNormalizedEndTime(
            { hours: startHour, minutes: startMinute },
            { hours: nextEndHour, minutes: nextEndMinute },
        );

        if (!normalizedEnd) {
            return;
        }

        setEndHour(normalizedEnd.hours);
        setEndMinute(normalizedEnd.minutes);
    };

    const { hours: endHoursOptions, minutes: endMinutesOptions, hasAvailableEndTime } = getEndTimeOptions(
        { hours: startHour, minutes: startMinute },
        endHour,
    );

    const startMs = timeToMs(startHour, startMinute);
    const endMs = timeToMs(endHour, endMinute);
    const isDisabled = !currentDay || (!allDay && !hasAvailableEndTime);

    const handleSave = () => {
        if (isDisabled || !currentDay) return;
        const saveStart = allDay ? 0 : startMs;
        const saveEnd = allDay ? FULL_DAY_END_MS : endMs;
        onSave(currentDay, saveStart, saveEnd);
    };

    const dayLabel = getDayName(currentDay);
    const title = currentData
        ? intl.getMessage('inactivity_schedule_edit')
        : intl.getMessage('inactivity_schedule_add');

    return (
        <Dialog visible={visible} mask onClose={onClose} title={title} wrapClassName="rc-dialog-update">
            <div className={cn(theme.dialog.body, s.content)}>
                <div className={s.dayLabel}>{dayLabel}</div>

                <div className={s.timeWrapper}>
                    <div className={s.timeField}>
                        <span className={s.timeLabel}>
                            {intl.getMessage('inactivity_schedule_time_from')}
                        </span>
                        <div className={s.selects}>
                            <Select
                                options={HOURS_OPTIONS}
                                value={HOURS_OPTIONS[startHour]}
                                onChange={(opt) => applyStartTime(opt.value, startMinute)}
                                isDisabled={allDay}
                                height="big"
                                size="responsive"
                            />
                            <Select
                                options={MINUTES_OPTIONS}
                                value={MINUTES_OPTIONS[startMinute]}
                                onChange={(opt) => applyStartTime(startHour, opt.value)}
                                isDisabled={allDay}
                                height="big"
                                size="responsive"
                            />
                        </div>
                    </div>
                    <div className={s.timeField}>
                        <span className={s.timeLabel}>
                            {intl.getMessage('inactivity_schedule_time_to')}
                        </span>
                        <div className={s.selects}>
                            <Select
                                options={endHoursOptions}
                                value={HOURS_OPTIONS[endHour]}
                                onChange={(opt) => applyEndTime(opt.value, endMinute)}
                                isDisabled={allDay}
                                height="big"
                                size="responsive"
                            />
                            <Select
                                options={endMinutesOptions}
                                value={MINUTES_OPTIONS[endMinute]}
                                onChange={(opt) => applyEndTime(endHour, opt.value)}
                                isDisabled={allDay}
                                height="big"
                                size="responsive"
                            />
                        </div>
                    </div>
                </div>

                <div className={s.allDayRow}>
                    <Checkbox
                        id="all-day-checkbox"
                        checked={allDay}
                        onChange={handleAllDayChange}
                    >
                        {intl.getMessage('inactivity_schedule_all_day')}
                    </Checkbox>
                </div>

                <div className={cn(s.notice, theme.text.t2, theme.text.t1_tablet)}>
                    {intl.getMessage('inactivity_schedule_replace_desc')}
                </div>
            </div>

            <div className={theme.dialog.footer}>
                <Button
                    variant="primary"
                    size="small"
                    onClick={handleSave}
                    disabled={isDisabled}
                    className={theme.dialog.button}
                >
                    {intl.getMessage('save_btn')}
                </Button>
                <Button
                    variant="secondary"
                    size="small"
                    onClick={onClose}
                    className={theme.dialog.button}
                >
                    {intl.getMessage('cancel_btn')}
                </Button>
            </div>
        </Dialog>
    );
};
