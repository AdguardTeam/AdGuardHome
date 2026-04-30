import React, { useEffect, useMemo, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';
import ct from 'countries-and-timezones';
import intl from 'panel/common/intl';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Select } from 'panel/common/controls/Select';
import { RootState } from 'panel/initialState';
import { getBlockedServices, updateBlockedServices } from 'panel/actions/services';
import theme from 'panel/lib/theme';

import { RoutePath } from 'panel/components/Routes/Paths';
import { ScheduleRow } from './ScheduleRow';
import { ScheduleModal } from './ScheduleModal';
import {
    DayKey,
    DAYS_OF_WEEK,
    ScheduleData,
    ScheduleDayData,
    getLocalTimezone,
} from './helpers';
import { getDayName } from './getDayName';
import s from './InactivitySchedule.module.pcss';

const timezones = ct.getAllTimezones();

const TIMEZONE_OPTIONS = Object.entries(timezones).map(([tz, data]) => ({
    label: `${tz} (GMT${data.utcOffsetStr})`,
    value: tz,
}));

export const InactivitySchedule = () => {
    const dispatch = useDispatch();
    const services = useSelector((state: RootState) => state.services);
    const { list, processing } = services;

    const [modalVisible, setModalVisible] = useState(false);
    const [editDay, setEditDay] = useState<DayKey | undefined>();
    const [deleteDay, setDeleteDay] = useState<DayKey | null>(null);

    useEffect(() => {
        if (!list?.ids) {
            dispatch(getBlockedServices());
        }
    }, [dispatch, list]);

    const schedule: ScheduleData | undefined = list?.schedule;
    const currentTimezone = schedule?.time_zone || getLocalTimezone();

    const timezoneValue = useMemo(() => {
        return TIMEZONE_OPTIONS.find((opt) => opt.value === currentTimezone) || {
            label: currentTimezone,
            value: currentTimezone,
        };
    }, [currentTimezone]);

    const handleTimezoneChange = (option: { value: string }) => {
        if (!option) {
            return;
        }
        const newSchedule = { ...schedule, time_zone: option.value };
        dispatch(updateBlockedServices({ ids: list?.ids || [], schedule: newSchedule }));
    };

    const handleAdd = (day: DayKey) => {
        setEditDay(day);
        setModalVisible(true);
    };

    const handleEdit = (day: DayKey) => {
        setEditDay(day);
        setModalVisible(true);
    };

    const handleDelete = (day: DayKey) => {
        setDeleteDay(day);
    };

    const confirmDelete = () => {
        if (!deleteDay) {
            return;
        }
        const newSchedule: Record<string, unknown> = { time_zone: currentTimezone };
        DAYS_OF_WEEK.forEach((d) => {
            if (d !== deleteDay && schedule?.[d]) {
                newSchedule[d] = schedule[d];
            }
        });
        dispatch(updateBlockedServices({ ids: list?.ids || [], schedule: newSchedule }));
        setDeleteDay(null);
    };

    const handleModalSave = (day: DayKey, start: number, end: number) => {
        const newSchedule: Record<string, unknown> = { time_zone: currentTimezone };
        DAYS_OF_WEEK.forEach((d) => {
            if (schedule?.[d]) {
                newSchedule[d] = schedule[d];
            }
        });
        newSchedule[day] = { start, end };
        dispatch(updateBlockedServices({ ids: list?.ids || [], schedule: newSchedule }));
        setModalVisible(false);
        setEditDay(undefined);
    };

    const handleModalClose = () => {
        setModalVisible(false);
        setEditDay(undefined);
    };

    if (processing) {
        return null;
    }

    return (
        <div className={cn(theme.layout.container, s.container)}>
            <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
            <div className={s.breadcrumbs}>
                <Breadcrumbs
                    parentLinks={[
                        { path: RoutePath.BlockedServices, title: intl.getMessage('blocked_services') },
                    ]}
                    currentTitle={intl.getMessage('inactivity_schedule')}
                />
            </div>

            <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>{intl.getMessage('inactivity_schedule')}</h1>

            <div className={s.timezoneWrapper}>
                <div className={s.timezoneLabel}>
                    {intl.getMessage('inactivity_schedule_timezone')}
                </div>
                <Select
                    options={TIMEZONE_OPTIONS}
                    value={timezoneValue}
                    onChange={handleTimezoneChange}
                    isSearchable
                />
            </div>

            <div className={s.scheduleList}>
                {DAYS_OF_WEEK.map((day) => (
                    <ScheduleRow
                        key={day}
                        day={day}
                        data={schedule?.[day] as ScheduleDayData | undefined}
                        onEdit={handleEdit}
                        onDelete={handleDelete}
                        onAdd={handleAdd}
                    />
                ))}
            </div>

            {modalVisible && (
                <ScheduleModal
                    visible={modalVisible}
                    currentDay={editDay}
                    currentData={editDay ? (schedule?.[editDay] as ScheduleDayData | undefined) : undefined}
                    onClose={handleModalClose}
                    onSave={handleModalSave}
                />
            )}

            {deleteDay && (
                <ConfirmDialog
                    title={intl.getMessage('inactivity_schedule_delete')}
                    text={intl.getMessage('inactivity_schedule_delete_desc', {
                        value: getDayName(deleteDay),
                    })}
                    buttonText={intl.getMessage('delete_btn')}
                    cancelText={intl.getMessage('cancel_btn')}
                    onConfirm={confirmDelete}
                    onClose={() => setDeleteDay(null)}
                    buttonVariant="danger"
                />
            )}
            </div>
        </div>
    );
};
