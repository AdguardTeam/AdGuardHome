import React, { useEffect, useState } from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog';
import { Button } from 'panel/common/ui/Button';
import { Select } from 'panel/common/controls/Select';
import { Checkbox } from 'panel/common/controls/Checkbox';
import theme from 'panel/lib/theme';

import { DayKey, FULL_DAY_END_MS, ScheduleDayData, msToTime, timeToMs, padTime } from './helpers';
import { getDayName } from './getDayName';
import s from './ScheduleModal.module.pcss';

type Props = {
    visible: boolean;
    currentDay?: DayKey;
    currentData?: ScheduleDayData;
    onClose: () => void;
    onSave: (day: DayKey, start: number, end: number) => void;
};

const HOURS_OPTIONS = Array.from({ length: 24 }, (_, i) => ({
    label: padTime(i),
    value: i,
}));

const MINUTES_OPTIONS = Array.from({ length: 60 }, (_, i) => ({
    label: padTime(i),
    value: i,
}));

export const ScheduleModal = ({ visible, currentDay, currentData, onClose, onSave }: Props) => {
    const [startHour, setStartHour] = useState(0);
    const [startMinute, setStartMinute] = useState(0);
    const [endHour, setEndHour] = useState(23);
    const [endMinute, setEndMinute] = useState(59);
    const [allDay, setAllDay] = useState(false);

    useEffect(() => {
        if (currentData) {
            const isFullDayData = currentData.start === 0 && currentData.end === FULL_DAY_END_MS;
            setAllDay(isFullDayData);
            const start = msToTime(currentData.start);
            const end = msToTime(currentData.end);
            setStartHour(start.hours);
            setStartMinute(start.minutes);
            setEndHour(end.hours);
            setEndMinute(end.minutes);
        }
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

    const startMs = timeToMs(startHour, startMinute);
    const endMs = timeToMs(endHour, endMinute);
    const isInvalidPeriod = !allDay && startMs >= endMs;
    const isDisabled = !currentDay || isInvalidPeriod;

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

                <div className={s.allDayRow}>
                    <Checkbox
                        id="all-day-checkbox"
                        checked={allDay}
                        onChange={handleAllDayChange}
                    >
                        {intl.getMessage('inactivity_schedule_all_day')}
                    </Checkbox>
                </div>

                <div className={s.timeWrapper}>
                    <div className={s.timeField}>
                        <span className={s.timeLabel}>
                            {intl.getMessage('inactivity_schedule_time_from')}
                        </span>
                        <div className={s.selects}>
                            <Select
                                options={HOURS_OPTIONS}
                                value={HOURS_OPTIONS[startHour]}
                                onChange={(opt) => setStartHour(opt.value)}
                                isDisabled={allDay}
                            />
                            <Select
                                options={MINUTES_OPTIONS}
                                value={MINUTES_OPTIONS[startMinute]}
                                onChange={(opt) => setStartMinute(opt.value)}
                                isDisabled={allDay}
                            />
                        </div>
                    </div>
                    <div className={s.timeField}>
                        <span className={s.timeLabel}>
                            {intl.getMessage('inactivity_schedule_time_to')}
                        </span>
                        <div className={s.selects}>
                            <Select
                                options={HOURS_OPTIONS}
                                value={HOURS_OPTIONS[endHour]}
                                onChange={(opt) => setEndHour(opt.value)}
                                isDisabled={allDay}
                            />
                            <Select
                                options={MINUTES_OPTIONS}
                                value={MINUTES_OPTIONS[endMinute]}
                                onChange={(opt) => setEndMinute(opt.value)}
                                isDisabled={allDay}
                            />
                        </div>
                    </div>
                </div>

                {isInvalidPeriod && (
                    <div className={s.error}>
                        {intl.getMessage('schedule_invalid_select')}
                    </div>
                )}

                <div className={s.notice}>
                    {intl.getMessage('inactivity_schedule_replace_desc')}
                </div>
            </div>

            <div className={cn(theme.dialog.footer, s.footer)}>
                <Button
                    variant="primary"
                    size="small"
                    onClick={handleSave}
                    disabled={isDisabled}
                >
                    {intl.getMessage('save_btn')}
                </Button>
                <Button
                    variant="secondary"
                    size="small"
                    onClick={onClose}
                >
                    {intl.getMessage('cancel_btn')}
                </Button>
            </div>
        </Dialog>
    );
};
